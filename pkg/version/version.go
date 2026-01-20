// Package version provides version information for the CLI.
package version

// These variables are set at build time using ldflags.
// For development builds, Version defaults to the current release version.
// Production builds should use: -ldflags "-X github.com/jtimothystewart/dtiam/pkg/version.Version=X.Y.Z"
var (
	// Version is the semantic version of the CLI.
	// Update this value before merging features/fixes to main.
	// See CLAUDE.md for version management guidelines.
	Version = "1.0.0"
	// Commit is the git commit hash.
	Commit = "unknown"
	// Date is the build date.
	Date = "unknown"
)
