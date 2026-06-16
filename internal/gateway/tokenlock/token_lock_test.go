package tokenlock

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/runtimeproc"
)

func TestTokenScopedGatewayLockAcquireAndReleaseAllowNilContext(t *testing.T) {
	dir := t.TempDir()
	store := newTokenLockTestStore(t, dir, 1001, 501, nil)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("token lock panicked with nil context: %v", r)
		}
	}()

	lock, evidence, err := store.Acquire(nil, TokenLockRequest{Platform: "telegram", Credential: "shared-token"})
	if err != nil {
		t.Fatalf("Acquire nil context: %v", err)
	}
	if lock == nil || evidence.Status != TokenLockStatusAcquired {
		t.Fatalf("Acquire nil context lock=%v evidence=%+v, want acquired", lock, evidence)
	}
	if releaseEvidence, err := lock.Release(nil); err != nil || releaseEvidence.Status != TokenLockStatusReleased {
		t.Fatalf("Release nil context evidence=%+v err=%v, want released", releaseEvidence, err)
	}
}

func TestReadTokenLockRecordPropagatesUnreadablePath(t *testing.T) {
	record, err := readTokenLockRecord(t.TempDir())
	if err == nil {
		t.Fatalf("readTokenLockRecord err = nil record=%+v, want read error for directory lock path", record)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("readTokenLockRecord err = %v, must not mask unreadable lock as missing", err)
	}
}

func TestTokenScopedGatewayLockRejectsMissingKindWithoutClearingFile(t *testing.T) {
	dir := t.TempDir()
	credential := "shared-token"
	path := filepath.Join(dir, "telegram-"+TokenCredentialHash(credential)+".lock")
	writeTokenLockJSONFixture(t, path, map[string]any{
		"platform":        "telegram",
		"credential_hash": TokenCredentialHash(credential),
		"pid":             1001,
		"start_time":      501,
		"updated_at":      "2026-04-25T17:00:00Z",
	})

	store := newTokenLockTestStore(t, dir, 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 501, command: "gormes gateway", stopped: true},
	})
	lock, evidence, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram", Credential: credential})
	if err == nil || lock != nil {
		t.Fatalf("Acquire missing-kind lock err=%v lock=%v evidence=%+v, want blocked without lock", err, lock, evidence)
	}
	record := readTokenLockRecordFixture(t, path)
	if record.Kind != "" || record.PID != 1001 {
		t.Fatalf("missing-kind lock was overwritten/cleared: %+v", record)
	}
}

func TestTokenScopedGatewayLockClearsStaleEmptyLock(t *testing.T) {
	dir := t.TempDir()
	credential := "shared-token"
	path := filepath.Join(dir, "telegram-"+TokenCredentialHash(credential)+".lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir lock dir: %v", err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write empty lock: %v", err)
	}
	store := newTokenLockTestStore(t, dir, 2002, 602, nil)
	old := store.now().Add(-10 * time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("age empty lock: %v", err)
	}
	lock, evidence, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram", Credential: credential})
	if err != nil || lock == nil {
		t.Fatalf("Acquire stale empty lock err=%v lock=%v evidence=%+v, want acquired", err, lock, evidence)
	}
	if evidence.Status != TokenLockStatusStaleCleared {
		t.Fatalf("evidence status = %q, want stale-lock-cleared", evidence.Status)
	}
	record := readTokenLockRecordFixture(t, path)
	if record.Kind != TokenLockKind || record.PID != 2002 {
		t.Fatalf("stale empty lock replacement = %+v, want current process record", record)
	}
}

