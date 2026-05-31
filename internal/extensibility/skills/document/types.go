package document

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document/model"

const (
	DefaultMaxDocumentBytes = model.DefaultMaxDocumentBytes
	DefaultSelectionCap     = model.DefaultSelectionCap
)

// Skill is the typed in-memory representation of one SKILL.md artifact.
type Skill = model.Skill

type CredentialGroup = model.CredentialGroup

type SkillConditions = model.SkillConditions
