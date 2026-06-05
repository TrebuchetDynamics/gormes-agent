package kernel

import kernelprovenance "github.com/TrebuchetDynamics/gormes-agent/internal/kernel/provenance"

type Provenance = kernelprovenance.Provenance

func newProvenance(endpoint string) Provenance { return kernelprovenance.New(endpoint) }
