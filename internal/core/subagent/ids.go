// internal/core/subagent/ids.go
package subagent

import "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/lifecycle"

// newSubagentID returns a fresh subagent ID of the form "sa_<13-char-base32>".
// 8 bytes of crypto/rand entropy → 13 base32 (no-padding) characters, giving
// 64 bits of randomness — collision-resistant for any realistic subagent volume.
func newSubagentID() string {
	return lifecycle.NewSubagentID()
}
