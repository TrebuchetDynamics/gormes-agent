package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type EnvironmentStatus string

const (
	EnvironmentStatusRecorded    EnvironmentStatus = "recorded"
	EnvironmentStatusUnavailable EnvironmentStatus = "unavailable"
)

const (
	EnvironmentCommandRecorded      = "environment_command_recorded"
	EnvironmentFileUploadRecorded   = "environment_file_upload_recorded"
	EnvironmentFileDownloadRecorded = "environment_file_download_recorded"
	EnvironmentCleanupRecorded      = "environment_cleanup_recorded"
	EnvironmentBackendUnavailable   = "environment_backend_unavailable"
)

type EnvironmentEvidence struct {
	Code      string
	Status    EnvironmentStatus
	Backend   string
	Operation string
	Resource  string
	Message   string
}

type EnvironmentEvidenceError struct {
	Evidence EnvironmentEvidence
}

func (e *EnvironmentEvidenceError) Error() string {
	if e == nil {
		return "environment evidence error"
	}
	msg := e.Evidence.Message
	if msg == "" {
		msg = string(e.Evidence.Status)
	}
	if e.Evidence.Backend == "" {
		return fmt.Sprintf("environment %s: %s", e.Evidence.Operation, msg)
	}
	return fmt.Sprintf("environment %s %s: %s", e.Evidence.Backend, e.Evidence.Operation, msg)
}

func EnvironmentEvidenceFromError(err error) (EnvironmentEvidence, bool) {
	var evidenceErr *EnvironmentEvidenceError
	if errors.As(err, &evidenceErr) && evidenceErr != nil {
		return evidenceErr.Evidence, true
	}
	return EnvironmentEvidence{}, false
}

type EnvironmentPathMapper struct {
	HostRoot        string
	EnvironmentRoot string
}

func NewEnvironmentPathMapper(hostRoot, environmentRoot string) EnvironmentPathMapper {
	if abs, err := filepath.Abs(hostRoot); err == nil {
		hostRoot = abs
	}
	return EnvironmentPathMapper{
		HostRoot:        filepath.Clean(hostRoot),
		EnvironmentRoot: normalizeEnvironmentPath(environmentRoot),
	}
}

func (m EnvironmentPathMapper) Map(hostPath string) (string, error) {
	hostPath, err := normalizeHostPath(hostPath)
	if err != nil {
		return "", err
	}
	hostRoot := m.HostRoot
	if hostRoot == "" {
		return "", errors.New("tools: environment host root is empty")
	}
	if abs, err := filepath.Abs(hostRoot); err == nil {
		hostRoot = filepath.Clean(abs)
	}
	rel, err := filepath.Rel(hostRoot, hostPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("tools: host path %q is outside environment root %q", hostPath, hostRoot)
	}
	if rel == "." {
		return m.EnvironmentRoot, nil
	}
	return path.Join(m.EnvironmentRoot, filepath.ToSlash(rel)), nil
}

type FileSyncDirection string

const (
	FileSyncUpload   FileSyncDirection = "upload"
	FileSyncDownload FileSyncDirection = "download"
	FileSyncDelete   FileSyncDirection = "delete"
)

type FileSyncIntent struct {
	Direction       FileSyncDirection
	HostPath        string
	EnvironmentPath string
	Checksum        string
}

type FileSyncResult struct {
	Intent   FileSyncIntent
	Evidence EnvironmentEvidence
}

type HostFileSnapshot struct {
	HostPath string
	Checksum string
	Size     int64
}

type FileSyncState struct {
	ChecksumsByEnvironmentPath map[string]string
}

type FileSyncPlan struct {
	Intents []FileSyncIntent
}

