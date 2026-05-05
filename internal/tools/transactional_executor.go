package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type TransactionalExecutor struct {
	inner      *InProcessToolExecutor
	classifier *CommandClassifier
}

func NewTransactionalExecutor(inner *InProcessToolExecutor, classifier *CommandClassifier) *TransactionalExecutor {
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

	switch classification {
	case CommandUnsafe:
		result.Success = false
		result.Error = fmt.Sprintf("blocked: command classified as unsafe")
		return result, nil
	case CommandUncertain:
		result.SnapshotTaken = decision.RequiresSnapshot
		ch, err := te.inner.Execute(ctx, req)
		if err != nil {
			result.Success = false
			result.RolledBack = true
			result.Error = err.Error()
			return result, nil
		}
		for evt := range ch {
			if evt.Err != nil {
				result.Success = false
				result.RolledBack = true
				result.Error = evt.Err.Error()
				return result, nil
			}
			result.Output = evt.Output
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