func TestTokenScopedGatewayLockBlocksFreshEmptyLock(t *testing.T) {
	dir := t.TempDir()
	credential := "shared-token"
	path := filepath.Join(dir, "telegram-"+TokenCredentialHash(credential)+".lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir lock dir: %v", err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write empty lock: %v", err)
	}

	store := newTokenLockTestStore(t, dir, 2002, 602, nil)
	lock, evidence, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram", Credential: credential})
	if !errors.Is(err, ErrTokenLockHeld) || lock != nil {
		t.Fatalf("Acquire fresh empty lock err=%v lock=%v evidence=%+v, want held", err, lock, evidence)
	}
	if evidence.ProcessValidation.Status != runtimeproc.ValidationLive {
		t.Fatalf("fresh empty lock validation = %+v, want live/in-progress", evidence.ProcessValidation)
	}
	info, statErr := os.Stat(path)
	if statErr != nil || info.Size() != 0 {
		t.Fatalf("fresh empty lock was modified stat=%+v err=%v", info, statErr)
	}
}

func TestTokenScopedGatewayLockRedactsExistingOwnerCommandEvidence(t *testing.T) {
	dir := t.TempDir()
	credential := "shared-token"
	path := filepath.Join(dir, "telegram-"+TokenCredentialHash(credential)+".lock")
	writeTokenLockJSONFixture(t, path, map[string]any{
		"kind":            "gormes-gateway-token-lock",
		"platform":        "telegram",
		"credential_hash": TokenCredentialHash(credential),
		"pid":             1001,
		"start_time":      501,
		"command":         "gormes gateway --api-key=plain-secret-token",
		"updated_at":      "2026-04-25T17:00:00Z",
	})

	store := newTokenLockTestStore(t, dir, 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 501, command: "gormes gateway --api-key=plain-secret-token"},
	})
	_, evidence, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram", Credential: credential})
	if !errors.Is(err, ErrTokenLockHeld) {
		t.Fatalf("Acquire err = %v, want ErrTokenLockHeld", err)
	}
	for _, forbidden := range []string{"plain-secret-token", "api-key"} {
		if strings.Contains(evidence.ProcessValidation.Command, forbidden) {
			t.Fatalf("process validation command leaked %q in %+v", forbidden, evidence.ProcessValidation)
		}
	}
	if evidence.ProcessValidation.Command != "gormes gateway [redacted]" {
		t.Fatalf("process validation command = %q, want redacted command", evidence.ProcessValidation.Command)
	}
}

func TestTokenScopedGatewayLockHonorsLegacySanitizedPlatformLock(t *testing.T) {
	dir := t.TempDir()
	credential := "shared-token"
	legacyPath := filepath.Join(dir, "telegram_work-"+TokenCredentialHash(credential)+".lock")
	writeTokenLockJSONFixture(t, legacyPath, map[string]any{
		"kind":            "gormes-gateway-token-lock",
		"platform":        "telegram_work",
		"credential_hash": TokenCredentialHash(credential),
		"pid":             1001,
		"start_time":      501,
		"updated_at":      "2026-04-25T17:00:00Z",
	})

	store := newTokenLockTestStore(t, dir, 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 501, command: "gormes gateway"},
	})
	_, evidence, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram:work", Credential: credential})
	if !errors.Is(err, ErrTokenLockHeld) {
		t.Fatalf("Acquire err = %v, want ErrTokenLockHeld for live legacy lock", err)
	}
	if evidence.Path != legacyPath || evidence.OwnerPID != 1001 {
		t.Fatalf("evidence = %+v, want legacy path held by pid 1001", evidence)
	}
}

func TestTokenScopedGatewayLockPathDisambiguatesSanitizedPlatformCollisions(t *testing.T) {
	dir := t.TempDir()
	store := newTokenLockTestStore(t, dir, 1001, 501, nil)
	credential := "shared-token"

	colonPath := store.LockPath("telegram:work", credential)
	underscorePath := store.LockPath("telegram_work", credential)
	if colonPath == underscorePath {
		t.Fatalf("LockPath collision for sanitized platform names: %q", colonPath)
	}
	if !strings.Contains(filepath.Base(underscorePath), "telegram_work-") {
		t.Fatalf("safe platform path = %q, want readable platform prefix", underscorePath)
	}
	if !strings.Contains(filepath.Base(colonPath), "telegram_work_") {
		t.Fatalf("escaped platform path = %q, want readable sanitized prefix plus disambiguator", colonPath)
	}
}

