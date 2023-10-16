package cli

import (
	"runtime"
	"time"
)

// AppName is the name of the CLI application.
const AppName = "pkgdmp"

// Build information set by the compiler.
var (
	buildVersion   = ""
	buildCommit    = ""
	buildTime      = ""
	buildGoVersion = ""
)

// Version of pkgdmp.
//
// Returns `0.0.0-dev` if no version is set.
func Version() string {
	if buildVersion == "" {
		return "0.0.0-dev"
	}

	return buildVersion
}

// BuildCommit returns the git commit hash pkgdmp was built from.
//
// Returns `HEAD` if no build commit is set.
func BuildCommit() string {
	if buildCommit == "" {
		return "HEAD"
	}

	return buildCommit
}

// BuildTime returns the UTC time pkgdmp was built.
//
// Returns current time in UTC if not set.
func BuildTime() string {
	if buildTime == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}

	return buildTime
}

// BuildGoVersion returns the go version pkgdmp was built with.
//
// Returns version from [runtime.Version] if not set.
func BuildGoVersion() string {
	if buildGoVersion == "" {
		return runtime.Version()
	}

	return buildGoVersion
}
