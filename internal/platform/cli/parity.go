package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/parity"

type CLISurface = parity.CLISurface
type DocsSurface = parity.DocsSurface

func LoadCLISurface(path string) (CLISurface, error)   { return parity.LoadCLISurface(path) }
func LoadDocsSurface(path string) (DocsSurface, error) { return parity.LoadDocsSurface(path) }
