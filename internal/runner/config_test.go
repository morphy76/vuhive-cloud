package runner_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/runner"
)

func TestWrapperConfig_APICallbackURLNormalization(t *testing.T) {
	tests := []struct {
		name                   string
		initialCallbackURL     string
		runID                  string
		coordinatorURL         string
		expectedCallbackURL    string
		expectedCoordinatorURL string
	}{
		{
			name:                   "empty callback URL",
			initialCallbackURL:     "",
			runID:                  "run-1",
			expectedCallbackURL:    "",
			expectedCoordinatorURL: "",
		},
		{
			name:                   "bare host and port without path",
			initialCallbackURL:     "http://vuhive-cloud:8080",
			runID:                  "run-101",
			expectedCallbackURL:    "http://vuhive-cloud:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://vuhive-cloud:8080/api/v1/runs/run-101/barrier",
		},
		{
			name:                   "bare host and port with trailing slash",
			initialCallbackURL:     "http://vuhive-cloud:8080/",
			runID:                  "run-102",
			expectedCallbackURL:    "http://vuhive-cloud:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://vuhive-cloud:8080/api/v1/runs/run-102/barrier",
		},
		{
			name:                   "base runs path without complete",
			initialCallbackURL:     "http://vuhive-cloud:8080/api/v1/runs",
			runID:                  "run-103",
			expectedCallbackURL:    "http://vuhive-cloud:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://vuhive-cloud:8080/api/v1/runs/run-103/barrier",
		},
		{
			name:                   "base runs path with trailing slash",
			initialCallbackURL:     "http://vuhive-cloud:8080/api/v1/runs/",
			runID:                  "run-104",
			expectedCallbackURL:    "http://vuhive-cloud:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://vuhive-cloud:8080/api/v1/runs/run-104/barrier",
		},
		{
			name:                   "fully qualified endpoint with complete",
			initialCallbackURL:     "http://vuhive-cloud:8080/api/v1/runs/complete",
			runID:                  "run-105",
			expectedCallbackURL:    "http://vuhive-cloud:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://vuhive-cloud:8080/api/v1/runs/run-105/barrier",
		},
		{
			name:                   "run-specific endpoint with id and complete",
			initialCallbackURL:     "http://vuhive-cloud:8080/api/v1/runs/run-106/complete",
			runID:                  "run-106",
			expectedCallbackURL:    "http://vuhive-cloud:8080/api/v1/runs/run-106/complete",
			expectedCoordinatorURL: "http://vuhive-cloud:8080/api/v1/runs/run-106/barrier",
		},
		{
			name:                   "cross-namespace FQDN with trailing dot",
			initialCallbackURL:     "http://vuhive-cloud.vuhive.svc.cluster.local.:8080/api/v1/runs/complete",
			runID:                  "run-107",
			expectedCallbackURL:    "http://vuhive-cloud.vuhive.svc.cluster.local.:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://vuhive-cloud.vuhive.svc.cluster.local.:8080/api/v1/runs/run-107/barrier",
		},
		{
			name:                   "cross-namespace bare FQDN with trailing dot",
			initialCallbackURL:     "http://vuhive-cloud.vuhive.svc.cluster.local.:8080",
			runID:                  "run-108",
			expectedCallbackURL:    "http://vuhive-cloud.vuhive.svc.cluster.local.:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://vuhive-cloud.vuhive.svc.cluster.local.:8080/api/v1/runs/run-108/barrier",
		},
		{
			name:                   "explicit coordinator URL takes precedence",
			initialCallbackURL:     "http://vuhive-cloud:8080/api/v1/runs/complete",
			coordinatorURL:         "http://custom-coordinator:9090/barrier",
			runID:                  "run-109",
			expectedCallbackURL:    "http://vuhive-cloud:8080/api/v1/runs/complete",
			expectedCoordinatorURL: "http://custom-coordinator:9090/barrier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := runner.WrapperConfig{
				RunID:          tt.runID,
				ReportKey:      "reports/" + tt.runID + ".json",
				LogsKey:        "logs/" + tt.runID + ".log",
				APICallbackURL: tt.initialCallbackURL,
				CoordinatorURL: tt.coordinatorURL,
				BarrierTimeout: 5 * time.Second,
				ReleaseDelay:   100 * time.Millisecond,
			}

			err := cfg.Validate()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCallbackURL, cfg.APICallbackURL)
			assert.Equal(t, tt.expectedCoordinatorURL, cfg.CoordinatorURL)
		})
	}
}
