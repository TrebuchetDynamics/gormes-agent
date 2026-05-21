package modassert

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

// RequirePublicModuleVersion asserts that the current module graph resolves
// modulePath to wantVersion without a local replace directive.
func RequirePublicModuleVersion(t *testing.T, modulePath, wantVersion string) {
	t.Helper()

	cmd := exec.Command("go", "list", "-m", "-json", modulePath)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list module %s: %v\n%s", modulePath, err, out)
	}

	var mod struct {
		Path    string
		Version string
		Replace *struct {
			Path    string
			Version string
		}
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		t.Fatalf("parse go list module JSON for %s: %v\n%s", modulePath, err, out)
	}
	if mod.Path != modulePath {
		t.Fatalf("module path = %q, want %q", mod.Path, modulePath)
	}
	if mod.Replace != nil {
		t.Fatalf("module %s is replaced by %s %s, want public release %s", modulePath, mod.Replace.Path, mod.Replace.Version, wantVersion)
	}
	if mod.Version != wantVersion {
		t.Fatalf("module %s version = %q, want public release %s", modulePath, mod.Version, wantVersion)
	}
}
