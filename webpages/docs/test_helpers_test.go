package docs_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readDoc(t *testing.T, rel string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(".", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func readFirstExisting(t *testing.T, rels ...string) string {
	t.Helper()
	for _, rel := range rels {
		raw, err := os.ReadFile(filepath.Join(".", rel))
		if err == nil {
			return string(raw)
		}
	}
	t.Fatalf("none of the candidate files exist: %s", strings.Join(rels, ", "))
	return ""
}

func readRepoText(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(findRepoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func runGormesHelp(t *testing.T, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/gormes"}, args...)...)
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "GORMES_HOME="+t.TempDir())

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("go run ./cmd/gormes %s timed out", strings.Join(args, " "))
	}
	if err != nil {
		t.Fatalf("go run ./cmd/gormes %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func assertContainsAll(t *testing.T, label, raw string, wants []string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("%s is missing %q", label, want)
		}
	}
}

func readPlaywrightE2EScript(t *testing.T, rel string) string {
	t.Helper()

	type packageJSON struct {
		Scripts map[string]string `json:"scripts"`
	}

	var pkg packageJSON
	if err := json.Unmarshal([]byte(readDoc(t, rel)), &pkg); err != nil {
		t.Fatalf("parse %s as json: %v", rel, err)
	}

	script, ok := pkg.Scripts["test:e2e"]
	if !ok {
		t.Fatalf("%s does not define scripts.test:e2e", rel)
	}

	return script
}
