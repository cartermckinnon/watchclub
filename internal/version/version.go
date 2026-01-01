package version

// These variables are set via -ldflags at build time
var (
	// GitCommit is the git commit SHA
	GitCommit = "unknown"
	// Version is the version tag (if any)
	Version = "dev"
)
