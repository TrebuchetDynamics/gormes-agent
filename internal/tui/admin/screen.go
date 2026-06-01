package admin

import shellcontracts "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/contracts/shell"

// Screen is one tab in the unified admin TUI. The root alias preserves the
// original admin package API while the focused contracts package lets future
// screen subpackages depend on the narrow shell contract without importing all
// concrete admin screens.
type Screen = shellcontracts.Screen

// KeyCapturingScreen can ask the shell to deliver focused keypresses before
// global shortcuts are considered. This is used by text-entry modes.
type KeyCapturingScreen = shellcontracts.KeyCapturingScreen

// KeyHelp is one row in the help overlay: a list of keys and a short
// description of what they do on the active screen.
type KeyHelp = shellcontracts.KeyHelp
