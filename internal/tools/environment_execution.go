package tools

import "time"

func normalizeEnvironmentCommand(command EnvironmentCommand, defaultTimeout time.Duration) EnvironmentCommand {
	if command.Timeout == 0 && defaultTimeout > 0 {
		command.Timeout = defaultTimeout
	}
	return command
}

func recordedEnvironmentEvidence(backend string, operation EnvironmentOperationKind, resource, message string) EnvironmentEvidence {
	code := EnvironmentCommandRecorded
	switch operation {
	case EnvironmentOperationUpload:
		code = EnvironmentFileUploadRecorded
	case EnvironmentOperationDownload:
		code = EnvironmentFileDownloadRecorded
	case EnvironmentOperationCleanup:
		code = EnvironmentCleanupRecorded
	}
	return EnvironmentEvidence{
		Code:      code,
		Status:    EnvironmentStatusRecorded,
		Backend:   backend,
		Operation: string(operation),
		Resource:  resource,
		Message:   message,
	}
}

func cwdDeletedEnvironmentEvidence(backend, resource string) EnvironmentEvidence {
	return EnvironmentEvidence{
		Code:      EnvironmentTerminalCWDDeleted,
		Status:    EnvironmentStatusRecorded,
		Backend:   backend,
		Operation: string(EnvironmentOperationExecute),
		Resource:  resource,
		Message:   "cwd was deleted; resetting to a safe fallback",
	}
}

func unavailableEnvironmentError(backend string, operation EnvironmentOperationKind, reason string) error {
	if reason == "" {
		reason = "backend unavailable"
	}
	return &EnvironmentEvidenceError{Evidence: EnvironmentEvidence{
		Code:      EnvironmentBackendUnavailable,
		Status:    EnvironmentStatusUnavailable,
		Backend:   backend,
		Operation: string(operation),
		Message:   reason,
	}}
}
