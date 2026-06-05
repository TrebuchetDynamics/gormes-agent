package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/dotenv"

func ReadDotenvValues(path string) map[string]string {
	return dotenv.ReadValues(path)
}