func BuildFileSyncPlan(previous FileSyncState, current []HostFileSnapshot, mapper EnvironmentPathMapper) (FileSyncPlan, error) {
	seen := make(map[string]struct{}, len(current))
	intents := make([]FileSyncIntent, 0)
	for _, snapshot := range current {
		hostPath, err := normalizeHostPath(snapshot.HostPath)
		if err != nil {
			return FileSyncPlan{}, err
		}
		environmentPath, err := mapper.Map(hostPath)
		if err != nil {
			return FileSyncPlan{}, err
		}
		seen[environmentPath] = struct{}{}
		if previous.ChecksumsByEnvironmentPath[environmentPath] == snapshot.Checksum {
			continue
		}
		intents = append(intents, FileSyncIntent{
			Direction:       FileSyncUpload,
			HostPath:        hostPath,
			EnvironmentPath: environmentPath,
			Checksum:        snapshot.Checksum,
		})
	}

	deletes := make([]string, 0)
	for environmentPath := range previous.ChecksumsByEnvironmentPath {
		if _, ok := seen[environmentPath]; !ok {
			deletes = append(deletes, environmentPath)
		}
	}
	sort.Strings(deletes)
	for _, environmentPath := range deletes {
		intents = append(intents, FileSyncIntent{
			Direction:       FileSyncDelete,
			EnvironmentPath: normalizeEnvironmentPath(environmentPath),
			Checksum:        previous.ChecksumsByEnvironmentPath[environmentPath],
		})
	}

	return FileSyncPlan{Intents: intents}, nil
}

type EnvironmentCommand struct {
	Command    string
	WorkingDir string
	Timeout    time.Duration
	Stdin      string
}

type EnvironmentResult struct {
	Command  EnvironmentCommand
	Output   string
	ExitCode int
	Evidence []EnvironmentEvidence
}

type EnvironmentCleanupResult struct {
	Evidence []EnvironmentEvidence
}

type EnvironmentOperationKind string

const (
	EnvironmentOperationMapPath  EnvironmentOperationKind = "map_path"
	EnvironmentOperationUpload   EnvironmentOperationKind = "upload"
	EnvironmentOperationDownload EnvironmentOperationKind = "download"
	EnvironmentOperationExecute  EnvironmentOperationKind = "execute"
	EnvironmentOperationCleanup  EnvironmentOperationKind = "cleanup"
)

type EnvironmentOperation struct {
	Kind            EnvironmentOperationKind
	Backend         string
	HostPath        string
	EnvironmentPath string
	Command         string
	Timeout         time.Duration
}

type Environment interface {
	MapPath(hostPath string) (string, error)
	Upload(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error)
	Download(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error)
	Execute(ctx context.Context, command EnvironmentCommand) (EnvironmentResult, error)
	Cleanup(ctx context.Context) (EnvironmentCleanupResult, error)
}

type FakeEnvironment struct {
	backend          string
	mapper           EnvironmentPathMapper
	cleanupResources []string
	operations       []EnvironmentOperation
}

func NewFakeEnvironment(backend string, mapper EnvironmentPathMapper) *FakeEnvironment {
	if backend == "" {
		backend = "fake"
	}
	return &FakeEnvironment{backend: backend, mapper: mapper}
}

func (e *FakeEnvironment) RegisterCleanup(resource string) {
	e.cleanupResources = append(e.cleanupResources, resource)
}

func (e *FakeEnvironment) MapPath(hostPath string) (string, error) {
	environmentPath, err := e.mapper.Map(hostPath)
	if err != nil {
		return "", err
	}
	e.record(EnvironmentOperation{
		Kind:            EnvironmentOperationMapPath,
		Backend:         e.backend,
		HostPath:        filepath.Clean(hostPath),
		EnvironmentPath: environmentPath,
	})
	return environmentPath, nil
}

func (e *FakeEnvironment) Upload(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}
	intent.Direction = FileSyncUpload
	intent.HostPath = filepath.Clean(intent.HostPath)
	intent.EnvironmentPath = normalizeEnvironmentPath(intent.EnvironmentPath)
	e.record(EnvironmentOperation{
		Kind:            EnvironmentOperationUpload,
		Backend:         e.backend,
		HostPath:        intent.HostPath,
		EnvironmentPath: intent.EnvironmentPath,
	})
	return FileSyncResult{Intent: intent, Evidence: e.evidence(EnvironmentFileUploadRecorded, "upload", intent.EnvironmentPath)}, nil
}

