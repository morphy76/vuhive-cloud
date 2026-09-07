package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/rs/zerolog"
)

// SessionCleanerConfig configures the schedule for periodic expired session purges.
type SessionCleanerConfig struct {
	Interval time.Duration // Interval between cleanup runs (default: 10m)
}

// SessionCleaner periodically removes expired sessions from the persistent store.
type SessionCleaner struct {
	store outbound.SessionStore
	cfg   SessionCleanerConfig
	wg    sync.WaitGroup
}

// NewSessionCleaner constructs an initialized SessionCleaner instance.
func NewSessionCleaner(store outbound.SessionStore, cfgs ...SessionCleanerConfig) *SessionCleaner {
	cfg := SessionCleanerConfig{
		Interval: 10 * time.Minute,
	}
	if len(cfgs) > 0 {
		c := cfgs[0]
		if c.Interval > 0 {
			cfg.Interval = c.Interval
		}
	}

	return &SessionCleaner{
		store: store,
		cfg:   cfg,
	}
}

// CleanOnce triggers a single purge of expired sessions before the current time.
func (c *SessionCleaner) CleanOnce(ctx context.Context) (int64, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "SessionCleaner.CleanOnce").Logger()
	log.Debug().Msg("starting expired session cleanup cycle")

	if c.store == nil {
		return 0, nil
	}

	count, err := c.store.DeleteExpired(ctx, time.Now())
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed cleaning expired sessions")
		return 0, err
	}

	log.Info().
		Int64("deleted_sessions", count).
		Dur("duration_ms", time.Since(start)).
		Msg("completed expired session cleanup cycle")

	return count, nil
}

// Start begins the background janitor loop until ctx is cancelled.
// It tracks in-flight executions with sync.WaitGroup for clean, non-leaking teardown.
func (c *SessionCleaner) Start(ctx context.Context) error {
	c.wg.Add(1)
	defer c.wg.Done()

	log := zerolog.Ctx(ctx).With().
		Str("op", "SessionCleaner.Start").
		Dur("interval", c.cfg.Interval).
		Logger()
	log.Info().Msg("starting background session cleaner ticker loop")

	ticker := time.NewTicker(c.cfg.Interval)
	defer ticker.Stop()

	// Perform initial cleanup on startup
	if _, err := c.CleanOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Warn().Err(err).Msg("initial session cleanup cycle encountered error")
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("stopping background session cleaner loop")
			return ctx.Err()
		case <-ticker.C:
			if _, err := c.CleanOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Warn().Err(err).Msg("periodic session cleanup cycle encountered error")
			}
		}
	}
}

// Wait blocks until all active cleaner routines have completed.
func (c *SessionCleaner) Wait() {
	c.wg.Wait()
}
