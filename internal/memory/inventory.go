package memory

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/provenance"
)

// InventoryState is the file/directory presence state used by the memory
// provenance inventory. It stays intentionally small so JSON consumers can
// branch without parsing free-form text.
type InventoryState = provenance.InventoryState

const (
	InventoryStatePresent = provenance.InventoryStatePresent
	InventoryStateMissing = provenance.InventoryStateMissing
	InventoryStateError   = provenance.InventoryStateError
)

// InventoryOptions controls ReadInventory. Callers pass the active profile
// root explicitly so tests and profile-aware CLI invocations never inspect the
// developer's live home by accident.
type InventoryOptions = provenance.InventoryOptions

// Inventory is the provenance-explicit read model behind
// `gormes memory status`. Goncho counts are deliberately separate from
// durable markdown files and legacy Hermes memory files so "0 active items" can
// never be mistaken for "no memory exists".
type Inventory = provenance.Inventory
type InventoryGoncho = provenance.InventoryGoncho
type InventoryMemoryDir = provenance.InventoryMemoryDir
type InventoryDir = provenance.InventoryDir
type InventoryFile = provenance.InventoryFile

// ReadInventory inspects memory provenance metadata for one profile root. It
// does not read memory contents.
func ReadInventory(ctx context.Context, opts InventoryOptions) (Inventory, error) {
	return provenance.ReadInventory(ctx, opts)
}
