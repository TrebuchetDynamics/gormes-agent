package transcript

import "github.com/TrebuchetDynamics/gormes-agent/internal/transcript/contextrefs"

const ContextReferenceStatusPending = contextrefs.StatusPending

type ContextReferenceRecord = contextrefs.Record

type ContextReferenceHandle = contextrefs.Handle

type ContextReferenceStore = contextrefs.Store

func NewContextReferenceStore() *ContextReferenceStore {
	return contextrefs.NewStore()
}
