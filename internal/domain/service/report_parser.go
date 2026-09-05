package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// ParsedReport contains indexed metrics and SLA compliance status extracted from summary.json.
type ParsedReport struct {
	Metrics   model.RunMetrics
	SLAPassed bool
	RawJSON   []byte
}

// rawMetricEntry represents an individual metric item inside vuhive summary.json.
type rawMetricEntry struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Count int64   `json:"count,omitempty"`
	Value float64 `json:"value,omitempty"`
	Rate  float64 `json:"rate,omitempty"`
	Min   any     `json:"min,omitempty"`
	Mean  any     `json:"mean,omitempty"`
	P50   any     `json:"p50,omitempty"`
	P90   any     `json:"p90,omitempty"`
	P95   any     `json:"p95,omitempty"`
	P99   any     `json:"p99,omitempty"`
	Max   any     `json:"max,omitempty"`
}

// rawThresholdEntry represents an SLA threshold result inside vuhive summary.json.
type rawThresholdEntry struct {
	Metric   string `json:"metric"`
	Stat     string `json:"stat"`
	Operator string `json:"operator"`
	Target   string `json:"target"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

// rawSummaryDocument represents the full structure of summary.json, covering both
// standard vuhive.SummaryData schema and flat/fallback report formats.
type rawSummaryDocument struct {
	SuiteName         string              `json:"suite_name"`
	Scenario          string              `json:"scenario"`
	Status            string              `json:"status"`
	Passed            *bool               `json:"passed"`
	SLAPassed         *bool               `json:"sla_passed"`
	SLA               *bool               `json:"sla"`
	StartedAt         *time.Time          `json:"started_at"`
	EndedAt           *time.Time          `json:"ended_at"`
	Duration          any                 `json:"duration"`
	Metrics           []rawMetricEntry    `json:"metrics"`
	Thresholds        []rawThresholdEntry `json:"thresholds"`
	TotalIterations   *int64              `json:"total_iterations"`
	Iterations        *int64              `json:"iterations"`
	TotalRequests     *int64              `json:"total_requests"`
	Requests          *int64              `json:"requests"`
	AvgTPS            *float64            `json:"avg_tps"`
	TPS               *float64            `json:"tps"`
	Throughput        *float64            `json:"throughput"`
	P50DurationMs     *float64            `json:"p50_duration_ms"`
	P90DurationMs     *float64            `json:"p90_duration_ms"`
	P95DurationMs     *float64            `json:"p95_duration_ms"`
	P99DurationMs     *float64            `json:"p99_duration_ms"`
	P50               any                 `json:"p50"`
	P90               any                 `json:"p90"`
	P95               any                 `json:"p95"`
	P99               any                 `json:"p99"`
	ErrorRatePct      *float64            `json:"error_rate_pct"`
	ErrorRate         *float64            `json:"error_rate"`
}

// ParseSummaryReport parses deterministic vuhive summary.json bytes and extracts
// SLA compliance pass/fail, throughput TPS, error rate %, and latency percentiles.
func ParseSummaryReport(raw []byte) (*ParsedReport, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: summary json cannot be empty", model.ErrValidation)
	}

	var doc rawSummaryDocument
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%w: invalid summary json format: %v", model.ErrValidation, err)
	}

	// 1. SLA pass/fail determination
	slaPassed := true
	if doc.Passed != nil {
		slaPassed = *doc.Passed
	}
	if doc.SLAPassed != nil {
		slaPassed = *doc.SLAPassed
	}
	if doc.SLA != nil {
		slaPassed = *doc.SLA
	}
	if strings.EqualFold(doc.Status, "FAILED") {
		slaPassed = false
	}
	for _, th := range doc.Thresholds {
		if !th.Passed {
			slaPassed = false
			break
		}
	}

	// 2. Metrics extraction
	var (
		totalIterations  int64
		iterationsFailed int64
		totalRequests    int64
		avgTPS           float64
		p50Ms            float64
		p90Ms            float64
		p95Ms            float64
		p99Ms            float64
		errorRatePct     float64
	)

	// Map metrics by name for lookup
	metricMap := make(map[string]rawMetricEntry, len(doc.Metrics))
	var primaryDurationMetric *rawMetricEntry

	for i := range doc.Metrics {
		m := doc.Metrics[i]
		metricMap[m.Name] = m

		if m.Type == "duration" {
			if m.Name == "vuhive.http.req_duration" {
				primaryDurationMetric = &m
			} else if primaryDurationMetric == nil || primaryDurationMetric.Name != "vuhive.http.req_duration" {
				if m.Name == "vuhive.vu.iteration_duration" || primaryDurationMetric == nil {
					primaryDurationMetric = &m
				}
			}
		}
	}

	// Total iterations
	if doc.TotalIterations != nil {
		totalIterations = *doc.TotalIterations
	} else if doc.Iterations != nil {
		totalIterations = *doc.Iterations
	} else if m, ok := metricMap["vuhive.vu.iterations_total"]; ok {
		totalIterations = m.Count
	}

	if m, ok := metricMap["vuhive.vu.iterations_failed"]; ok {
		iterationsFailed = m.Count
	}

	// Total requests
	if doc.TotalRequests != nil {
		totalRequests = *doc.TotalRequests
	} else if doc.Requests != nil {
		totalRequests = *doc.Requests
	} else if m, ok := metricMap["vuhive.http.reqs"]; ok {
		totalRequests = m.Count
	} else if m, ok := metricMap["vuhive.kafka.pub_total"]; ok {
		totalRequests = m.Count
	} else if m, ok := metricMap["vuhive.nats.pub_total"]; ok {
		totalRequests = m.Count
	}

	// Duration in seconds
	var durationSec float64
	if doc.StartedAt != nil && doc.EndedAt != nil && doc.EndedAt.After(*doc.StartedAt) {
		durationSec = doc.EndedAt.Sub(*doc.StartedAt).Seconds()
	} else if doc.Duration != nil {
		durationSec = parseDurationSeconds(doc.Duration)
	}

	// Avg TPS
	if doc.AvgTPS != nil {
		avgTPS = *doc.AvgTPS
	} else if doc.TPS != nil {
		avgTPS = *doc.TPS
	} else if doc.Throughput != nil {
		avgTPS = *doc.Throughput
	} else if durationSec > 0 {
		if totalRequests > 0 {
			avgTPS = float64(totalRequests) / durationSec
		} else if totalIterations > 0 {
			avgTPS = float64(totalIterations) / durationSec
		}
	}

	// Latency percentiles
	if doc.P50DurationMs != nil {
		p50Ms = *doc.P50DurationMs
	} else if doc.P50 != nil {
		p50Ms = parseDurationMs(doc.P50)
	} else if primaryDurationMetric != nil && primaryDurationMetric.P50 != nil {
		p50Ms = parseDurationMs(primaryDurationMetric.P50)
	}

	if doc.P90DurationMs != nil {
		p90Ms = *doc.P90DurationMs
	} else if doc.P90 != nil {
		p90Ms = parseDurationMs(doc.P90)
	} else if primaryDurationMetric != nil && primaryDurationMetric.P90 != nil {
		p90Ms = parseDurationMs(primaryDurationMetric.P90)
	}

	if doc.P95DurationMs != nil {
		p95Ms = *doc.P95DurationMs
	} else if doc.P95 != nil {
		p95Ms = parseDurationMs(doc.P95)
	} else if primaryDurationMetric != nil && primaryDurationMetric.P95 != nil {
		p95Ms = parseDurationMs(primaryDurationMetric.P95)
	}

	if doc.P99DurationMs != nil {
		p99Ms = *doc.P99DurationMs
	} else if doc.P99 != nil {
		p99Ms = parseDurationMs(doc.P99)
	} else if primaryDurationMetric != nil && primaryDurationMetric.P99 != nil {
		p99Ms = parseDurationMs(primaryDurationMetric.P99)
	}

	// Error rate %
	if doc.ErrorRatePct != nil {
		errorRatePct = *doc.ErrorRatePct
	} else if doc.ErrorRate != nil {
		errorRatePct = *doc.ErrorRate
		if errorRatePct <= 1.0 && errorRatePct > 0 {
			errorRatePct *= 100.0
		}
	} else if m, ok := metricMap["vuhive.http.req_failed"]; ok {
		errorRatePct = m.Rate * 100.0
	} else if totalIterations > 0 && iterationsFailed > 0 {
		errorRatePct = (float64(iterationsFailed) / float64(totalIterations)) * 100.0
	}

	metrics := model.RunMetrics{
		TotalIterations: totalIterations,
		TotalRequests:   totalRequests,
		AvgTPS:          avgTPS,
		P50DurationMs:   p50Ms,
		P90DurationMs:   p90Ms,
		P95DurationMs:   p95Ms,
		P99DurationMs:   p99Ms,
		ErrorRatePct:    errorRatePct,
	}

	return &ParsedReport{
		Metrics:   metrics,
		SLAPassed: slaPassed,
		RawJSON:   trimmed,
	}, nil
}

func parseDurationSeconds(v any) float64 {
	switch val := v.(type) {
	case float64:
		if val > 100000 {
			return val / 1e9
		}
		return val
	case int64:
		if val > 100000 {
			return float64(val) / 1e9
		}
		return float64(val)
	case json.Number:
		if f, err := val.Float64(); err == nil {
			if f > 100000 {
				return f / 1e9
			}
			return f
		}
	case string:
		s := strings.TrimSpace(val)
		if d, err := time.ParseDuration(s); err == nil {
			return d.Seconds()
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			if f > 100000 {
				return f / 1e9
			}
			return f
		}
	}
	return 0
}

func parseDurationMs(v any) float64 {
	switch val := v.(type) {
	case float64:
		if val > 100000 {
			return val / 1e6
		}
		return val
	case int64:
		if val > 100000 {
			return float64(val) / 1e6
		}
		return float64(val)
	case json.Number:
		if f, err := val.Float64(); err == nil {
			if f > 100000 {
				return f / 1e6
			}
			return f
		}
	case string:
		s := strings.TrimSpace(val)
		if d, err := time.ParseDuration(s); err == nil {
			return float64(d.Nanoseconds()) / 1e6
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			if f > 100000 {
				return f / 1e6
			}
			return f
		}
	}
	return 0
}
