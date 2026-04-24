package site

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const canonicalInstallScript = "../../../scripts/install.sh"
const embeddedInstallScript = "installers/install.sh"

func runInstallSH(t *testing.T, body string, env ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("sh", "-c", `. "`+canonicalInstallScript+`"; `+body)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), append([]string{"GORMES_INSTALL_TEST_MODE=1"}, env...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runInstallScript(t *testing.T, env ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("sh", canonicalInstallScript)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeExecutable(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestInstallSH_SiteCopyMatchesCanonicalScript(t *testing.T) {
	canonical, err := os.ReadFile(canonicalInstallScript)
	if err != nil {
		t.Fatalf("read canonical install.sh: %v", err)
	}
	embedded, err := os.ReadFile(embeddedInstallScript)
	if err != nil {
		t.Fatalf("read embedded install.sh: %v", err)
	}
	if !bytes.Equal(embedded, canonical) {
		t.Fatal("embedded install.sh differs from scripts/install.sh")
	}
}

func fakeGitScript() string {
	return `#!/bin/sh
set -eu
printf 'git' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"

case "$1" in
  clone)
    target=
    for arg in "$@"; do target=$arg; done
    mkdir -p "$target/.git" "$target/cmd/gormes"
    printf 'module github.com/TrebuchetDynamics/gormes-agent\n\ngo 1.25.0\n' > "$target/go.mod"
    ;;
  status)
    if [ "${2:-}" = "--porcelain" ]; then
      printf '%s\n' "${GORMES_FAKE_STATUS:-}"
    fi
    ;;
  stash|fetch|checkout|pull)
    ;;
  *)
    ;;
esac
`
}

func fakeGoScript(logName string, version string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu
printf '%s' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %%s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"

if [ "${1:-}" = "env" ] && [ "${2:-}" = "GOVERSION" ]; then
  printf '%s\n'
  exit 0
fi

if [ "${1:-}" = "env" ] && [ "${2:-}" = "GOBIN" ]; then
  printf '\n'
  exit 0
fi

if [ "${1:-}" = "env" ] && [ "${2:-}" = "GOPATH" ]; then
  printf '%%s/go\n' "$HOME"
  exit 0
fi

if [ "${1:-}" = "install" ]; then
  out="${HOME}/go/bin/gormes"
  mkdir -p "$(dirname "$out")"
  printf '#!/bin/sh\nexit 0\n' > "$out"
  chmod +x "$out"
  exit 0
fi

if [ "${1:-}" = "build" ]; then
  out=
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "-o" ]; then
      shift
      out="$1"
      break
    fi
    shift
  done
  mkdir -p "$(dirname "$out")"
  cat > "$out" <<'EOF'
#!/bin/sh
case "${1:-}" in
  version)
    printf 'gormes test-build\n'
    ;;
  doctor)
    if [ "${2:-}" = "--offline" ]; then
      printf 'doctor ok\n'
    fi
    ;;
esac
EOF
  chmod +x "$out"
  exit 0
fi

exit 0
`, logName, version)
}

func writeFakeUnixToolchain(t *testing.T, root string) (string, string) {
	t.Helper()
	bin := filepath.Join(root, "fakebin")
	logPath := filepath.Join(root, "toolchain.log")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}

	writeExecutable(t, filepath.Join(bin, "git"), fakeGitScript())
	writeExecutable(t, filepath.Join(bin, "go"), fakeGoScript("go", "go1.25.0"))

	return bin, logPath
}

func linkBasicUnixTools(t *testing.T, bin string) {
	t.Helper()
	for _, name := range []string{"cat", "chmod", "cp", "dirname", "ln", "mkdir", "mv", "rm", "uname"} {
		realPath, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("look up %s: %v", name, err)
		}
		linkPath := filepath.Join(bin, name)
		if err := os.Symlink(realPath, linkPath); err != nil && !os.IsExist(err) {
			t.Fatalf("symlink %s: %v", name, err)
		}
	}
}

func writeFakePackageManager(t *testing.T, bin string, name string) {
	t.Helper()
	writeExecutable(t, filepath.Join(bin, name), `#!/bin/sh
set -eu
printf '`+name+`' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"

for arg in "$@"; do
  case "$arg" in
    git)
      cp "$GORMES_FAKE_GIT_TEMPLATE" "$GORMES_FAKE_BIN/git"
      chmod +x "$GORMES_FAKE_BIN/git"
      ;;
    go|golang)
      cp "$GORMES_FAKE_GO_TEMPLATE" "$GORMES_FAKE_BIN/go"
      chmod +x "$GORMES_FAKE_BIN/go"
      ;;
  esac
