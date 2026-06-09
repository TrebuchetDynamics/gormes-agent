package tokenlock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/jsonfile"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/runtimeproc"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

const TokenLockKind = "gormes-gateway-token-lock"

// TokenLockStatus is operator-facing evidence for credential-scoped gateway
// lock decisions.
type TokenLockStatus string

const (
	TokenLockStatusAcquired               TokenLockStatus = "acquired"
	TokenLockStatusHeld                   TokenLockStatus = "lock-held"
	TokenLockStatusStaleCleared           TokenLockStatus = "stale-lock-cleared"
	TokenLockStatusCredentialHashMismatch TokenLockStatus = "credential-hash-mismatch"
	TokenLockStatusReleased               TokenLockStatus = "released"
	TokenLockStatusReleaseFailed          TokenLockStatus = "lock-release-failed"
)

var (
	ErrTokenLockHeld                   = errors.New("gateway token lock held")
	ErrTokenLockCredentialHashMismatch = errors.New("gateway token lock credential hash mismatch")
	ErrTokenLockReleaseFailed          = errors.New("gateway token lock release failed")
)

// TokenLockRequest describes the external credential identity a gateway
// process wants to reserve.
type TokenLockRequest struct {
	Platform   string
	Credential string
}

// TokenLockEvidence is safe to persist in runtime status JSON. It carries only
// platform names, paths, process identity, and non-reversible credential hashes.
type TokenLockEvidence struct {
	Status            TokenLockStatus        `json:"status,omitempty"`
	Platform          string                 `json:"platform,omitempty"`
	CredentialHash    string                 `json:"credential_hash,omitempty"`
	Path              string                 `json:"path,omitempty"`
	OwnerPID          int                    `json:"owner_pid,omitempty"`
	OwnerStartTime    int64                  `json:"owner_start_time,omitempty"`
	ProcessValidation runtimeproc.Validation `json:"process_validation,omitempty"`
	Message           string                 `json:"message,omitempty"`
	UpdatedAt         string                 `json:"updated_at,omitempty"`
}

// TokenLockStore manages credential-scoped gateway lock records under one
// machine-local lock directory.
type TokenLockStore struct {
	dir        string
	now        func() time.Time
	pid        func() int
	startTime  func(int) (int64, bool)
	argv       func() []string
	processes  runtimeproc.ProcessTable
	removeFile func(string) error
}

// TokenScopedGatewayLock represents a lock record owned by the current
// process according to PID and process start-time evidence.
type TokenScopedGatewayLock struct {
	store  *TokenLockStore
	path   string
	record tokenLockRecord
}

type tokenLockRecord struct {
	Kind           string   `json:"kind"`
	Platform       string   `json:"platform"`
	CredentialHash string   `json:"credential_hash"`
	PID            int      `json:"pid"`
	StartTime      int64    `json:"start_time,omitempty"`
	Command        string   `json:"command,omitempty"`
	Argv           []string `json:"argv,omitempty"`
	UpdatedAt      string   `json:"updated_at"`
}

// NewTokenLockStore returns a JSON-file-backed token lock store.
func NewTokenLockStore(dir string) *TokenLockStore {
	return &TokenLockStore{
		dir:        dir,
		now:        func() time.Time { return time.Now().UTC() },
		pid:        os.Getpid,
		startTime:  runtimeproc.ProcessStartTime,
		argv:       func() []string { return slices.Clone(os.Args) },
		processes:  runtimeproc.ProcTable{},
		removeFile: os.Remove,
	}
}

// TokenCredentialHash returns the non-reversible credential scope hash used in
// lock filenames and status evidence.
func TokenCredentialHash(credential string) string {
	sum := sha256.Sum256([]byte(credential))
	return hex.EncodeToString(sum[:])
}

// LockPath returns the lock path for platform plus credential identity.
func (s *TokenLockStore) LockPath(platform, credential string) string {
	return filepath.Join(s.lockDir(), sanitizeTokenLockPlatform(platform)+"-"+TokenCredentialHash(credential)+".lock")
}

