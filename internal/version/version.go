package version

var (
	// Version is the semantic version of the component, injected at compile time via ldflags.
	Version = "dev"
	// Commit is the git commit hash of the build, injected at compile time via ldflags.
	Commit = "none"
	// BuildTime is the UTC timestamp of the build, injected at compile time via ldflags.
	BuildTime = "unknown"
)