func (e *FakeEnvironment) Download(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}
	intent.Direction = FileSyncDownload
	intent.HostPath = filepath.Clean(intent.HostPath)
	intent.EnvironmentPath = normalizeEnvironmentPath(intent.EnvironmentPath)
	e.record(EnvironmentOperation{
		Kind:            EnvironmentOperationDownload,
		Backend:         e.backend,
		HostPath:        intent.HostPath,
		EnvironmentPath: intent.EnvironmentPath,
	})
	return FileSyncResult{Intent: intent, Evidence: e.evidence(EnvironmentFileDownloadRecorded, "download", intent.EnvironmentPath)}, nil
}

func (e *FakeEnvironment) Execute(ctx context.Context, command EnvironmentCommand) (EnvironmentResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentResult{}, err
	}
	e.record(EnvironmentOperation{
		Kind:            EnvironmentOperationExecute,
		Backend:         e.backend,
		Command:         command.Command,
		Timeout:         command.Timeout,
		HostPath:        command.WorkingDir,
		EnvironmentPath: command.WorkingDir,
	})
	return EnvironmentResult{
		Command:  command,
		Output:   "fake environment recorded command: " + command.Command,
		ExitCode: 0,
		Evidence: []EnvironmentEvidence{e.evidence(EnvironmentCommandRecorded, "execute", command.Command)},
	}, nil
}

func (e *FakeEnvironment) Cleanup(ctx context.Context) (EnvironmentCleanupResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentCleanupResult{}, err
	}
	e.record(EnvironmentOperation{Kind: EnvironmentOperationCleanup, Backend: e.backend})
	evidence := make([]EnvironmentEvidence, 0, len(e.cleanupResources))
	for i := len(e.cleanupResources) - 1; i >= 0; i-- {
		evidence = append(evidence, e.evidence(EnvironmentCleanupRecorded, "cleanup", e.cleanupResources[i]))
	}
	return EnvironmentCleanupResult{Evidence: evidence}, nil
}

func (e *FakeEnvironment) Operations() []EnvironmentOperation {
	out := make([]EnvironmentOperation, len(e.operations))
	copy(out, e.operations)
	return out
}

func (e *FakeEnvironment) record(op EnvironmentOperation) {
	e.operations = append(e.operations, op)
}

func (e *FakeEnvironment) evidence(code, operation, resource string) EnvironmentEvidence {
	return EnvironmentEvidence{
		Code:      code,
		Status:    EnvironmentStatusRecorded,
		Backend:   e.backend,
		Operation: operation,
		Resource:  resource,
		Message:   "fake environment recorded " + operation,
	}
}

type UnsupportedEnvironment struct {
	Backend string
	Reason  string
}

func (e UnsupportedEnvironment) MapPath(hostPath string) (string, error) {
	return "", e.unavailable("map_path")
}

func (e UnsupportedEnvironment) Upload(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}
	return FileSyncResult{}, e.unavailable("upload")
}

func (e UnsupportedEnvironment) Download(ctx context.Context, intent FileSyncIntent) (FileSyncResult, error) {
	if err := ctx.Err(); err != nil {
		return FileSyncResult{}, err
	}
	return FileSyncResult{}, e.unavailable("download")
}

func (e UnsupportedEnvironment) Execute(ctx context.Context, command EnvironmentCommand) (EnvironmentResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentResult{}, err
	}
	return EnvironmentResult{}, e.unavailable("execute")
}

func (e UnsupportedEnvironment) Cleanup(ctx context.Context) (EnvironmentCleanupResult, error) {
	if err := ctx.Err(); err != nil {
		return EnvironmentCleanupResult{}, err
	}
	return EnvironmentCleanupResult{}, e.unavailable("cleanup")
}

func (e UnsupportedEnvironment) unavailable(operation string) error {
	reason := e.Reason
	if reason == "" {
		reason = "backend unavailable"
	}
	return &EnvironmentEvidenceError{Evidence: EnvironmentEvidence{
		Code:      EnvironmentBackendUnavailable,
		Status:    EnvironmentStatusUnavailable,
		Backend:   e.Backend,
		Operation: operation,
		Message:   reason,
	}}
}

func ChecksumBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeHostPath(hostPath string) (string, error) {
	if hostPath == "" {
		return "", errors.New("tools: host path is empty")
	}
	abs, err := filepath.Abs(hostPath)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func normalizeEnvironmentPath(environmentPath string) string {
	p := filepath.ToSlash(environmentPath)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}
