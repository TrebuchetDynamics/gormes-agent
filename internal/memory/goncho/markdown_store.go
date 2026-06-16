package goncho

import (
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho/markdown"
)

type GonchoMarkdownStoreConfig = markdown.Config
type GonchoMarkdownStore = markdown.Store
type GonchoMarkdownReloadResult = markdown.ReloadResult
type GonchoMarkdownExportResult = markdown.ExportResult
type GonchoMarkdownConflict = markdown.Conflict

func NewGonchoMarkdownStore(db *sql.DB, cfg GonchoMarkdownStoreConfig) *GonchoMarkdownStore {
	return markdown.NewStore(db, cfg)
}
