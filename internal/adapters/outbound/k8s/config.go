package k8s

import "time"

// Config encapsulates configuration parameters for Kubernetes job management and orchestration.
type Config struct {
	Namespace               string
	BuilderImage            string
	CPURequest              string
	CPULimit                string
	MemoryRequest           string
	MemoryLimit             string
	ActiveDeadlineSeconds   int64
	TTLSecondsAfterFinished int32
	BackoffLimit            int32
	PollInterval            time.Duration

	// Runner specific configurations
	RunnerNamespace               string
	RunnerInitImage               string
	RunnerDefaultImage            string
	RunnerActiveDeadlineSeconds   int64
	RunnerTTLSecondsAfterFinished int32
	RunnerBackoffLimit            int32
	S3Endpoint                    string
	S3Region                      string
	S3Bucket                      string
	S3AccessKeyID                 string
	S3SecretAccessKey             string
	S3UsePathStyle                bool
	APICallbackURL                string
}

// DefaultConfig returns a Config initialized with production-grade defaults.
func DefaultConfig() Config {
	return Config{
		Namespace:               "vuhive-system",
		BuilderImage:            "golang:1.26-alpine",
		CPURequest:              "1000m",
		CPULimit:                "2000m",
		MemoryRequest:           "1Gi",
		MemoryLimit:             "2Gi",
		ActiveDeadlineSeconds:   600,
		TTLSecondsAfterFinished: 3600,
		BackoffLimit:            0,
		PollInterval:            500 * time.Millisecond,

		RunnerNamespace:               "vuhive-runners",
		RunnerInitImage:               "ghcr.io/morphy76/vuhive-cloud/runner-init:latest",
		RunnerDefaultImage:            "alpine:3.20",
		RunnerActiveDeadlineSeconds:   3600,
		RunnerTTLSecondsAfterFinished: 86400,
		RunnerBackoffLimit:            0,
	}
}
