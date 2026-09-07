package runner

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	"golang.org/x/oauth2"
)

// ResolveBearerToken returns an existing bearer token or fetches one using OAuth2 client credentials via zitadel.
func ResolveBearerToken(ctx context.Context, cfg WrapperConfig, client HTTPClient) (string, error) {
	if cfg.AuthToken != "" {
		return cfg.AuthToken, nil
	}

	if cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.TokenURL != "" {
		start := time.Now()
		log := zerolog.Ctx(ctx).With().
			Str("op", "ResolveBearerToken").
			Str("client_id", cfg.ClientID).
			Str("token_url", cfg.TokenURL).
			Logger()
		log.Debug().Msg("fetching m2m access token via client credentials")

		var opts []rp.Option
		opts = append(opts, rp.WithAuthStyle(oauth2.AuthStyleInParams))
		if client != nil {
			if hc, ok := client.(*http.Client); ok {
				opts = append(opts, rp.WithHTTPClient(hc))
			}
		}

		oauthCfg := &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL: cfg.TokenURL,
			},
		}

		relyingParty, err := rp.NewRelyingPartyOAuth(oauthCfg, opts...)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating zitadel relying party")
			return "", err
		}

		token, err := rp.ClientCredentials(ctx, relyingParty, url.Values{})
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed obtaining m2m token")
			return "", err
		}

		log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully obtained m2m access token")
		return token.AccessToken, nil
	}

	return "", nil
}

