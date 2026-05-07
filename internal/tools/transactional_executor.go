package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type TransactionalExecutor struct {
	inner      ToolExecutor
	classifier *CommandClassifier
}

func NewTransactionalExecutor(inner ToolExecutor, classifier *CommandClassifier) *TransactionalExecutor {
	return &TransactionalExecutor{inner: inner, classifier: classifier}
}

type TransactionalResult struct {
	Success        bool
	RolledBack     bool
	SnapshotTaken  bool
	Classification string
	Audit          CommandAuditEntry
	Output         json.RawMessage
	Error          string
}

func (te *TransactionalExecutor) Execute(ctx context.Context, req ToolRequest) (TransactionalResult, error) {
	classifier := te.classifier
	if classifier == nil {
		classifier = NewCommandClassifier()
	}
	decision := classifier.ClassifyToolRequest(req)
	classification := decision.Class

	var result TransactionalResult
	result.Classification = classification.String()
	result.Audit = decision.Audit

	if te.inner == nil && classification != CommandUnsafe {
		result.Success = false
		result.Error = "tool executor is unavailable"
		return result, nil
	}

	switch classification {
	case CommandUnsafe:
		result.Success = false
		result.Error = fmt.Sprintf("blocked: command classified as unsafe")
		return result, nil
	case CommandUncertain:
		snapshot, snapshotErr := takeTransactionalWorkspaceSnapshot(req, decision)
		if snapshotErr != nil {
			result.Success = false
			result.Error = "snapshot failed: " + snapshotErr.Error()
			return result, nil
		}
		result.SnapshotTaken = snapshot != nil
		ch, err := te.inner.Execute(ctx, req)
		if err != nil {
			return rollbackTransactionalResult(result, snapshot, err), nil
		}
		for evt := range ch {
			if evt.Err != nil {
				return rollbackTransactionalResult(result, snapshot, evt.Err), nil
			}
			result.Output = evt.Output
		}
		if snapshot != nil {
			if err := snapshot.Commit(); err != nil {
				result.Success = false
				result.Error = "snapshot commit failed: " + err.Error()
				return result, nil
			}
		}
		result.Success = true
		return result, nil
	default:
		ch, err := te.inner.Execute(ctx, req)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			return result, nil
		}
		for evt := range ch {
			if evt.Err != nil {
				result.Success = false
				result.Error = evt.Err.Error()
				return result, nil
			}
			result.Output = evt.Output
		}
		result.Success = true
		return result, nil
	}
}

func takeTransactionalWorkspaceSnapshot(req ToolRequest, decision CommandDecision) (*WorkspaceSnapshot, error) {
	if !decision.RequiresSnapshot {
		return nil, nil
	}
	root := transactionalWorkspaceRoot(req)
	if root == "" {
		return nil, fmt.Errorf("workspace root metadata is required")
	}
	return TakeWorkspaceSnapshot(root)
}

func rollbackTransactionalResult(result TransactionalResult, snapshot *WorkspaceSnapshot, toolErr error) TransactionalResult {
	result.Success = false
	if snapshot == nil {
		result.Error = toolErr.Error()
		return result
	}
	if err := snapshot.Restore(); err != nil {
		result.RolledBack = false
		result.Error = fmt.Sprintf("%s; rollback failed: %v", toolErr.Error(), err)
		return result
	}
	result.RolledBack = true
	result.Error = toolErr.Error()
	return result
}

func transactionalWorkspaceRoot(req ToolRequest) string {
	for _, key := range []string{"workspace_root", "workspace", "workdir", "cwd"} {
		if value := strings.TrimSpace(req.Metadata[key]); value != "" {
			return value
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(req.Input, &payload); err != nil {
		return ""
	}
	for _, key := range []string{"workspace_root", "workspace", "workdir", "cwd"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
