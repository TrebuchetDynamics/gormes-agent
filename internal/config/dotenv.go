package config

import (
	"io"

	envoverlay "github.com/TrebuchetDynamics/gormes-agent/internal/config/envoverlay"
)

// parseDotenv reads KEY=VALUE lines from r and returns the key→value map.
func parseDotenv(r io.Reader) (map[string]string, error) {
	return envoverlay.ParseDotenv(r)
}

// loadDotenvFiles reads the Gormes-native dotenv file and populates os.Setenv
// for any key not already present in the original shell environment.
func loadDotenvFiles() {
	envoverlay.LoadDotenvFiles()
}

type OptionalEnvEvidence = envoverlay.OptionalEnvEvidence

func CheckOptionalEnvAny(names ...string) OptionalEnvEvidence {
	return envoverlay.CheckOptionalEnvAny(names...)
}

func checkOptionalEnvAnyWithLookup(lookup func(string) (string, bool), names ...string) OptionalEnvEvidence {
	return envoverlay.CheckOptionalEnvAnyWithLookup(lookup, names...)
}
