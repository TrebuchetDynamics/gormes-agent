package update

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRunUpdateBinaryPublishBuildsVerifiesPublishesAndRefreshesActivePath(t *testing.T) {
	root := t.TempDir()
	checkout := writeUpdatePublishCheckout(t, root)
	managed := filepath.Join(root, "home", "bin", "gormes")
	published := filepath.Join(root, "published", "gormes")
	active := filepath.Join(root, "shadow", "gormes")
	if err := os.MkdirAll(filepath.Dir(active), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(active, []byte("old-active"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeUpdateCommandRunner{}

	report := RunUpdateBinaryPublish(context.Background(), UpdateBinaryPublishOptions{
		CheckoutDir:       checkout,
		ManagedBinPath:    managed,
		PublishedBinPath:  published,
		ActivePathPath:    active,
		RefreshActivePath: true,
		Runner:            runner,
		Git: newFakeUpdateGitRunner(map[string]UpdateGitResult{
			"rev-parse --short HEAD": {Stdout: "abc1234\n"},
			"diff --quiet":           {},
			"diff --cached --quiet":  {},
		}),
	})

	if report.Failed {
		t.Fatalf("RunUpdateBinaryPublish failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceBuildCompleted)
	assertUpdateEvidence(t, report, UpdateEvidencePublishCompleted)
	assertUpdateEvidence(t, report, UpdateEvidenceActivePathRefreshed)
	for _, path := range []string{managed, published, active} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(body) != fakeBuiltBinaryBody {
			t.Fatalf("%s body = %q, want built binary", path, body)
		}
	}
	if !runner.sawCommandPrefix("go build -trimpath -ldflags") {
		t.Fatalf("runner commands missing go build: %#v", runner.commands)
	}
	if got := runner.verifyTargets(); !reflect.DeepEqual(got, []string{managed, published, active}) {
		t.Fatalf("verify targets = %#v, want managed, published, active", got)
	}
}

func TestRunUpdateBinaryPublishSkipsActivePathInsideSandboxBoundary(t *testing.T) {
	root := t.TempDir()
	checkout := writeUpdatePublishCheckout(t, root)
	active := filepath.Join(root, "shadow", "gormes")
	if err := os.MkdirAll(filepath.Dir(active), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(active, []byte("old-active"), 0o755); err != nil {
		t.Fatal(err)
	}

	report := RunUpdateBinaryPublish(context.Background(), UpdateBinaryPublishOptions{
		CheckoutDir:       checkout,
		ManagedBinPath:    filepath.Join(root, "home", "bin", "gormes"),
		PublishedBinPath:  filepath.Join(root, "published", "gormes"),
		ActivePathPath:    active,
		RefreshActivePath: false,
		Runner:            &fakeUpdateCommandRunner{},
		Git: newFakeUpdateGitRunner(map[string]UpdateGitResult{
			"rev-parse --short HEAD": {Stdout: "abc1234\n"},
			"diff --quiet":           {},
			"diff --cached --quiet":  {},
		}),
	})

	if report.Failed {
		t.Fatalf("RunUpdateBinaryPublish failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceActivePathSkipped)
	body, err := os.ReadFile(active)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "old-active" {
		t.Fatalf("sandbox active path was mutated: %q", body)
	}
}

func TestRunUpdateBinaryPublishHandlesPublishedPathEqualToManagedPath(t *testing.T) {
	root := t.TempDir()
	checkout := writeUpdatePublishCheckout(t, root)
	managed := filepath.Join(root, "home", "bin", "gormes")
	runner := &fakeUpdateCommandRunner{}

	report := RunUpdateBinaryPublish(context.Background(), UpdateBinaryPublishOptions{
		CheckoutDir:       checkout,
		ManagedBinPath:    managed,
		PublishedBinPath:  managed,
		RefreshActivePath: true,
		Runner:            runner,
		Git: newFakeUpdateGitRunner(map[string]UpdateGitResult{
			"rev-parse --short HEAD": {Stdout: "abc1234\n"},
			"diff --quiet":           {},
			"diff --cached --quiet":  {},
		}),
	})

	if report.Failed {
		t.Fatalf("RunUpdateBinaryPublish failed: %+v", report)
	}
	body, err := os.ReadFile(managed)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != fakeBuiltBinaryBody {
		t.Fatalf("managed binary body = %q, want built binary", body)
	}
	if got := runner.verifyTargets(); !reflect.DeepEqual(got, []string{managed, managed}) {
		t.Fatalf("verify targets = %#v, want managed verified for build and publish", got)
	}
}

func TestRunUpdateBinaryPublishBuildFailureLeavesPublishedBinaryUntouched(t *testing.T) {
	root := t.TempDir()
	checkout := writeUpdatePublishCheckout(t, root)
	published := filepath.Join(root, "published", "gormes")
	if err := os.MkdirAll(filepath.Dir(published), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(published, []byte("old-published"), 0o755); err != nil {
		t.Fatal(err)
	}

	report := RunUpdateBinaryPublish(context.Background(), UpdateBinaryPublishOptions{
		CheckoutDir:      checkout,
		ManagedBinPath:   filepath.Join(root, "home", "bin", "gormes"),
		PublishedBinPath: published,
		Runner:           &fakeUpdateCommandRunner{buildErr: errors.New("compile failed")},
		Git: newFakeUpdateGitRunner(map[string]UpdateGitResult{
			"rev-parse --short HEAD": {Stdout: "abc1234\n"},
			"diff --quiet":           {},
			"diff --cached --quiet":  {},
		}),
	})

	if !report.Failed {
		t.Fatalf("build failure should fail report: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceBuildFailed)
	body, err := os.ReadFile(published)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "old-published" {
		t.Fatalf("published binary changed after failed build: %q", body)
	}
}

func writeUpdatePublishCheckout(t *testing.T, root string) string {
	t.Helper()
	checkout := filepath.Join(root, "checkout")
	versionDir := filepath.Join(checkout, "cmd", "gormes")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "version.go"), []byte(`package main

var Version = "9.8.7"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return checkout
}

const fakeBuiltBinaryBody = "fake-gormes-binary"

type fakeUpdateCommandRunner struct {
	commands []string
	buildErr error
}

func (r *fakeUpdateCommandRunner) RunCommand(_ context.Context, _ string, _ []string, name string, args ...string) UpdateCommandResult {
	key := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.commands = append(r.commands, key)
	if name == "go" && len(args) > 0 && args[0] == "build" {
		if r.buildErr != nil {
			return UpdateCommandResult{Stderr: "compile failed", Err: r.buildErr}
		}
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-o" {
				if err := os.WriteFile(args[i+1], []byte(fakeBuiltBinaryBody), 0o755); err != nil {
					return UpdateCommandResult{Err: err}
				}
				return UpdateCommandResult{}
			}
		}
		return UpdateCommandResult{Err: errors.New("missing -o")}
	}
	if len(args) == 1 && args[0] == "version" {
		return UpdateCommandResult{Stdout: "gormes version 9.8.7 abc1234\n"}
	}
	return UpdateCommandResult{}
}

func (r *fakeUpdateCommandRunner) sawCommandPrefix(prefix string) bool {
	for _, command := range r.commands {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

func (r *fakeUpdateCommandRunner) verifyTargets() []string {
	var out []string
	for _, command := range r.commands {
		if strings.HasSuffix(command, " version") && !strings.HasPrefix(command, "go ") {
			out = append(out, strings.TrimSuffix(command, " version"))
		}
	}
	return out
}
