package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// Config defines the configuration for the Keycloak OIDC token verifier.
type Config struct {
	JWKSURL    string
	IssuerURL  string
	ClientID   string
	HTTPClient *http.Client
}

type jwksKey struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

type jwtClaimsPayload struct {
	Sub               string                 `json:"sub"`
	PreferredUsername string                 `json:"preferred_username"`
	Email             string                 `json:"email"`
	Exp               int64                  `json:"exp"`
	Iat               int64                  `json:"iat"`
	Nbf               int64                  `json:"nbf"`
	Iss               string                 `json:"iss"`
	Aud               interface{}            `json:"aud"`
	RealmAccess       realmAccessPayload     `json:"realm_access"`
	ResourceAccess    map[string]realmAccess `json:"resource_access"`
	Groups            []string               `json:"groups"`
}

type realmAccessPayload struct {
	Roles []string `json:"roles"`
}

type realmAccess struct {
	Roles []string `json:"roles"`
}

// KeycloakTokenVerifier verifies incoming OIDC JWT tokens against Keycloak JWKS public keys.
type KeycloakTokenVerifier struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastFetch  time.Time
}

// NewKeycloakTokenVerifier constructs a new KeycloakTokenVerifier instance.
func NewKeycloakTokenVerifier(cfg Config) *KeycloakTokenVerifier {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &KeycloakTokenVerifier{
		cfg:        cfg,
		httpClient: client,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// VerifyToken validates an encoded JWT string and returns parsed domain Claims.
func (v *KeycloakTokenVerifier) VerifyToken(ctx context.Context, tokenString string) (*model.Claims, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "KeycloakTokenVerifier.VerifyToken").
		Logger()
	log.Debug().Msg("starting jwt verification")

	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		err := fmt.Errorf("%w: empty token", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: empty token")
		return nil, err
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		err := fmt.Errorf("%w: malformed jwt structure", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: invalid token parts")
		return nil, err
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		err := fmt.Errorf("%w: failed decoding jwt header", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: invalid header encoding")
		return nil, err
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		err := fmt.Errorf("%w: failed unmarshaling jwt header", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: invalid header json")
		return nil, err
	}

	if header.Alg != "RS256" {
		err := fmt.Errorf("%w: unsupported jwt alg %s", model.ErrUnauthorized, header.Alg)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: unsupported algorithm")
		return nil, err
	}

	pubKey, err := v.getKey(ctx, header.Kid)
	if err != nil {
		mapped := fmt.Errorf("%w: unable to find public key %s: %v", model.ErrUnauthorized, header.Kid, err)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: public key resolution error")
		return nil, mapped
	}

	signingInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		err := fmt.Errorf("%w: failed decoding signature", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: invalid signature encoding")
		return nil, err
	}

	hash := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes); err != nil {
		err := fmt.Errorf("%w: invalid signature", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: signature mismatch")
		return nil, err
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		err := fmt.Errorf("%w: failed decoding claims payload", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: invalid payload encoding")
		return nil, err
	}

	var payload jwtClaimsPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		err := fmt.Errorf("%w: failed unmarshaling claims payload", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: invalid payload json")
		return nil, err
	}

	now := time.Now().Unix()
	if payload.Exp > 0 && now >= payload.Exp {
		err := fmt.Errorf("%w: token expired", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: token expired")
		return nil, err
	}
	if payload.Nbf > 0 && now < payload.Nbf {
		err := fmt.Errorf("%w: token not active yet", model.ErrUnauthorized)
		log.Warn().Dur("duration_ms", time.Since(start)).Msg("jwt verification failed: token not active")
		return nil, err
	}

	rolesMap := make(map[string]struct{})
	for _, r := range payload.RealmAccess.Roles {
		rolesMap[strings.TrimSpace(r)] = struct{}{}
	}
	if v.cfg.ClientID != "" {
		if clientAccess, ok := payload.ResourceAccess[v.cfg.ClientID]; ok {
			for _, r := range clientAccess.Roles {
				rolesMap[strings.TrimSpace(r)] = struct{}{}
			}
		}
	}

	var roles []string
	for r := range rolesMap {
		if r != "" {
			roles = append(roles, r)
		}
	}

	var expiresAt time.Time
	if payload.Exp > 0 {
		expiresAt = time.Unix(payload.Exp, 0).UTC()
	}

	claims := model.NewClaims(
		payload.Sub,
		payload.PreferredUsername,
		payload.Email,
		roles,
		payload.Groups,
		expiresAt,
	)

	log.Info().
		Str("sub", claims.Subject()).
		Str("username", claims.Username()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed jwt verification")

	return claims, nil
}

func (v *KeycloakTokenVerifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, exists := v.keys[kid]
	lastFetch := v.lastFetch
	v.mu.RUnlock()

	if exists {
		return key, nil
	}

	// Rate-limit JWKS refresh to once per 10 seconds unless no keys exist
	if time.Since(lastFetch) < 10*time.Second && len(v.keys) > 0 {
		return nil, fmt.Errorf("key id %q not found in cached jwks", kid)
	}

	if err := v.refreshJWKS(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, exists = v.keys[kid]
	if !exists {
		// If kid is empty and there is exactly one key, default to it
		if kid == "" && len(v.keys) == 1 {
			for _, k := range v.keys {
				return k, nil
			}
		}
		return nil, fmt.Errorf("key id %q not found in refreshed jwks", kid)
	}
	return key, nil
}

func (v *KeycloakTokenVerifier) refreshJWKS(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.cfg.JWKSURL == "" {
		return fmt.Errorf("jwks url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating jwks request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed fetching jwks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed decoding jwks json: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		eInt := 0
		for _, b := range eBytes {
			eInt = (eInt << 8) | int(b)
		}

		pubKey := &rsa.PublicKey{
			N: n,
			E: eInt,
		}
		newKeys[k.Kid] = pubKey
	}

	v.keys = newKeys
	v.lastFetch = time.Now()
	return nil
}

// Static compile-time interface assertion
var _ outbound.TokenVerifierPort = (*KeycloakTokenVerifier)(nil)
