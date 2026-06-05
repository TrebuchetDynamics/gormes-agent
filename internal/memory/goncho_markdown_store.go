package memory

import (
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/goncho"
)

type GonchoMarkdownStoreConfig = goncho.GonchoMarkdownStoreConfig
type GonchoMarkdownStore = goncho.GonchoMarkdownStore
type GonchoMarkdownReloadResult = goncho.GonchoMarkdownReloadResult
type GonchoMarkdownExportResult = goncho.GonchoMarkdownExportResult
type GonchoMarkdownConflict = goncho.GonchoMarkdownConflict

func NewGonchoMarkdownStore(db *sql.DB, cfg GonchoMarkdownStoreConfig) *GonchoMarkdownStore {
	return goncho.NewGonchoMarkdownStore(db, cfg)
}
