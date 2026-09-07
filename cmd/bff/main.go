package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/cache"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/controlplane"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/eventhub"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/keycloak"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/crypto"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/memory"
	sessionpg "github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/session/postgres"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/service"
	"github.com/morphy76/vuhive-cloud/internal/version"
	"github.com/morphy76/vuhive-cloud/web"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	portFlag := flag.String("port", "", "BFF HTTP port (defaults to PORT env or 8081)")
	cpURLFlag := flag.String("control-plane-url", "", "Upstream control plane URL (defaults to CONTROL_PLANE_URL env or http://localhost:8080)")
	devProxyFlag := flag.String("dev-proxy-url", "", "Vite/frontend dev server proxy URL for live-reload (defaults to DEV_PROXY_URL env)")
	staticDirFlag := flag.String("static-dir", "", "Local static directory for web assets (defaults to STATIC_DIR env, overrides embedded assets)")
	tokenFlag := flag.String("control-plane-token", "", "Bearer token for control plane API (defaults to CONTROL_PLANE_TOKEN env)")
	retriesFlag := flag.Int("control-plane-retries", 2, "Max retries for idempotent control plane requests")
	ssePollFlag := flag.Duration("sse-poll-interval", 0, "Polling interval for active run/build state changes (defaults to SSE_POLL_INTERVAL env or 2s)")
	sseHbFlag := flag.Duration("sse-heartbeat-interval", 0, "Heartbeat interval for SSE streams (defaults to SSE_HEARTBEAT_INTERVAL env or 15s)")

	// Session & PostgreSQL flags
	dbURLFlag := flag.String("database-url", "", "PostgreSQL database connection URL for persistent HTTP sessions (defaults to DATABASE_URL or POSTGRES_URL env)")
	autoMigrateFlag := flag.Bool("auto-migrate", true, "Auto-run database migrations on startup if database-url is provided (defaults to AUTO_MIGRATE env)")
	sessionEncKeyFlag := flag.String("session-encryption-key", "", "Secret key for AES-256-GCM session token encryption (defaults to SESSION_ENCRYPTION_KEY env)")
	sessionTTLFlag := flag.Duration("session-ttl", 0, "Session inactivity timeout (defaults to SESSION_TTL env or 24h)")
	sessionSlidingFlag := flag.Duration("session-sliding-threshold", 0, "Sliding expiration write-throttling threshold (defaults to SESSION_SLIDING_THRESHOLD env or 15m)")
	sessionCleanerFlag := flag.Duration("session-cleaner-interval", 0, "Background janitor cleanup interval for expired sessions (defaults to SESSION_CLEANER_INTERVAL env or 10m)")

	// Keycloak OIDC flags
	keycloakIssuerFlag := flag.String("keycloak-issuer-url", "", "Keycloak realm issuer URL (defaults to KEYCLOAK_ISSUER_URL or KEYCLOAK_URL env)")
	keycloakBaseURLFlag := flag.String("keycloak-base-url", "", "Keycloak base URL (defaults to KEYCLOAK_BASE_URL env)")
	keycloakRealmFlag := flag.String("keycloak-realm", "", "Keycloak realm name (defaults to KEYCLOAK_REALM env or vuhive)")
	keycloakClientIDFlag := flag.String("keycloak-client-id", "", "Keycloak confidential client ID for BFF (defaults to KEYCLOAK_CLIENT_ID env or vuhive-cloud-bff)")
	keycloakSecretFlag := flag.String("keycloak-client-secret", "", "Keycloak confidential client secret (defaults to KEYCLOAK_CLIENT_SECRET env)")
	keycloakJWKSFlag := flag.String("keycloak-jwks-url", "", "Custom Keycloak JWKS public keys URL override (defaults to KEYCLOAK_JWKS_URL env)")
	cookieNameFlag := flag.String("session-cookie-name", "", "Name for session cookie (defaults to SESSION_COOKIE_NAME env or vuhive_session)")
	authEnabledFlag := flag.Bool("auth-enabled", false, "Force enable OIDC authentication and session middleware (defaults to AUTH_ENABLED env)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vuhive-cloud bff %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildTime)
		os.Exit(0)
	}

	// Configure structured logging with zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8081"
	}

	cpURL := *cpURLFlag
	if cpURL == "" {
		cpURL = os.Getenv("CONTROL_PLANE_URL")
	}
	if cpURL == "" {
		cpURL = "http://localhost:8080"
	}

	cpToken := *tokenFlag
	if cpToken == "" {
		cpToken = os.Getenv("CONTROL_PLANE_TOKEN")
	}

	maxRetries := *retriesFlag

	devProxyURLStr := *devProxyFlag
	if devProxyURLStr == "" {
		devProxyURLStr = os.Getenv("DEV_PROXY_URL")
	}

	staticDir := *staticDirFlag
	if staticDir == "" {
		staticDir = os.Getenv("STATIC_DIR")
	}

	pollInterval := *ssePollFlag
	if pollInterval <= 0 {
		if envVal := os.Getenv("SSE_POLL_INTERVAL"); envVal != "" {
			if d, err := time.ParseDuration(envVal); err == nil && d > 0 {
				pollInterval = d
			}
		}
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	heartbeatInterval := *sseHbFlag
	if heartbeatInterval <= 0 {
		if envVal := os.Getenv("SSE_HEARTBEAT_INTERVAL"); envVal != "" {
			if d, err := time.ParseDuration(envVal); err == nil && d > 0 {
				heartbeatInterval = d
			}
		}
	}
	if heartbeatInterval <= 0 {
		heartbeatInterval = 15 * time.Second
	}

	// Session & Database configuration
	dbURL := *dbURLFlag
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = os.Getenv("POSTGRES_URL")
	}

	autoMigrate := *autoMigrateFlag
	if envVal := os.Getenv("AUTO_MIGRATE"); envVal != "" {
		autoMigrate = strings.ToLower(envVal) != "false"
	}

	sessionEncKey := *sessionEncKeyFlag
	if sessionEncKey == "" {
		sessionEncKey = os.Getenv("SESSION_ENCRYPTION_KEY")
	}

	sessionTTL := *sessionTTLFlag
	if sessionTTL <= 0 {
		if envVal := os.Getenv("SESSION_TTL"); envVal != "" {
			if d, err := time.ParseDuration(envVal); err == nil && d > 0 {
				sessionTTL = d
			}
		}
	}
	if sessionTTL <= 0 {
		sessionTTL = 24 * time.Hour
	}

	sessionSliding := *sessionSlidingFlag
	if sessionSliding <= 0 {
		if envVal := os.Getenv("SESSION_SLIDING_THRESHOLD"); envVal != "" {
			if d, err := time.ParseDuration(envVal); err == nil && d > 0 {
				sessionSliding = d
			}
		}
	}
	if sessionSliding <= 0 {
		sessionSliding = 15 * time.Minute
	}

	sessionCleanerInterval := *sessionCleanerFlag
	if sessionCleanerInterval <= 0 {
		if envVal := os.Getenv("SESSION_CLEANER_INTERVAL"); envVal != "" {
			if d, err := time.ParseDuration(envVal); err == nil && d > 0 {
				sessionCleanerInterval = d
			}
		}
	}
	if sessionCleanerInterval <= 0 {
		sessionCleanerInterval = 10 * time.Minute
	}

	// Keycloak OIDC configuration
	keycloakIssuerURL := *keycloakIssuerFlag
	if keycloakIssuerURL == "" {
		keycloakIssuerURL = os.Getenv("KEYCLOAK_ISSUER_URL")
	}
	if keycloakIssuerURL == "" {
		keycloakIssuerURL = os.Getenv("KEYCLOAK_URL")
	}

	keycloakBaseURL := *keycloakBaseURLFlag
	if keycloakBaseURL == "" {
		keycloakBaseURL = os.Getenv("KEYCLOAK_BASE_URL")
	}

	keycloakRealm := *keycloakRealmFlag
	if keycloakRealm == "" {
		keycloakRealm = os.Getenv("KEYCLOAK_REALM")
	}
	if keycloakRealm == "" {
		keycloakRealm = "vuhive"
	}

	keycloakClientID := *keycloakClientIDFlag
	if keycloakClientID == "" {
		keycloakClientID = os.Getenv("KEYCLOAK_CLIENT_ID")
	}
	if keycloakClientID == "" {
		keycloakClientID = "vuhive-cloud-bff"
	}

	keycloakSecret := *keycloakSecretFlag
	if keycloakSecret == "" {
		keycloakSecret = os.Getenv("KEYCLOAK_CLIENT_SECRET")
	}

	keycloakJWKS := *keycloakJWKSFlag
	if keycloakJWKS == "" {
		keycloakJWKS = os.Getenv("KEYCLOAK_JWKS_URL")
	}

	cookieName := *cookieNameFlag
	if cookieName == "" {
		cookieName = os.Getenv("SESSION_COOKIE_NAME")
	}
	if cookieName == "" {
		cookieName = "vuhive_session"
	}

	authEnabled := *authEnabledFlag
	if envVal := os.Getenv("AUTH_ENABLED"); envVal != "" {
		authEnabled = strings.ToLower(envVal) == "true"
	} else if !authEnabled {
		// Auto-detect authentication if Keycloak is configured
		authEnabled = keycloakIssuerURL != "" || keycloakBaseURL != "" || keycloakSecret != ""
	}

	log.Info().
		Str("version", version.Version).
		Str("commit", version.Commit).
		Str("build_time", version.BuildTime).
		Str("port", port).
		Str("control_plane_url", cpURL).
		Str("dev_proxy_url", devProxyURLStr).
		Str("static_dir", staticDir).
		Dur("sse_poll_interval", pollInterval).
		Dur("sse_heartbeat_interval", heartbeatInterval).
		Bool("auth_enabled", authEnabled).
		Bool("has_database", dbURL != "").
		Bool("auto_migrate", autoMigrate).
		Dur("session_ttl", sessionTTL).
		Dur("session_sliding_threshold", sessionSliding).
		Dur("session_cleaner_interval", sessionCleanerInterval).
		Msg("starting vuhive-cloud backend-for-frontend (bff) service")

	// Initialize outbound control plane client, cache, and event hub
	cpClient := controlplane.NewClient(controlplane.Config{
		BaseURL:    cpURL,
		Timeout:    5 * time.Second,
		AuthToken:  cpToken,
		MaxRetries: maxRetries,
	})
	cacheAdapter := cache.NewMemoryCache()
	eventHub := eventhub.NewHub(64)

	// Initialize application service
	bffService := service.NewBFFService(cpClient, cacheAdapter, version.Version, eventHub)

	// Start background event poller for live status updates
	poller := service.NewPoller(cpClient, eventHub, service.PollerConfig{
		PollInterval:      pollInterval,
		HeartbeatInterval: heartbeatInterval,
	})
	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	go func() {
		if err := poller.Start(pollerCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("background event poller encountered error")
		}
	}()

	// Initialize optional token encryption cipher
	var cipher *crypto.TokenCipher
	if sessionEncKey != "" {
		var err error
		if len([]byte(sessionEncKey)) == 32 {
			cipher, err = crypto.NewTokenCipher([]byte(sessionEncKey))
		} else {
			cipher, err = crypto.NewTokenCipherFromPassphrase(sessionEncKey)
		}
		if err != nil {
			log.Fatal().Err(err).Msg("failed initializing session token encryption cipher")
		}
		log.Info().Msg("initialized AES-256-GCM session token encryption cipher")
	}

	// Initialize session storage (PostgreSQL with auto-migrations or in-memory fallback)
	var sessionStore outbound.SessionStore
	var pool *pgxpool.Pool
	if dbURL != "" {
		if autoMigrate {
			log.Info().Msg("running bff database migrations on startup")
			migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := sessionpg.MigrateUpURL(migrateCtx, dbURL); err != nil {
				migrateCancel()
				log.Fatal().Err(err).Msg("bff database migrations failed on startup; halting bff")
			}
			migrateCancel()
			log.Info().Msg("completed bff database migrations on startup")
		}

		poolConfig, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			log.Fatal().Err(err).Msg("failed parsing postgres database url")
		}
		pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			log.Fatal().Err(err).Msg("failed connecting to postgres database pool")
		}
		defer pool.Close()

		var storeOpts []sessionpg.Option
		if cipher != nil {
			storeOpts = append(storeOpts, sessionpg.WithCipher(cipher))
		}
		sessionStore = sessionpg.NewPostgresSessionStore(pool, storeOpts...)
		log.Info().Msg("initialized persistent postgresql session store")
	} else {
		sessionStore = memory.NewMemorySessionStore()
		log.Warn().Msg("running with in-memory session store (sessions will not survive restarts or pod rescheduling)")
	}

	// Initialize SessionService orchestration
	sessionService := service.NewSessionService(sessionStore, service.SessionServiceConfig{
		SessionTTL:       sessionTTL,
		SlidingThreshold: sessionSliding,
	})

	// Start background session cleaner janitor
	sessionCleaner := service.NewSessionCleaner(sessionStore, service.SessionCleanerConfig{
		Interval: sessionCleanerInterval,
	})
	cleanerCtx, cleanerCancel := context.WithCancel(context.Background())
	go func() {
		if err := sessionCleaner.Start(cleanerCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("background session cleaner encountered error")
		}
	}()

	// Initialize Keycloak OIDC client & AuthHandler if authentication is configured
	var oidcClient outbound.OIDCClient
	var authHandler *rest.AuthHandler

	if keycloakIssuerURL != "" || keycloakBaseURL != "" || keycloakSecret != "" {
		kcCfg := keycloak.Config{
			BaseURL:      keycloakBaseURL,
			Realm:        keycloakRealm,
			ClientID:     keycloakClientID,
			ClientSecret: keycloakSecret,
			IssuerURL:    keycloakIssuerURL,
			JWKSURL:      keycloakJWKS,
		}
		if kcCfg.BaseURL == "" && kcCfg.IssuerURL != "" {
			parts := strings.Split(kcCfg.IssuerURL, "/realms/")
			if len(parts) == 2 {
				kcCfg.BaseURL = parts[0]
				if kcCfg.Realm == "" {
					kcCfg.Realm = parts[1]
				}
			}
		}
		client := keycloak.NewKeycloakClient(kcCfg)
		oidcClient = client

		authHandler = rest.NewAuthHandler(rest.AuthHandlerConfig{
			SessionService: sessionService,
			OIDCClient:     oidcClient,
			CookieName:     cookieName,
			CookiePath:     "/",
			SessionTTL:     sessionTTL,
		})
		log.Info().
			Str("client_id", keycloakClientID).
			Str("issuer_url", keycloakIssuerURL).
			Msg("configured keycloak oidc client and authentication handlers")
	}

	// Configure SPA serving (embedded assets, local filesystem override, or Vite dev proxy)
	spaConfig := rest.SPAConfig{}
	if devProxyURLStr != "" {
		parsedProxyURL, err := url.Parse(devProxyURLStr)
		if err != nil {
			log.Fatal().Err(err).Str("dev_proxy_url", devProxyURLStr).Msg("invalid dev proxy URL")
		}
		spaConfig.DevProxyURL = parsedProxyURL
		log.Info().Str("target", parsedProxyURL.String()).Msg("configured SPA live-reload dev proxy")
	} else if staticDir != "" {
		spaConfig.FileSystem = os.DirFS(staticDir)
		log.Info().Str("dir", staticDir).Msg("configured SPA file server with local static directory")
	} else {
		distFS, err := web.GetFS()
		if err != nil {
			log.Fatal().Err(err).Msg("failed initializing embedded SPA assets")
		}
		spaConfig.FileSystem = distFS
		log.Info().Msg("configured SPA file server with embedded production assets")
	}

	// Setup inbound REST router with control plane transparent proxying & auth session middleware
	router := rest.SetupRouterWithConfig(rest.RouterConfig{
		BFFService:      bffService,
		Version:         version.Version,
		ControlPlaneURL: cpURL,
		ProxyTransport:  nil,
		SPAConfig:       &spaConfig,
		AuthHandler:     authHandler,
		SessionService:  sessionService,
		OIDCClient:      oidcClient,
		CookieName:      cookieName,
		AuthEnabled:     authEnabled,
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Listen for OS interrupt and termination signals for graceful teardown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", server.Addr).Msg("bff listening for incoming HTTP requests")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed starting bff http server")
		}
	}()

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("received shutdown signal, shutting down bff gracefully")

	pollerCancel()
	cleanerCancel()
	sessionCleaner.Wait()
	_ = eventHub.Close()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("bff graceful shutdown encountered error")
	} else {
		log.Info().Msg("bff server gracefully stopped")
	}
}
