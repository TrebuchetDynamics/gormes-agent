package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document"

const (
	DefaultMaxDocumentBytes = document.DefaultMaxDocumentBytes
	DefaultSelectionCap     = document.DefaultSelectionCap
)

// Skill is the typed in-memory representation of one SKILL.md artifact.
type Skill = document.Skill

type CredentialGroup = document.CredentialGroup

type SkillConditions = document.SkillConditions