done
`)
}

func writeFakeDownloadTools(t *testing.T, bin string) {
	t.Helper()
	writeExecutable(t, filepath.Join(bin, "curl"), `#!/bin/sh
set -eu
printf 'curl' >> "$GORMES_FAKE_LOG"
out=
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
  if [ "${prev:-}" = "-o" ]; then
    out="$arg"
  fi
  prev="$arg"
done
printf '\n' >> "$GORMES_FAKE_LOG"
if [ -n "$out" ]; then
  mkdir -p "$(dirname "$out")"
  printf 'fake go tarball\n' > "$out"
fi
`)
	writeExecutable(t, filepath.Join(bin, "wget"), `#!/bin/sh
set -eu
printf 'wget' >> "$GORMES_FAKE_LOG"
out=
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
  if [ "${prev:-}" = "-O" ]; then
    out="$arg"
  fi
  prev="$arg"
done
printf '\n' >> "$GORMES_FAKE_LOG"
if [ -n "$out" ]; then
  mkdir -p "$(dirname "$out")"
  printf 'fake go tarball\n' > "$out"
fi
`)
	writeExecutable(t, filepath.Join(bin, "tar"), `#!/bin/sh
set -eu
printf 'tar' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"
home="${GORMES_INSTALL_HOME:-$HOME/.gormes}"
mkdir -p "$home/go/bin"
cp "$GORMES_FAKE_MANAGED_GO_TEMPLATE" "$home/go/bin/go"
chmod +x "$home/go/bin/go"
`)
}

func writeBootstrapTemplates(t *testing.T, root string, systemGoVersion string) (string, string, string) {
	t.Helper()
	gitTemplate := filepath.Join(root, "git.template")
	goTemplate := filepath.Join(root, "go.template")
	managedGoTemplate := filepath.Join(root, "managed-go.template")
	writeExecutable(t, gitTemplate, fakeGitScript())
	writeExecutable(t, goTemplate, fakeGoScript("go", systemGoVersion))
	writeExecutable(t, managedGoTemplate, fakeGoScript("managed-go", "go1.25.0"))
	return gitTemplate, goTemplate, managedGoTemplate
}

func TestInstallSH_DefaultManagedPaths(t *testing.T) {
	home := t.TempDir()
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`printf '%s|%s|%s\n' "$(managed_home_dir)" "$(managed_checkout_dir)" "$(pick_bin_dir)"`,
		"HOME="+home,
		"PATH="+fakebin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("runInstallSH: %v\n%s", err, out)
	}
	want := home + "/.gormes|" + home + "/.gormes/gormes-agent|" + home + "/.local/bin"
	if strings.TrimSpace(out) != want {
		t.Fatalf("paths = %q, want %q", strings.TrimSpace(out), want)
	}
}

func TestInstallSH_TermuxUsesPrefixBin(t *testing.T) {
	home := t.TempDir()
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`printf '%s\n' "$(pick_bin_dir)"`,
		"HOME="+home,
		"PATH="+fakebin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GORMES_FAKE_LOG="+logPath,
		"PREFIX=/data/data/com.termux/files/usr",
		"TERMUX_VERSION=0.118.0",
	)
	if err != nil {
		t.Fatalf("runInstallSH: %v\n%s", err, out)
	}
	if strings.TrimSpace(out) != "/data/data/com.termux/files/usr/bin" {
		t.Fatalf("pick_bin_dir = %q", strings.TrimSpace(out))
	}
}

func TestInstallSH_WindowsShellHintMentionsPowerShell(t *testing.T) {
	home := t.TempDir()
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`UNAME=MSYS_NT-10.0 check_platform`,
		"HOME="+home,
		"PATH="+fakebin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GORMES_FAKE_LOG="+logPath,
	)
	if err == nil {
		t.Fatal("expected check_platform to fail for Windows-like shell")
	}
	if !strings.Contains(out, "install.ps1") {
		t.Fatalf("Windows shell hint missing install.ps1:\n%s", out)
	}
}

func TestInstallSH_FirstInstallCreatesManagedCheckoutAndPublishedCommand(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	if _, err := os.Stat(filepath.Join(checkout, ".git")); err != nil {
		t.Fatalf("managed checkout missing: %v", err)
	}
	published := filepath.Join(home, ".local", "bin", "gormes")
	if _, err := os.Stat(published); err != nil {
		t.Fatalf("published command missing: %v", err)
	}
	if !strings.Contains(out, "Core install: succeeded") {
		t.Fatalf("success summary missing:\n%s", out)
	}
	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake log: %v", err)
	}
	log := string(logBody)
	for _, want := range []string{
		"git clone --branch main",
		"go build -o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
}

func TestInstallSH_RerunUpdatesManagedCheckoutWithoutCloning(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	if err := os.MkdirAll(filepath.Join(checkout, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir checkout: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(checkout, "cmd", "gormes"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/gormes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "go.mod"), []byte("module github.com/TrebuchetDynamics/gormes-agent\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake log: %v", err)
	}
	log := string(logBody)
	for _, want := range []string{"git status --porcelain", "git fetch origin main", "git checkout main", "git pull --ff-only origin main"} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	if strings.Contains(log, "git clone") {
		t.Fatalf("rerun cloned instead of updating:\n%s", log)
	}
}

func TestInstallSH_TermuxInstallsMissingGitAndGo(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	prefix := filepath.Join(root, "data", "data", "com.termux", "files", "usr")
	fakebin := filepath.Join(root, "fakebin")
	logPath := filepath.Join(root, "toolchain.log")
	if err := os.MkdirAll(fakebin, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	linkBasicUnixTools(t, fakebin)
	gitTemplate, goTemplate, managedGoTemplate := writeBootstrapTemplates(t, root, "go1.25.0")
	writeFakePackageManager(t, fakebin, "pkg")

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_BIN="+fakebin,
		"GORMES_FAKE_GIT_TEMPLATE="+gitTemplate,
		"GORMES_FAKE_GO_TEMPLATE="+goTemplate,
		"GORMES_FAKE_MANAGED_GO_TEMPLATE="+managedGoTemplate,
		"PREFIX="+prefix,
		"TERMUX_VERSION=0.118.0",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake log: %v", err)
	}
	log := string(logBody)
	for _, want := range []string{
		"pkg install -y git golang",
		"git clone --branch main",
		"go build -o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	if !strings.Contains(out, "Core install: succeeded") {
		t.Fatalf("success summary missing:\n%s", out)
	}
}

func TestInstallSH_InstallsManagedGoWhenGoIsMissing(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin := filepath.Join(root, "fakebin")
	logPath := filepath.Join(root, "toolchain.log")
	if err := os.MkdirAll(fakebin, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	linkBasicUnixTools(t, fakebin)
	_, _, managedGoTemplate := writeBootstrapTemplates(t, root, "go1.25.0")
	writeExecutable(t, filepath.Join(fakebin, "git"), fakeGitScript())
	writeFakeDownloadTools(t, fakebin)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_MANAGED_GO_TEMPLATE="+managedGoTemplate,
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake log: %v", err)
	}
	log := string(logBody)
	for _, want := range []string{
		"curl -fsSL",
		"tar -C " + filepath.Join(home, ".gormes"),
		"managed-go build -o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	if !strings.Contains(out, "Core install: succeeded") {
		t.Fatalf("success summary missing:\n%s", out)
	}
}

func TestInstallSH_ReplacesTooOldGoWithManagedGo(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin := filepath.Join(root, "fakebin")
	logPath := filepath.Join(root, "toolchain.log")
	if err := os.MkdirAll(fakebin, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	linkBasicUnixTools(t, fakebin)
	_, _, managedGoTemplate := writeBootstrapTemplates(t, root, "go1.24.0")
	writeExecutable(t, filepath.Join(fakebin, "git"), fakeGitScript())
	writeExecutable(t, filepath.Join(fakebin, "go"), fakeGoScript("old-go", "go1.24.0"))
	writeFakeDownloadTools(t, fakebin)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_MANAGED_GO_TEMPLATE="+managedGoTemplate,
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake log: %v", err)
	}
	log := string(logBody)
	for _, want := range []string{
		"old-go env GOVERSION",
		"curl -fsSL",
		"managed-go build -o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
}
