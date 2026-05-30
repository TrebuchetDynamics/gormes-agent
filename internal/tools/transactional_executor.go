package tools

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/transaction"
)

type TransactionalResult = transaction.Result

type TransactionalExecutor struct {
	inner *transaction.Executor
}

func NewTransactionalExecutor(inner ToolExecutor, classifier *CommandClassifier) *TransactionalExecutor {
	return &TransactionalExecutor{inner: transaction.NewExecutor(inner, classifier.ensure())}
}

func (te *TransactionalExecutor) Execute(ctx context.Context, req ToolRequest) (TransactionalResult, error) {
	return te.inner.Execute(ctx, req)
}