func TestTokenScopedGatewayLockRedactsSplitAuthorizationArgvSecrets(t *testing.T) {
	dir := t.TempDir()
	store := newTokenLockTestStore(t, dir, 1001, 501, nil)
	store.argv = func() []string { return []string{"gormes", "gateway", "--authorization", "Bearer plain-secret-token"} }

	lock, _, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram", Credential: "shared-token"})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	raw, err := os.ReadFile(lock.Path())
	if err != nil {
		t.Fatalf("read lock record: %v", err)
	}
	for _, forbidden := range []string{"plain-secret-token", "authorization", "Bearer", "bearer"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("lock record leaked split authorization argv %q:\n%s", forbidden, raw)
		}
	}
	if !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("lock record missing redacted argv evidence:\n%s", raw)
	}
}

func TestTokenScopedGatewayLockRemovesHiddenFormattingArgv(t *testing.T) {
	dir := t.TempDir()
	store := newTokenLockTestStore(t, dir, 1001, 501, nil)
	store.argv = func() []string { return []string{"gormes", "gateway", "--profile", "evil\u202egnp"} }

	lock, _, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram", Credential: "shared-token"})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	raw, err := os.ReadFile(lock.Path())
	if err != nil {
		t.Fatalf("read lock record: %v", err)
	}
	if strings.Contains(string(raw), "\u202e") {
		t.Fatalf("lock record leaked hidden formatting rune:\n%s", raw)
	}
	if !strings.Contains(string(raw), "evilgnp") {
		t.Fatalf("lock record missing sanitized argv text:\n%s", raw)
	}
}

func TestTokenScopedGatewayLockRedactsArgvSecrets(t *testing.T) {
	dir := t.TempDir()
	store := newTokenLockTestStore(t, dir, 1001, 501, nil)
	store.argv = func() []string { return []string{"gormes", "gateway", "--api-key=plain-secret-token"} }

	lock, _, err := store.Acquire(context.Background(), TokenLockRequest{Platform: "telegram", Credential: "shared-token"})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	raw, err := os.ReadFile(lock.Path())
	if err != nil {
		t.Fatalf("read lock record: %v", err)
	}
	for _, forbidden := range []string{"plain-secret-token", "api-key"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("lock record leaked argv secret %q:\n%s", forbidden, raw)
		}
	}
	if !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("lock record missing redacted argv evidence:\n%s", raw)
	}
}

func TestTokenScopedGatewayLockPathUsesGormesHomeHash(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "xdg-state"))
	credential := "123456:ABC-raw-token"
	store := newTokenLockTestStore(t, config.GatewayLockDir(), 1001, 501, nil)

	lock, evidence, err := store.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: credential,
	})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	hash := TokenCredentialHash(credential)
	if len(hash) != 64 {
		t.Fatalf("TokenCredentialHash length = %d, want full sha256 hex", len(hash))
	}
	wantPath := filepath.Join(gormesHome, "runtime", "gateway-locks", "telegram-"+hash+".lock")
	if lock.Path() != wantPath {
		t.Fatalf("lock path = %q, want %q", lock.Path(), wantPath)
	}
	if evidence.CredentialHash != hash || evidence.Platform != "telegram" {
		t.Fatalf("evidence = %+v, want platform telegram and credential hash %s", evidence, hash)
	}
	raw, err := os.ReadFile(lock.Path())
	if err != nil {
		t.Fatalf("read lock record: %v", err)
	}
	for _, leak := range []string{credential, "123456", "ABC-raw-token"} {
		if strings.Contains(lock.Path(), leak) {
			t.Fatalf("lock path %q leaks credential material %q", lock.Path(), leak)
		}
		if strings.Contains(string(raw), leak) {
			t.Fatalf("lock record leaks credential material %q:\n%s", leak, raw)
		}
		if strings.Contains(evidence.Message, leak) {
			t.Fatalf("lock evidence message leaks credential material %q: %+v", leak, evidence)
		}
	}
}

