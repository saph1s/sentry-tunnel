package build

// Injected at compile time via -ldflags -X
var (
	Version = "dev"
	Date    = "unknown"
	Commit  = "unknown"
)
