package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newSecretsCommand() *cobra.Command {
	return gormescli.NewSecretsCommand(secretsCommandOptions())
}

func secretsCommandOptions() gormescli.SecretsOptions {
	return gormescli.SecretsOptions{BuildProvenance: secretsBuildProvenance}
}

func secretsBuildProvenance() gormescli.SecretsBuildProvenance {
	build := newBuildProvenance()
	return gormescli.SecretsBuildProvenance{
		Version:   build.Version,
		GitCommit: build.GitCommit,
	}
}