func TestTokenScopedGatewayLockProfileHomesShareCredentialLockDir(t *testing.T) {
	root := t.TempDir()
	credential := "123456:ABC-shared-token"

	t.Setenv("GORMES_HOME", filepath.Join(root, "profiles", "main"))
	mainStore := newTokenLockTestStore(t, config.GatewayLockDir(), 1001, 501, nil)
	mainLock, _, err := mainStore.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: credential,
	})
	if err != nil {
		t.Fatalf("main acquire: %v", err)
	}
	wantDir := filepath.Join(root, "runtime", "gateway-locks")
	if filepath.Dir(mainLock.Path()) != wantDir {
		t.Fatalf("main lock dir = %q, want shared owner root %q", filepath.Dir(mainLock.Path()), wantDir)
	}

	t.Setenv("GORMES_HOME", filepath.Join(root, "profiles", "mineru"))
	profileStore := newTokenLockTestStore(t, config.GatewayLockDir(), 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 501, command: "gormes gateway"},
	})
	profileLock, evidence, err := profileStore.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: credential,
	})
	if !errors.Is(err, ErrTokenLockHeld) || profileLock != nil {
		t.Fatalf("profile acquire err=%v lock=%v evidence=%+v, want shared-token lock held", err, profileLock, evidence)
	}
	if evidence.Path != mainLock.Path() || evidence.OwnerPID != 1001 {
		t.Fatalf("profile evidence = %+v, want main lock path %q owned by pid 1001", evidence, mainLock.Path())
	}
}

func TestTokenScopedGatewayLockRejectsSameTokenAndAllowsDifferentScopes(t *testing.T) {
	dir := t.TempDir()
	first := newTokenLockTestStore(t, dir, 1001, 501, nil)
	if _, _, err := first.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	}); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	second := newTokenLockTestStore(t, dir, 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 501, command: "gormes gateway"},
	})
	_, evidence, err := second.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	})
	if !errors.Is(err, ErrTokenLockHeld) {
		t.Fatalf("same token err = %v, want ErrTokenLockHeld", err)
	}
	if evidence.Status != TokenLockStatusHeld || evidence.OwnerPID != 1001 {
		t.Fatalf("same token evidence = %+v, want held by pid 1001", evidence)
	}

	if _, _, err := second.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "other-token",
	}); err != nil {
		t.Fatalf("different token acquire: %v", err)
	}
	if _, _, err := second.Acquire(context.Background(), TokenLockRequest{
		Platform:   "discord",
		Credential: "shared-token",
	}); err != nil {
		t.Fatalf("different platform acquire: %v", err)
	}

	locks, err := filepath.Glob(filepath.Join(dir, "*.lock"))
	if err != nil {
		t.Fatalf("glob locks: %v", err)
	}
	if len(locks) != 3 {
		t.Fatalf("lock count = %d, want same-token rejection plus two independent locks to leave 3 files: %v", len(locks), locks)
	}
}

func TestTokenScopedGatewayLockClearsStaleStoppedOwnerWithoutDeletingUnrelatedLocks(t *testing.T) {
	dir := t.TempDir()
	owner := newTokenLockTestStore(t, dir, 1001, 501, nil)
	ownerLock, _, err := owner.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	})
	if err != nil {
		t.Fatalf("owner acquire: %v", err)
	}
	unrelated, _, err := newTokenLockTestStore(t, dir, 1002, 502, nil).Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "unrelated-token",
	})
	if err != nil {
		t.Fatalf("unrelated acquire: %v", err)
	}

	contender := newTokenLockTestStore(t, dir, 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 501, command: "gormes gateway", stopped: true},
		1002: {startTime: 502, command: "gormes gateway"},
	})
	newLock, evidence, err := contender.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	})
	if err != nil {
		t.Fatalf("contender acquire after stale owner: %v", err)
	}
	if evidence.Status != TokenLockStatusStaleCleared {
		t.Fatalf("stale evidence status = %q, want %q", evidence.Status, TokenLockStatusStaleCleared)
	}
	if evidence.ProcessValidation.Status != runtimeproc.ValidationStopped {
		t.Fatalf("stale validation = %+v, want stopped process evidence", evidence.ProcessValidation)
	}
	if newLock.Path() != ownerLock.Path() {
		t.Fatalf("new lock path = %q, want reused scoped path %q", newLock.Path(), ownerLock.Path())
	}
	if _, err := os.Stat(unrelated.Path()); err != nil {
		t.Fatalf("unrelated lock was removed: %v", err)
	}
	record := readTokenLockRecordFixture(t, ownerLock.Path())
	if record.PID != 2002 {
		t.Fatalf("reacquired lock pid = %d, want contender pid 2002", record.PID)
	}
}

