package hermesrowbacked

const (
	GatewayCronRow = "Gateway, platform, webhook, and cron management CLI"
	DiagnosticsRow = "Diagnostics, backup, logs, and status CLI"
	ConfigRow      = "Hermes config migration dry-run manifest"
	ToolRow        = "Tool/runtime/security rows"
	SkillsRow      = "Skills hub direct URL install name/category guard"
	MemoryRow      = "Goncho memory integration into normal agent turn"
	KanbanRow      = "Hermes Kanban durable board core"
)

// Spec describes a Hermes-compatible command surface that is intentionally
// visible while its full implementation remains row-backed.
type Spec struct {
	Use         string
	Short       string
	Row         string
	Destructive bool
}

func DumpSpec() Spec {
	return Spec{
		Use:   "dump",
		Short: "Collect a Hermes-compatible debug dump",
		Row:   DiagnosticsRow,
	}
}

func DebugShareSpec() Spec {
	return Spec{
		Use:   "share",
		Short: "Share a debug bundle",
		Row:   DiagnosticsRow,
	}
}

func DebugDeleteSpec() Spec {
	return Spec{
		Use:         "delete",
		Short:       "Delete a shared debug bundle",
		Row:         DiagnosticsRow,
		Destructive: true,
	}
}

func BackupSpec() Spec {
	return Spec{
		Use:   "backup",
		Short: "Create a Hermes-compatible backup archive",
		Row:   "Backup/update opt-in and exclusion policy",
	}
}

func ImportSpec() Spec {
	return Spec{
		Use:   "import",
		Short: "Import a Hermes configuration or state archive",
		Row:   ConfigRow,
	}
}
