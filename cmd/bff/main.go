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
	"syscall"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/cache"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/controlplane"
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

	devProxyURLStr := *devProxyFlag
	if devProxyURLStr == "" {
		devProxyURLStr = os.Getenv("DEV_PROXY_URL")
	}

	staticDir := *staticDirFlag
	if staticDir == "" {
		staticDir = os.Getenv("STATIC_DIR")
	}

	log.Info().
		Str("version", version.Version).
		Str("commit", version.Commit).
		Str("build_time", version.BuildTime).
		Str("port", port).
		Str("control_plane_url", cpURL).
		Str("dev_proxy_url", devProxyURLStr).
		Str("static_dir", staticDir).
		Msg("starting vuhive-cloud backend-for-frontend (bff) service")

	// Initialize outbound adapters
	cpClient := controlplane.NewClient(controlplane.Config{
		BaseURL: cpURL,
		Timeout: 5 * time.Second,
	})
	cacheAdapter := cache.NewMemoryCache()

	// Initialize application service
	bffService := service.NewBFFService(cpClient, cacheAdapter, version.Version)

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

	// Setup inbound REST router
	router := rest.SetupRouter(bffService, version.Version, spaConfig)

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("bff graceful shutdown encountered error")
	} else {
		log.Info().Msg("bff server gracefully stopped")
	}
}