// Acquire claims a platform/credential lock or returns evidence explaining why
// the current process could not safely acquire it.
func (s *TokenLockStore) Acquire(ctx context.Context, req TokenLockRequest) (*TokenScopedGatewayLock, TokenLockEvidence, error) {
	if s == nil {
		s = NewTokenLockStore("")
	}
	if err := ctx.Err(); err != nil {
		return nil, TokenLockEvidence{}, err
	}

	platform := sanitizeTokenLockPlatform(req.Platform)
	hash := TokenCredentialHash(req.Credential)
	path := filepath.Join(s.lockDir(), platform+"-"+hash+".lock")
	legacyPlatform := legacyTokenLockPlatform(req.Platform)
	legacyPath := ""
	if legacyPlatform != "" && legacyPlatform != platform {
		legacyPath = filepath.Join(s.lockDir(), legacyPlatform+"-"+hash+".lock")
	}
	record := s.currentRecord(platform, hash)
	evidenceStatus := TokenLockStatusAcquired
	var staleValidation runtimeproc.Validation

	for attempt := 0; attempt < 2; attempt++ {
		activePath := path
		existing, err := readTokenLockRecord(activePath)
		if errors.Is(err, os.ErrNotExist) && legacyPath != "" {
			if legacyExisting, legacyErr := readTokenLockRecord(legacyPath); !errors.Is(legacyErr, os.ErrNotExist) {
				activePath = legacyPath
				existing = legacyExisting
				err = legacyErr
			}
		}
		switch {
		case errors.Is(err, os.ErrNotExist):
			lock, evidence, err := s.createLock(ctx, path, record, evidenceStatus, staleValidation)
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return lock, evidence, err
		case err != nil:
			return nil, s.evidence(record, activePath, TokenLockStatusHeld, runtimeproc.Validation{}, err.Error()), err
		}

		if !tokenLockRecordMatchesPlatform(existing.Platform, platform, legacyPlatform) || existing.CredentialHash != hash {
			evidence := s.evidence(record, activePath, TokenLockStatusCredentialHashMismatch, runtimeproc.Validation{}, "lock record identity does not match requested platform and credential hash")
			evidence.OwnerPID = existing.PID
			evidence.OwnerStartTime = existing.StartTime
			return nil, evidence, ErrTokenLockCredentialHashMismatch
		}

		if s.ownsRecord(existing) {
			if err := writeTokenLockRecordAtomic(activePath, record); err != nil {
				return nil, s.evidence(record, activePath, TokenLockStatusHeld, runtimeproc.Validation{}, err.Error()), err
			}
			lock := &TokenScopedGatewayLock{store: s, path: activePath, record: record}
			return lock, s.evidence(record, activePath, TokenLockStatusAcquired, runtimeproc.Validation{}, ""), nil
		}

		validation := s.validateTokenLockOwner(existing)
		if !tokenLockValidationProvesGone(validation) {
			evidence := s.evidence(record, activePath, TokenLockStatusHeld, validation, "credential lock is held by a live or unverified process")
			evidence.OwnerPID = existing.PID
			evidence.OwnerStartTime = existing.StartTime
			return nil, evidence, fmt.Errorf("%w: %s", ErrTokenLockHeld, path)
		}

		if err := s.remove(activePath); err != nil {
			evidence := s.evidence(record, activePath, TokenLockStatusHeld, validation, "stale credential lock could not be cleared: "+err.Error())
			evidence.OwnerPID = existing.PID
			evidence.OwnerStartTime = existing.StartTime
			return nil, evidence, fmt.Errorf("%w: %v", ErrTokenLockHeld, err)
		}
		evidenceStatus = TokenLockStatusStaleCleared
		staleValidation = validation
	}

	evidence := s.evidence(record, path, TokenLockStatusHeld, staleValidation, "credential lock changed while acquiring")
	return nil, evidence, fmt.Errorf("%w: %s", ErrTokenLockHeld, path)
}

