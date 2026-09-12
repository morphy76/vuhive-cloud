package k8s

import "time"

// BuilderProxyConfig encapsulates optional proxy and Go module network settings
// injected into ephemeral builder container environments. All fields are optional;
// empty strings are omitted from the generated Job env vars.
type BuilderProxyConfig struct {
	// HTTP_PROXY — HTTP proxy URL for the builder container (e.g. "http://proxy.corp:3128").
	HTTPProxy string
	// HTTPS_PROXY — HTTPS proxy URL for the builder container.
	HTTPSProxy string
	// NO_PROXY — comma-separated list of hosts/CIDRs that bypass the proxy.
	NoProxy string
	// GOPROXY — Go module proxy chain (e.g. "https://goproxy.corp,direct").
	// When empty, Go's built-in default ("https://proxy.golang.org,direct") applies.
	GoProxy string
	// GOPRIVATE — comma-separated module path prefixes fetched directly (bypassing GOPROXY and GONOSUMCHECK).
	GoPrivate string
	// GONOSUMCHECK — comma-separated module path patterns whose checksums are not verified.
	GoNosumcheck string
}

// BuilderDNSConfig mirrors the fields of corev1.PodDNSConfig relevant for
// builder pod DNS customization without introducing a hard k8s import at the config layer.
type BuilderDNSConfig struct {
	// Nameservers is a list of DNS server IP addresses.
	Nameservers []string
	// Searches is a list of DNS search domains.
	Searches []string
}

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

	// Builder proxy & network configuration (Issue #187).
	// All proxy settings are optional; when zero-valued no extra env vars are injected.
	BuilderProxy BuilderProxyConfig
	// BuilderDNSPolicy optionally overrides the builder pod's dnsPolicy
	// (e.g. "None", "ClusterFirstWithHostNet"). Empty string preserves the cluster default.
	BuilderDNSPolicy string
	// BuilderDNSConfig provides custom DNS nameservers and search domains when
	// BuilderDNSPolicy is "None" or requires custom resolution.
	BuilderDNSConfig *BuilderDNSConfig

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
	RunnerAuthToken               string
	RunnerClientID                string
	RunnerClientSecret            string
	RunnerTokenURL                string
}

// DefaultGoImage is the cluster-wide fallback container image for ephemeral compilation jobs.
const DefaultGoImage = "golang:1.26-alpine"

// DefaultConfig returns a Config initialized with production-grade defaults.
func DefaultConfig() Config {
	return Config{
		Namespace:               "vuhive-system",
		BuilderImage:            DefaultGoImage,
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
