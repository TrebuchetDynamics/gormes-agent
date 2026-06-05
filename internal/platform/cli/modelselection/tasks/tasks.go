package tasks

// AuxiliaryTaskEntry describes an auxiliary model-configuration slot shown by
// model-selection flows.
type AuxiliaryTaskEntry struct {
	Key         string
	Label       string
	Description string
}

func DefaultAuxiliaryTaskEntries() []AuxiliaryTaskEntry {
	return []AuxiliaryTaskEntry{
		{Key: "vision", Label: "Vision", Description: "image/screenshot analysis"},
		{Key: "compression", Label: "Compression", Description: "context summarization"},
		{Key: "web_extract", Label: "Web extract", Description: "web page summarization"},
		{Key: "session_search", Label: "Session search", Description: "past-conversation recall"},
		{Key: "approval", Label: "Approval", Description: "smart command approval"},
		{Key: "mcp", Label: "MCP", Description: "MCP tool reasoning"},
		{Key: "title_generation", Label: "Title generation", Description: "session titles"},
		{Key: "skills_hub", Label: "Skills hub", Description: "skills search/install"},
		{Key: "curator", Label: "Curator", Description: "skill-usage review pass"},
	}
}