// Path returns the filesystem path of the acquired lock.
func (l *TokenScopedGatewayLock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// CredentialHash returns the lock's credential hash.
func (l *TokenScopedGatewayLock) CredentialHash() string {
	if l == nil {
		return ""
	}
	return l.record.CredentialHash
}

// Release removes this lock only when the on-disk record still belongs to the
// current process identity.
func (l *TokenScopedGatewayLock) Release(ctx context.Context) (TokenLockEvidence, error) {
	if l == nil || l.store == nil || l.path == "" {
		return TokenLockEvidence{}, nil
	}
	if err := ctx.Err(); err != nil {
		return TokenLockEvidence{}, err
	}

	record := l.store.currentRecord(l.record.Platform, l.record.CredentialHash)
	evidence := l.store.evidence(record, l.path, TokenLockStatusReleased, runtimeproc.Validation{}, "")
	existing, err := readTokenLockRecord(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return evidence, nil
	}
	if err != nil {
		evidence.Status = TokenLockStatusReleaseFailed
		evidence.Message = err.Error()
		return evidence, fmt.Errorf("%w: %v", ErrTokenLockReleaseFailed, err)
	}
	if existing.Platform != l.record.Platform ||
		existing.CredentialHash != l.record.CredentialHash ||
		existing.PID != record.PID ||
		existing.StartTime != record.StartTime {
		evidence.OwnerPID = existing.PID
		evidence.OwnerStartTime = existing.StartTime
		evidence.Message = "lock is no longer owned by this process"
		return evidence, nil
	}
	if err := l.store.remove(l.path); err != nil {
		evidence.Status = TokenLockStatusReleaseFailed
		evidence.OwnerPID = existing.PID
		evidence.OwnerStartTime = existing.StartTime
		evidence.Message = err.Error()
		return evidence, fmt.Errorf("%w: %v", ErrTokenLockReleaseFailed, err)
	}
	return evidence, nil
}

func (s *TokenLockStore) lockDir() string {
	if s == nil || strings.TrimSpace(s.dir) == "" {
		return filepath.Join(".", "gateway-locks")
	}
	return s.dir
}

func (s *TokenLockStore) currentRecord(platform, credentialHash string) tokenLockRecord {
	pid := s.pid()
	startTime, _ := s.startTime(pid)
	argv := sanitizeTokenLockArgv(s.argv())
	return tokenLockRecord{
		Kind:           TokenLockKind,
		Platform:       platform,
		CredentialHash: credentialHash,
		PID:            pid,
		StartTime:      startTime,
		Command:        strings.Join(argv, " "),
		Argv:           argv,
		UpdatedAt:      s.now().UTC().Format(time.RFC3339Nano),
	}
}

func sanitizeTokenLockArgv(argv []string) []string {
	out := slices.Clone(argv)
	for i, arg := range out {
		out[i] = sanitizeTokenLockArg(arg)
	}
	return out
}

func sanitizeTokenLockArg(arg string) string {
	arg = strings.Join(strings.Fields(arg), " ")
	arg = redaction.RedactSecrets(arg)
	lower := strings.ToLower(arg)
	if strings.Contains(lower, "[redacted]") && (strings.Contains(lower, "api-key") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password")) {
		return "[redacted]"
	}
	return arg
}

func sanitizeTokenLockCommand(command string) string {
	parts := strings.Fields(command)
	for i, part := range parts {
		parts[i] = sanitizeTokenLockArg(part)
	}
	return strings.Join(parts, " ")
}

func (s *TokenLockStore) ownsRecord(record tokenLockRecord) bool {
	current := s.currentRecord(record.Platform, record.CredentialHash)
	if record.PID <= 0 || record.PID != current.PID {
		return false
	}
	if record.StartTime == 0 || current.StartTime == 0 {
		return record.StartTime == current.StartTime
	}
	return record.StartTime == current.StartTime
}

func (s *TokenLockStore) createLock(ctx context.Context, path string, record tokenLockRecord, status TokenLockStatus, validation runtimeproc.Validation) (*TokenScopedGatewayLock, TokenLockEvidence, error) {
	if err := ctx.Err(); err != nil {
		return nil, TokenLockEvidence{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, s.evidence(record, path, TokenLockStatusHeld, validation, err.Error()), err
	}
	raw, err := jsonfile.MarshalIndentNewline(record)
	if err != nil {
		return nil, s.evidence(record, path, TokenLockStatusHeld, validation, err.Error()), err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, TokenLockEvidence{}, os.ErrExist
		}
		return nil, s.evidence(record, path, TokenLockStatusHeld, validation, err.Error()), err
	}
	tmpClosed := false
	defer func() {
		if !tmpClosed {
			_ = file.Close()
			_ = s.remove(path)
		}
	}()
	if _, err := file.Write(raw); err != nil {
		return nil, s.evidence(record, path, TokenLockStatusHeld, validation, err.Error()), err
	}
	if err := file.Close(); err != nil {
		return nil, s.evidence(record, path, TokenLockStatusHeld, validation, err.Error()), err
	}
	tmpClosed = true
	lock := &TokenScopedGatewayLock{store: s, path: path, record: record}
	return lock, s.evidence(record, path, status, validation, ""), nil
}

func (s *TokenLockStore) validateTokenLockOwner(record tokenLockRecord) runtimeproc.Validation {
	checkedAt := s.now().UTC().Format(time.RFC3339Nano)
	validation := runtimeproc.Validation{
		PID:               record.PID,
		ExpectedStartTime: record.StartTime,
		Command:           sanitizeTokenLockCommand(record.Command),
		CheckedAt:         checkedAt,
	}
	if record.PID <= 0 {
		validation.Status = runtimeproc.ValidationStalePID
		validation.Message = "token lock PID is missing or invalid"
		return validation
	}

	processes := s.processes
	if processes == nil {
		processes = runtimeproc.ProcTable{}
	}
	process, err := processes.LookupRuntimeProcess(record.PID)
	if err != nil {
		switch {
		case errors.Is(err, runtimeproc.ErrPermissionDenied):
			validation.Status = runtimeproc.ValidationPermissionDenied
			validation.Message = "process lookup was denied"
		case errors.Is(err, runtimeproc.ErrNotFound):
			validation.Status = runtimeproc.ValidationStalePID
			validation.Message = "process is not running"
		default:
			validation.Status = runtimeproc.ValidationStalePID
			validation.Message = err.Error()
		}
		return validation
	}

	validation.ActualStartTime = process.StartTime
	if process.Stopped {
		validation.Status = runtimeproc.ValidationStopped
		validation.Message = "process is stopped"
		return validation
	}
	if record.StartTime == 0 || process.StartTime == 0 {
		validation.Status = runtimeproc.ValidationLive
		validation.Live = true
		validation.Message = "process exists but start time could not be validated"
		return validation
	}
	if process.StartTime != record.StartTime {
		validation.Status = runtimeproc.ValidationPIDReused
		validation.Message = "process start time does not match token lock"
		return validation
	}
	validation.Status = runtimeproc.ValidationLive
	validation.Live = true
	if validation.Command == "" {
		validation.Command = sanitizeTokenLockCommand(process.Command)
	}
	return validation
}

func tokenLockValidationProvesGone(validation runtimeproc.Validation) bool {
	if validation.Live {
		return false
	}
	switch validation.Status {
	case runtimeproc.ValidationStalePID, runtimeproc.ValidationPIDReused, runtimeproc.ValidationStopped:
		return true
	default:
		return false
	}
}

func (s *TokenLockStore) evidence(record tokenLockRecord, path string, status TokenLockStatus, validation runtimeproc.Validation, message string) TokenLockEvidence {
	evidence := TokenLockEvidence{
		Status:            status,
		Platform:          record.Platform,
		CredentialHash:    record.CredentialHash,
		Path:              path,
		OwnerPID:          record.PID,
		OwnerStartTime:    record.StartTime,
		ProcessValidation: validation,
		Message:           message,
		UpdatedAt:         s.now().UTC().Format(time.RFC3339Nano),
	}
	if evidence.ProcessValidation.Status == "" {
		evidence.ProcessValidation = runtimeproc.Validation{}
	}
	return evidence
}

func (s *TokenLockStore) remove(path string) error {
	if s != nil && s.removeFile != nil {
		return s.removeFile(path)
	}
	return os.Remove(path)
}

func readTokenLockRecord(path string) (tokenLockRecord, error) {
	var record tokenLockRecord
	exists, err := jsonfile.Read(context.Background(), path, &record, "token lock record")
	if errors.Is(err, jsonfile.ErrEmpty) {
		return tokenLockRecord{}, os.ErrNotExist
	}
	if err != nil {
		if jsonfile.IsReadError(err) || !exists {
			return tokenLockRecord{}, err
		}
		return tokenLockRecord{}, fmt.Errorf("decode token lock record: %w", err)
	}
	if !exists {
		return tokenLockRecord{}, os.ErrNotExist
	}
	return record, nil
}

func writeTokenLockRecordAtomic(path string, record tokenLockRecord) error {
	return jsonfile.WriteAtomicWithOptions(context.Background(), path, record, "token lock record", jsonfile.WriteOptions{
		DirMode:    0o755,
		TmpPattern: ".token-lock-*.tmp",
	})
}

func tokenLockRecordMatchesPlatform(recordPlatform, platform, legacyPlatform string) bool {
	if recordPlatform == platform {
		return true
	}
	return legacyPlatform != "" && recordPlatform == legacyPlatform
}

func legacyTokenLockPlatform(platform string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range platform {
		allowed := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
		if allowed {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "unknown"
	}
	return out
}

func sanitizeTokenLockPlatform(platform string) string {
	normalized := strings.ToLower(strings.TrimSpace(platform))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range normalized {
		allowed := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
		if allowed {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		out = "unknown"
	}
	if normalized != "" && out != normalized {
		out += "_" + tokenLockPlatformHash(normalized)
	}
	return out
}

func tokenLockPlatformHash(platform string) string {
	sum := sha256.Sum256([]byte(platform))
	return hex.EncodeToString(sum[:])[:12]
}
