package memory

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/diagnostics"
)

// DeadLetterSummary is the operator-facing shape for one recent turn that
// exhausted extractor retries and was parked in the dead-letter state.
type DeadLetterSummary = diagnostics.DeadLetterSummary

// DeadLetterErrorSummary groups dead-letter turns by the persisted extractor
// error message so operators can spot repeated failure modes quickly.
type DeadLetterErrorSummary = diagnostics.DeadLetterErrorSummary

// SkippedSyncSummary is one interrupted/cancelled turn that deliberately
// stayed out of extraction and recall.
type SkippedSyncSummary = diagnostics.SkippedSyncSummary

// ExtractorStatus is the Phase 3.E.4 read model behind `gormes memory status`.
type ExtractorStatus = diagnostics.ExtractorStatus

// SchemaStatus is the operator-facing memory schema snapshot used by doctor
// commands. It intentionally reads the already-open database instead of
// opening or migrating a second store.
type SchemaStatus = diagnostics.SchemaStatus

// CurrentSchemaVersion returns the schema version this binary expects.
func CurrentSchemaVersion() string {
	return schemaVersion
}

// ReadSchemaStatus reports schema version and key table presence for memory
// and Goncho diagnostics.
func ReadSchemaStatus(ctx context.Context, db *sql.DB) (SchemaStatus, error) {
	return diagnostics.ReadSchemaStatus(ctx, db, schemaVersion)
}

// ReadExtractorStatus summarizes extractor backlog and recent dead letters from
// the persisted SQLite turns table. The worker is async and ephemeral, so
// health is inferred from durable queue/dead-letter state instead of process
// liveness.
func ReadExtractorStatus(ctx context.Context, db *sql.DB, deadLetterLimit int) (ExtractorStatus, error) {
	return diagnostics.ReadExtractorStatus(ctx, db, deadLetterLimit)
}
