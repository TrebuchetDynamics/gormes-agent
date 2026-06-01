package diagnostics

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/diagnostics/paste"

// PasteEntry represents one pending paste deletion record.
type PasteEntry = paste.PasteEntry

// PasteDeleter defines the interface for issuing a DELETE request to a paste service.
type PasteDeleter = paste.PasteDeleter

// PasteStore defines the interface for loading and saving pending paste entries.
type PasteStore = paste.PasteStore

// InMemoryPasteStore is a hermetic in-memory implementation of PasteStore.
type InMemoryPasteStore = paste.InMemoryPasteStore

// PasteSweeper coordinates expired-paste deletion.
type PasteSweeper = paste.PasteSweeper

// SweepResult captures the outcome of one sweep pass.
type SweepResult = paste.SweepResult

// SweepError describes one delete failure with evidence.
type SweepError = paste.SweepError

// PasteURLScheme extracts the paste service name from a URL for validation.
func PasteURLScheme(url string) string { return paste.PasteURLScheme(url) }
