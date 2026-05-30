package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/termux"

// TermuxWakeLockManager acquires and releases the Android wake lock through
// termux-wake-lock. It is a no-op on non-Termux hosts.
type TermuxWakeLockManager = termux.WakeLockManager