func TestTokenScopedGatewayLockReentrantAcquireWhenStartTimeUnavailable(t *testing.T) {
	dir := t.TempDir()
	store := newTokenLockTestStore(t, dir, 1001, 0, fakeRuntimeProcessTable{
		1001: {startTime: 0, command: "gormes gateway"},
	})
	first, _, err := store.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	})
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	second, evidence, err := store.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	})
	if err != nil {
		t.Fatalf("reentrant acquire without start time: %v evidence=%+v", err, evidence)
	}
	if second.Path() != first.Path() || evidence.Status != TokenLockStatusAcquired {
		t.Fatalf("reentrant acquire = path %q evidence %+v, want same path acquired", second.Path(), evidence)
	}
}

func TestTokenScopedGatewayLockClearsStoppedOwnerWhenStartTimeUnavailable(t *testing.T) {
	dir := t.TempDir()
	owner := newTokenLockTestStore(t, dir, 1001, 0, nil)
	ownerLock, _, err := owner.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	})
	if err != nil {
		t.Fatalf("owner acquire: %v", err)
	}

	contender := newTokenLockTestStore(t, dir, 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 0, command: "gormes gateway", stopped: true},
	})
	newLock, evidence, err := contender.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "shared-token",
	})
	if err != nil {
		t.Fatalf("contender acquire after stopped owner without start time: %v", err)
	}
	if evidence.Status != TokenLockStatusStaleCleared || evidence.ProcessValidation.Status != runtimeproc.ValidationStopped {
		t.Fatalf("evidence = %+v, want stale-cleared stopped-process evidence", evidence)
	}
	if newLock.Path() != ownerLock.Path() {
		t.Fatalf("new lock path = %q, want reused scoped path %q", newLock.Path(), ownerLock.Path())
	}
}

func TestTokenScopedGatewayLockCredentialHashMismatchIsReportedWithoutDeletingFile(t *testing.T) {
	dir := t.TempDir()
	credential := "shared-token"
	store := newTokenLockTestStore(t, dir, 2002, 602, fakeRuntimeProcessTable{
		1001: {startTime: 501, command: "gormes gateway", stopped: true},
	})
	path := store.LockPath("telegram", credential)
	writeTokenLockJSONFixture(t, path, map[string]any{
		"kind":            "gormes-gateway-token-lock",
		"platform":        "telegram",
		"credential_hash": "not-" + TokenCredentialHash(credential),
		"pid":             1001,
		"start_time":      501,
		"updated_at":      "2026-04-25T17:00:00Z",
	})

	_, evidence, err := store.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: credential,
	})
	if !errors.Is(err, ErrTokenLockCredentialHashMismatch) {
		t.Fatalf("Acquire err = %v, want ErrTokenLockCredentialHashMismatch", err)
	}
	if evidence.Status != TokenLockStatusCredentialHashMismatch {
		t.Fatalf("evidence status = %q, want credential-hash-mismatch", evidence.Status)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mismatched lock: %v", err)
	}
	if !strings.Contains(string(raw), "not-"+TokenCredentialHash(credential)) {
		t.Fatalf("mismatched lock was overwritten or deleted:\n%s", raw)
	}
}

