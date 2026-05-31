package guidance

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/text"

// DefaultSoulMD is the Gormes-owned port of Hermes' DEFAULT_SOUL_MD from
// hermes_cli/default_soul.py. The only intentional divergence is the product
// identity: Gorm is the editable default persona, while gormes is the
// Go-native Hermes-compatible runtime that runs it.
const DefaultSoulMD = text.DefaultSoulMD

const DefaultAgentIdentity = DefaultSoulMD
