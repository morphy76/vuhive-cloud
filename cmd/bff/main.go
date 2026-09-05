package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/cache"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/controlplane"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/service"
	"github.com/morphy76/vuhive-cloud/internal/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	portFlag := flag.String("port", "", "BFF HTTP port (defaults to PORT env or 8081)")
	cpURLFlag := flag.String("control-plane-url", "", "Upstream control plane URL (defaults to CONTROL_PLANE_URL env or http://localhost:8080)")
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

	log.Info().
		Str("version", version.Version).
		Str("commit", version.Commit).
		Str("build_time", version.BuildTime).
		Str("port", port).
		Str("control_plane_url", cpURL).
		Msg("starting vuhive-cloud backend-for-frontend (bff) service")

	// Initialize outbound adapters
	cpClient := controlplane.NewClient(controlplane.Config{
		BaseURL: cpURL,
		Timeout: 5 * time.Second,
	})
	cacheAdapter := cache.NewMemoryCache()

	// Initialize application service
	bffService := service.NewBFFService(cpClient, cacheAdapter, version.Version)

	// Setup inbound REST router
	router := rest.SetupRouter(bffService, version.Version)

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