func TestTokenScopedGatewayLockReleaseRemovesOnlyCurrentOwnerAndReportsReleaseFailures(t *testing.T) {
	dir := t.TempDir()
	current := newTokenLockTestStore(t, dir, 3003, 703, nil)
	currentLock, _, err := current.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "current-token",
	})
	if err != nil {
		t.Fatalf("current acquire: %v", err)
	}
	otherLock, _, err := newTokenLockTestStore(t, dir, 4004, 804, nil).Acquire(context.Background(), TokenLockRequest{
		Platform:   "discord",
		Credential: "other-token",
	})
	if err != nil {
		t.Fatalf("other acquire: %v", err)
	}

	evidence, err := currentLock.Release(context.Background())
	if err != nil {
		t.Fatalf("release current lock: %v", err)
	}
	if evidence.Status != TokenLockStatusReleased {
		t.Fatalf("release evidence = %+v, want released", evidence)
	}
	if _, err := os.Stat(currentLock.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("current lock stat err = %v, want removed", err)
	}
	if _, err := os.Stat(otherLock.Path()); err != nil {
		t.Fatalf("other lock was removed: %v", err)
	}

	failingLock, _, err := current.Acquire(context.Background(), TokenLockRequest{
		Platform:   "telegram",
		Credential: "failing-release-token",
	})
	if err != nil {
		t.Fatalf("failing lock acquire: %v", err)
	}
	current.removeFile = func(string) error {
		return errors.New("unlink denied\nAuthorization: Bearer sk-tokenlock-secret\n**Injected:** yes")
	}
	evidence, err = failingLock.Release(context.Background())
	if !errors.Is(err, ErrTokenLockReleaseFailed) {
		t.Fatalf("release failure err = %v, want ErrTokenLockReleaseFailed", err)
	}
	if evidence.Status != TokenLockStatusReleaseFailed || !strings.Contains(evidence.Message, "unlink denied") {
		t.Fatalf("release failure evidence = %+v, want release-failed with unlink evidence", evidence)
	}
	for _, forbidden := range []string{"sk-tokenlock-secret", "Bearer sk", "\n", "**Injected:**"} {
		if strings.Contains(evidence.Message, forbidden) {
			t.Fatalf("release failure evidence leaked %q in %q", forbidden, evidence.Message)
		}
	}
	if !strings.Contains(evidence.Message, "[redacted]") {
		t.Fatalf("release failure evidence = %q, want redaction marker", evidence.Message)
	}

}

type fakeRuntimeProcessTable map[int]fakeRuntimeProcess

type fakeRuntimeProcess struct {
	startTime int64
	command   string
	stopped   bool
	err       error
}

func (f fakeRuntimeProcessTable) LookupRuntimeProcess(pid int) (runtimeproc.ProcessInfo, error) {
	record, ok := f[pid]
	if !ok {
		return runtimeproc.ProcessInfo{}, runtimeproc.ErrNotFound
	}
	if record.err != nil {
		return runtimeproc.ProcessInfo{}, record.err
	}
	return runtimeproc.ProcessInfo{
		PID:       pid,
		StartTime: record.startTime,
		Command:   record.command,
		Stopped:   record.stopped,
	}, nil
}

func newTokenLockTestStore(t *testing.T, dir string, pid int, startTime int64, processes fakeRuntimeProcessTable) *TokenLockStore {
	t.Helper()
	store := NewTokenLockStore(dir)
	store.now = func() time.Time { return time.Date(2026, 4, 25, 17, 0, 0, 0, time.UTC) }
	store.pid = func() int { return pid }
	store.startTime = func(got int) (int64, bool) {
		if got != pid {
			return 0, false
		}
		return startTime, true
	}
	store.argv = func() []string { return []string{"gormes", "gateway"} }
	store.processes = processes
	return store
}

type tokenLockRecordFixture struct {
	Kind string `json:"kind"`
	PID  int    `json:"pid"`
}

func readTokenLockRecordFixture(t *testing.T, path string) tokenLockRecordFixture {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lock fixture: %v", err)
	}
	var record tokenLockRecordFixture
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatalf("decode lock fixture: %v", err)
	}
	return record
}

func writeTokenLockJSONFixture(t *testing.T, path string, record map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create lock fixture dir: %v", err)
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatalf("encode lock fixture: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write lock fixture: %v", err)
	}
}
