package main

import "github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescmd"

var Version = "0.2.27"

// VersionDateAlias is the Hermes-style vYYYY.M.D paired alias for the
// operator-facing semantic release version.
var VersionDateAlias = "v2026.6.21"

// GitCommit is injected by release builds via -ldflags -X main.GitCommit=...
var GitCommit = "unknown"

// GitDirty is injected by release builds via -ldflags -X main.GitDirty=...
var GitDirty = "false"

// BuildDate is injected by release builds via -ldflags -X main.BuildDate=...
var BuildDate = "unknown"

func main() {
	gormescmd.Version = Version
	gormescmd.VersionDateAlias = VersionDateAlias
	gormescmd.GitCommit = GitCommit
	gormescmd.GitDirty = GitDirty
	gormescmd.BuildDate = BuildDate
	gormescmd.Main()
}
