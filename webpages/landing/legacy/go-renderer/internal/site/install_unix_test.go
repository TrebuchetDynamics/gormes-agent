package site

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const canonicalInstallScript = "../../../../../../install.sh"

func runInstallSH(t *testing.T, body string, env ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("sh", "-c", `. "`+canonicalInstallScript+`"; `+body)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), append([]string{"GORMES_INSTALL_TEST_MODE=1"}, env...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runInstallScript(t *testing.T, env ...string) (string, error) {
	return runInstallScriptWithArgsInDir(t, ".", nil, env...)
}

func runInstallScriptInDir(t *testing.T, dir string, env ...string) (string, error) {
	return runInstallScriptWithArgsInDir(t, dir, nil, env...)
}

func runInstallScriptWithArgs(t *testing.T, args []string, env ...string) (string, error) {
	return runInstallScriptWithArgsInDir(t, ".", args, env...)
}

func runInstallScriptWithArgsInDir(t *testing.T, dir string, args []string, env ...string) (string, error) {
	t.Helper()
	scriptPath, err := filepath.Abs(canonicalInstallScript)
	if err != nil {
		t.Fatalf("Abs(%s): %v", canonicalInstallScript, err)
	}
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeExecutable(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestInstallSH_SiteDoesNotServeUnixInstaller(t *testing.T) {
	site, err := loadSite()
	if err != nil {
		t.Fatalf("loadSite: %v", err)
	}
	if got := site.InstallScript("install.sh"); got != nil {
		t.Fatal("legacy site must not serve install.sh from gormes.ai")
	}
}

func TestInstallSH_CanonicalScriptLivesAtRepoRoot(t *testing.T) {
	if _, err := os.Stat(canonicalInstallScript); err != nil {
		t.Fatalf("repository-root install.sh should be the canonical Unix installer: %v", err)
	}
	if _, err := os.Stat("../../../../../../scripts/install.sh"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scripts/install.sh should not be the canonical Unix installer: err=%v", err)
	}
}

func TestInstallSH_DeployWorkflowDoesNotWatchRepoRootScript(t *testing.T) {
	workflow, err := os.ReadFile("../../../../../../.github/workflows/deploy-gormes-www.yml")
	if err != nil {
		t.Fatalf("read deploy workflow: %v", err)
	}
	body := string(workflow)
	if strings.Contains(body, "\n      - 'install.sh'\n") {
		t.Fatalf("deploy workflow should not watch repository-root install.sh after /install.sh removal:\n%s", body)
	}
	if strings.Contains(body, "scripts/install.sh") {
		t.Fatalf("deploy workflow should not watch scripts/install.sh:\n%s", body)
	}
}

func TestInstallSH_HasNoSiteCopy(t *testing.T) {
	_, err := os.Stat("installers/install.sh")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("site install.sh copy should not exist: err=%v", err)
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
  --version)
    printf 'git version 2.53.0\n'
    ;;
  clone)
    if [ "${GORMES_FAKE_FAIL_SSH_CLONE:-}" = "1" ]; then
      for arg in "$@"; do
        case "$arg" in
          git@github.com:*)
            exit 1
            ;;
        esac
      done
    fi
    target=
    for arg in "$@"; do target=$arg; done
    mkdir -p "$target/.git" "$target/cmd/gormes"
    printf 'module github.com/TrebuchetDynamics/gormes-agent\n\ngo 1.26.0\n' > "$target/go.mod"
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
if [ -n "${GORMES_FAKE_LOG:-}" ]; then
  line='built-gormes'
  for arg in "$@"; do
    line="${line} ${arg}"
  done
  printf '%%s\n' "$line" >> "$GORMES_FAKE_LOG"
fi
case "${1:-}" in
  version)
    if [ "${GORMES_FAKE_BUILT_VERSION_FAIL:-}" = "1" ]; then
      printf 'version failed\n' >&2
      exit 9
    fi
    printf 'gormes test-build\n'
    ;;
  doctor)
    if [ "${2:-}" = "--offline" ]; then
      printf 'doctor ok\n'
    fi
    ;;
  gateway)
    case "${2:-}" in
      status)
        if [ "${3:-}" = "--json" ]; then
          printf '{"runtime":{"gateway_state":"running","pid":%%s,"active_agents":0},"validation":{"status":"live","live":true,"pid":%%s}}\n' "${GORMES_FAKE_GATEWAY_STATUS_PID:-7777}" "${GORMES_FAKE_GATEWAY_STATUS_PID:-7777}"
        else
          printf 'Gateway status\nruntime: running (pid=%%s active_agents=0)\n' "${GORMES_FAKE_GATEWAY_STATUS_PID:-7777}"
        fi
        ;;
      stop)
        printf 'gateway stop: stopped\n'
        ;;
      "")
        printf 'gateway started\n'
        ;;
    esac
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

	linkBasicUnixTools(t, bin)
	writeExecutable(t, filepath.Join(bin, "git"), fakeGitScript())
	writeExecutable(t, filepath.Join(bin, "go"), fakeGoScript("go", "go1.26.0"))
	writeVersionTool(t, filepath.Join(bin, "node"), "v22.22.0")
	writeVersionTool(t, filepath.Join(bin, "rg"), "ripgrep 14.1.1")
	writeVersionTool(t, filepath.Join(bin, "ffmpeg"), "ffmpeg version 7.1.1 Copyright")
	writeExecutable(t, filepath.Join(bin, "curl"), fakeNoReleaseCurlScript())
	writeExecutable(t, filepath.Join(bin, "tar"), fakeTarScript())

	return bin, logPath
}

func writeVersionTool(t *testing.T, path string, version string) {
	t.Helper()
	writeExecutable(t, path, `#!/bin/sh
set -eu
printf '`+version+`\n'
`)
}

func linkBasicUnixTools(t *testing.T, bin string) {
	t.Helper()
	for _, name := range []string{"awk", "cat", "chmod", "cp", "dirname", "head", "ln", "mkdir", "mv", "rm", "sed", "sha256sum", "sleep", "uname"} {
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

func fakeNoReleaseCurlScript() string {
	return `#!/bin/sh
set -eu
printf 'curl' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"
exit 22
`
}

func fakeTarScript() string {
	return `#!/bin/sh
set -eu
printf 'tar' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"
exit 2
`
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
    curl)
      cat > "$GORMES_FAKE_BIN/curl" <<'CURL'
#!/bin/sh
set -eu
printf 'curl' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"
exit 22
CURL
      chmod +x "$GORMES_FAKE_BIN/curl"
      ;;
    tar)
      cat > "$GORMES_FAKE_BIN/tar" <<'TAR'
#!/bin/sh
set -eu
printf 'tar' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"
exit 2
TAR
      chmod +x "$GORMES_FAKE_BIN/tar"
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
url=
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
  if [ "${prev:-}" = "-o" ]; then
    out="$arg"
  fi
  case "$arg" in
    http://*|https://*) url="$arg" ;;
  esac
  prev="$arg"
done
printf '\n' >> "$GORMES_FAKE_LOG"
if [ -n "$out" ]; then
  mkdir -p "$(dirname "$out")"
  case "$url" in
    */repos/TrebuchetDynamics/gormes-agent/releases/latest)
      if [ -z "${GORMES_FAKE_RELEASE_TAG:-}" ]; then
        exit 22
      fi
      printf '{"tag_name":"%s"}\n' "$GORMES_FAKE_RELEASE_TAG" > "$out"
      ;;
    */releases/download/*/*.tar.gz.sha256)
      asset="${url##*/}"
      asset="${asset%.sha256}"
      sum=$(printf 'fake release archive\n' | sha256sum | sed 's/ .*//')
      printf '%s  %s\n' "$sum" "$asset" > "$out"
      ;;
    */releases/download/*/*.tar.gz)
      printf 'fake release archive\n' > "$out"
      ;;
    *)
      printf 'fake go tarball\n' > "$out"
      ;;
  esac
elif [ "$url" = "https://api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest" ]; then
  if [ -z "${GORMES_FAKE_RELEASE_TAG:-}" ]; then
    exit 22
  fi
  printf '{"tag_name":"%s"}\n' "$GORMES_FAKE_RELEASE_TAG"
fi
`)
	writeExecutable(t, filepath.Join(bin, "wget"), `#!/bin/sh
set -eu
printf 'wget' >> "$GORMES_FAKE_LOG"
out=
url=
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
  if [ "${prev:-}" = "-O" ]; then
    out="$arg"
  fi
  case "$arg" in
    http://*|https://*) url="$arg" ;;
  esac
  prev="$arg"
done
printf '\n' >> "$GORMES_FAKE_LOG"
if [ -n "$out" ]; then
  mkdir -p "$(dirname "$out")"
  case "$url" in
    */repos/TrebuchetDynamics/gormes-agent/releases/latest)
      if [ -z "${GORMES_FAKE_RELEASE_TAG:-}" ]; then
        exit 22
      fi
      printf '{"tag_name":"%s"}\n' "$GORMES_FAKE_RELEASE_TAG" > "$out"
      ;;
    */releases/download/*/*.tar.gz.sha256)
      asset="${url##*/}"
      asset="${asset%.sha256}"
      sum=$(printf 'fake release archive\n' | sha256sum | sed 's/ .*//')
      printf '%s  %s\n' "$sum" "$asset" > "$out"
      ;;
    */releases/download/*/*.tar.gz)
      printf 'fake release archive\n' > "$out"
      ;;
    *)
      printf 'fake go tarball\n' > "$out"
      ;;
  esac
elif [ "$url" = "https://api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest" ]; then
  if [ -z "${GORMES_FAKE_RELEASE_TAG:-}" ]; then
    exit 22
  fi
  printf '{"tag_name":"%s"}\n' "$GORMES_FAKE_RELEASE_TAG"
fi
`)
	writeExecutable(t, filepath.Join(bin, "tar"), `#!/bin/sh
set -eu
printf 'tar' >> "$GORMES_FAKE_LOG"
dest=
archive=
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
  if [ "${prev:-}" = "-C" ]; then
    dest="$arg"
  fi
  case "$arg" in
    *.tar.gz) archive="$arg" ;;
  esac
  prev="$arg"
done
printf '\n' >> "$GORMES_FAKE_LOG"
base="${archive##*/}"
case "$base" in
  gormes-*.tar.gz)
    dir="${base%.tar.gz}"
    mkdir -p "$dest/$dir"
    cp "$GORMES_FAKE_RELEASE_BIN_TEMPLATE" "$dest/$dir/gormes"
    chmod +x "$dest/$dir/gormes"
    exit 0
    ;;
esac
home="${GORMES_INSTALL_HOME:-$HOME/.gormes}"
mkdir -p "$home/go/bin"
cp "$GORMES_FAKE_MANAGED_GO_TEMPLATE" "$home/go/bin/go"
chmod +x "$home/go/bin/go"
`)
}

func writeFakeReleaseToolchain(t *testing.T, root string) (string, string, string) {
	t.Helper()
	bin := filepath.Join(root, "fakebin")
	logPath := filepath.Join(root, "toolchain.log")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	linkBasicUnixTools(t, bin)
	writeFakeDownloadTools(t, bin)

	releaseTemplate := filepath.Join(root, "release-gormes.template")
	writeExecutable(t, releaseTemplate, `#!/bin/sh
set -eu
if [ -n "${GORMES_FAKE_LOG:-}" ]; then
  printf 'release-gormes' >> "$GORMES_FAKE_LOG"
  for arg in "$@"; do
    printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
  done
  printf '\n' >> "$GORMES_FAKE_LOG"
fi
case "${1:-}" in
  version)
    printf 'gormes release-test\n'
    ;;
  doctor)
    if [ "${2:-}" = "--offline" ]; then
      printf 'doctor ok\n'
    fi
    ;;
  gateway)
    case "${2:-}" in
      status)
        printf '{"runtime":{"gateway_state":"stopped","active_agents":0},"validation":{"status":"stopped","live":false}}\n'
        ;;
      stop)
        printf 'gateway stop: stopped\n'
        ;;
      "")
        printf 'gateway started\n'
        ;;
    esac
    ;;
esac
`)
	return bin, logPath, releaseTemplate
}

func writeBootstrapTemplates(t *testing.T, root string, systemGoVersion string) (string, string, string) {
	t.Helper()
	gitTemplate := filepath.Join(root, "git.template")
	goTemplate := filepath.Join(root, "go.template")
	managedGoTemplate := filepath.Join(root, "managed-go.template")
	writeExecutable(t, gitTemplate, fakeGitScript())
	writeExecutable(t, goTemplate, fakeGoScript("go", systemGoVersion))
	writeExecutable(t, managedGoTemplate, fakeGoScript("managed-go", "go1.26.0"))
	return gitTemplate, goTemplate, managedGoTemplate
}

func TestInstallSH_DefaultManagedPaths(t *testing.T) {
	home := t.TempDir()
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`printf '%s|%s|%s\n' "$(managed_home_dir)" "$(managed_checkout_dir)" "$(pick_bin_dir)"`,
		"HOME="+home,
		"PATH="+fakebin,
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
		"PATH="+fakebin,
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

func TestInstallSH_RootLinuxDefaultsToFHSLayout(t *testing.T) {
	home := t.TempDir()
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`printf '%s|%s|%s\n' "$(managed_home_dir)" "$(managed_checkout_dir)" "$(pick_bin_dir)"`,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_EFFECTIVE_UID=0",
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("runInstallSH: %v\n%s", err, out)
	}
	want := home + "/.gormes|/usr/local/lib/gormes-agent|/usr/local/bin"
	if strings.TrimSpace(out) != want {
		t.Fatalf("root paths = %q, want %q", strings.TrimSpace(out), want)
	}
}

func TestInstallSH_RootLinuxPreservesLegacyUserScopedCheckout(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, ".gormes", "gormes-agent", ".git")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("mkdir legacy checkout: %v", err)
	}
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`printf '%s|%s|%s\n' "$(managed_home_dir)" "$(managed_checkout_dir)" "$(pick_bin_dir)"`,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_EFFECTIVE_UID=0",
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("runInstallSH: %v\n%s", err, out)
	}
	want := home + "/.gormes|" + home + "/.gormes/gormes-agent|" + home + "/.local/bin"
	if strings.TrimSpace(out) != want {
		t.Fatalf("legacy root paths = %q, want %q", strings.TrimSpace(out), want)
	}
}

func TestInstallSH_ExplicitDirAndBinDirOverrideRootFHSLayout(t *testing.T) {
	home := t.TempDir()
	overrideDir := filepath.Join(home, "src", "gormes-agent")
	overrideBin := filepath.Join(home, "bin")
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`parse_args --dir "$OVERRIDE_DIR" --bin-dir "$OVERRIDE_BIN"; printf '%s|%s\n' "$(managed_checkout_dir)" "$(pick_bin_dir)"`,
		"HOME="+home,
		"OVERRIDE_DIR="+overrideDir,
		"OVERRIDE_BIN="+overrideBin,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_EFFECTIVE_UID=0",
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("runInstallSH: %v\n%s", err, out)
	}
	want := overrideDir + "|" + overrideBin
	if strings.TrimSpace(out) != want {
		t.Fatalf("explicit override paths = %q, want %q", strings.TrimSpace(out), want)
	}
}

func TestInstallSH_WindowsShellHintMentionsPowerShell(t *testing.T) {
	home := t.TempDir()
	fakebin, logPath := writeFakeUnixToolchain(t, t.TempDir())
	out, err := runInstallSH(t,
		`UNAME=MSYS_NT-10.0 check_platform`,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err == nil {
		t.Fatal("expected check_platform to fail for Windows-like shell")
	}
	if !strings.Contains(out, "install.ps1") {
		t.Fatalf("Windows shell hint missing install.ps1:\n%s", out)
	}
}

func TestInstallSH_DefaultInstallNarratesHermesStyleExperience(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_FAIL_SSH_CLONE=1",
		"GORMES_INSTALL_TEST_DISTRO=fedora",
		"GORMES_INSTALL_TEST_HAS_TTY=0",
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s\nlog:\n%s", err, out, readTextFile(t, logPath))
	}

	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	for _, want := range []string{
		"\x1b[1;34m",
		"██████╗  ██████╗ ██████╗",
		"Gormes Agent Installer",
		"An open source AI agent",
		"✓ Detected: linux (fedora)",
		"→ Checking Go",
		"✓ Go go1.26.0 found",
		"→ Checking Git",
		"✓ Git 2.53.0 found",
		"→ Checking Node.js (for browser tools)",
		"✓ Node.js v22.22.0 found",
		"→ Checking ripgrep (fast file search)",
		"✓ ripgrep 14.1.1 found",
		"→ Checking ffmpeg (TTS voice messages)",
		"✓ ffmpeg 7.1.1 found",
		"→ Installing to " + checkout,
		"→ Trying SSH clone",
		"→ SSH failed, trying HTTPS",
		"✓ Cloned via HTTPS",
		"✓ Repository ready",
		"→ Building gormes",
		"✓ Gormes binary ready",
		"→ Setting up gormes command",
		"✓ gormes command ready",
		"→ Setup wizard skipped (no terminal available).",
		"Run 'gormes setup' after install.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("install output missing %q\n%s", want, out)
		}
	}

	for _, reject := range []string{
		"Checking for uv package manager",
		"Checking Python 3.11",
		"Creating virtual environment",
		"Installing dependencies",
		"Installing Node.js dependencies",
		"hermes setup",
	} {
		if strings.Contains(out, reject) {
			t.Fatalf("Gormes installer should not include Hermes Python runtime step %q\n%s", reject, out)
		}
	}
}

func TestInstallSH_DefaultInstallRunsSetupWizardWhenTerminalAvailable(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_TEST_HAS_TTY=1",
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s\nlog:\n%s", err, out, readTextFile(t, logPath))
	}

	if !strings.Contains(out, "→ Starting setup wizard") {
		t.Fatalf("default install did not announce setup wizard:\n%s", out)
	}
	if strings.Contains(out, "Setup wizard skipped") {
		t.Fatalf("interactive install skipped setup wizard:\n%s", out)
	}
	log := readTextFile(t, logPath)
	if !strings.Contains(log, "built-gormes setup") {
		t.Fatalf("default install did not invoke gormes setup:\n%s", log)
	}
}

func TestInstallSH_SkipSetupFlagAvoidsWizardEvenWithTerminal(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScriptWithArgs(t,
		[]string{"--skip-setup"},
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_TEST_HAS_TTY=1",
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s\nlog:\n%s", err, out, readTextFile(t, logPath))
	}

	if !strings.Contains(out, "→ Skipping setup wizard (--skip-setup)") {
		t.Fatalf("--skip-setup did not explain setup skip:\n%s", out)
	}
	log := readTextFile(t, logPath)
	if strings.Contains(log, "built-gormes setup") {
		t.Fatalf("--skip-setup still invoked gormes setup:\n%s", log)
	}
}

func TestInstallSH_DefaultInstallFetchesReleaseBinary(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath, releaseTemplate := writeFakeReleaseToolchain(t, root)
	archiveHash := fmt.Sprintf("%x", sha256.Sum256([]byte("fake release archive\n")))

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_RELEASE_TAG=v9.9.9",
		"GORMES_FAKE_RELEASE_BIN_TEMPLATE="+releaseTemplate,
		"GORMES_FAKE_RELEASE_ARCHIVE_SHA="+archiveHash,
		"TMPDIR="+filepath.Join(root, "tmp"),
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s\nlog:\n%s", err, out, readTextFile(t, logPath))
	}

	managed := filepath.Join(home, ".gormes", "bin", "gormes")
	body, err := os.ReadFile(managed)
	if err != nil {
		t.Fatalf("read managed binary: %v", err)
	}
	if !strings.Contains(string(body), "release-gormes") {
		t.Fatalf("managed binary was not fetched from release:\n%s", body)
	}
	log := readTextFile(t, logPath)
	for _, want := range []string{
		"api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest",
		"releases/download/v9.9.9/gormes-9.9.9-linux-amd64.tar.gz",
		"release-gormes version",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	for _, reject := range []string{"git clone --branch main", "go build -o"} {
		if strings.Contains(log, reject) {
			t.Fatalf("binary-fetch default should not use source-build path %q\n%s", reject, log)
		}
	}
	if !strings.Contains(out, "source: GitHub Releases") {
		t.Fatalf("summary did not name GitHub Releases source:\n%s", out)
	}
}

func TestInstallSH_FromSourceInstallUsesManagedSourceCheckout(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_FROM_SOURCE=1",
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s\nlog:\n%s", err, out, readTextFile(t, logPath))
	}

	managed := filepath.Join(home, ".gormes", "bin", "gormes")
	body, err := os.ReadFile(managed)
	if err != nil {
		t.Fatalf("read managed binary: %v", err)
	}
	if !strings.Contains(string(body), "built-gormes") {
		t.Fatalf("managed binary was not built from source:\n%s", body)
	}
	published := filepath.Join(home, ".local", "bin", "gormes")
	if _, err := os.Stat(published); err != nil {
		t.Fatalf("published command missing: %v", err)
	}
	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	if _, err := os.Stat(filepath.Join(checkout, ".git")); err != nil {
		t.Fatalf("managed source checkout missing: %v", err)
	}
	log := readTextFile(t, logPath)
	for _, want := range []string{
		"git clone --branch main",
		"-o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	for _, reject := range []string{"api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest", "releases/download", "release-gormes"} {
		if strings.Contains(log, reject) {
			t.Fatalf("source-backed install should not use %q\n%s", reject, log)
		}
	}
	if !strings.Contains(out, "source: "+checkout) {
		t.Fatalf("summary did not name managed source checkout:\n%s", out)
	}
}

func TestInstallSH_FromSourceInstallDoesNotProbeReleaseEndpoints(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)
	writeFakeDownloadTools(t, fakebin)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_FROM_SOURCE=1",
		"TMPDIR="+filepath.Join(root, "tmp"),
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s\nlog:\n%s", err, out, readTextFile(t, logPath))
	}

	managedCheckout := filepath.Join(home, ".gormes", "gormes-agent")
	if _, err := os.Stat(filepath.Join(managedCheckout, ".git")); err != nil {
		t.Fatalf("managed checkout missing: %v", err)
	}
	published := filepath.Join(home, ".local", "bin", "gormes")
	if _, err := os.Stat(published); err != nil {
		t.Fatalf("published command missing: %v", err)
	}
	if !strings.Contains(out, "Gormes installed") {
		t.Fatalf("success summary missing:\n%s", out)
	}
	logBody, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake log: %v", err)
	}
	log := string(logBody)
	for _, want := range []string{
		"git clone --branch main",
		"-o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	for _, reject := range []string{
		"api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest",
		"releases/download",
	} {
		if strings.Contains(log, reject) {
			t.Fatalf("managed source install should not probe release endpoint %q:\n%s", reject, log)
		}
	}
	if !strings.Contains(out, "source: "+managedCheckout) {
		t.Fatalf("summary missing managed checkout evidence:\n%s", out)
	}
}

func TestInstallSH_BranchOverrideBuildsManagedSourceCheckout(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)
	writeFakeDownloadTools(t, fakebin)
	archiveHash := fmt.Sprintf("%x", sha256.Sum256([]byte("fake release archive\n")))

	out, err := runInstallScriptWithArgs(t,
		[]string{"--branch", "development"},
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_RELEASE_TAG=v9.9.9",
		"GORMES_FAKE_RELEASE_TARGET=gormes-9.9.9-linux-amd64",
		"GORMES_FAKE_RELEASE_ARCHIVE_SHA="+archiveHash,
		"TMPDIR="+filepath.Join(root, "tmp"),
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	log := readTextFile(t, logPath)
	if !strings.Contains(log, "git clone --branch development") {
		t.Fatalf("branch override did not build requested source branch:\n%s", log)
	}
	for _, reject := range []string{"releases/download/v9.9.9", "release-gormes version", "api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest"} {
		if strings.Contains(log, reject) {
			t.Fatalf("branch override should skip release path %q:\n%s", reject, log)
		}
	}
	if !strings.Contains(out, "source: "+filepath.Join(home, ".gormes", "gormes-agent")) {
		t.Fatalf("branch override output missing source evidence:\n%s", out)
	}
}

func TestInstallSH_UpdatesExistingCommandEarlierOnPATH(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	staleBin := filepath.Join(root, "go", "bin")
	staleCommand := filepath.Join(staleBin, "gormes")
	if err := os.MkdirAll(staleBin, 0o755); err != nil {
		t.Fatalf("mkdir stale bin: %v", err)
	}
	writeExecutable(t, staleCommand, "#!/bin/sh\nprintf 'stale gormes\\n'\n")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+staleBin+string(os.PathListSeparator)+fakebin,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	buildBin := filepath.Join(home, ".gormes", "bin", "gormes")
	target, err := os.Readlink(staleCommand)
	if err != nil {
		t.Fatalf("stale PATH command was not replaced with a link to the managed build: %v\n%s", err, out)
	}
	if target != buildBin {
		t.Fatalf("active PATH command target = %q, want %q", target, buildBin)
	}
	if !strings.Contains(out, "updating active PATH command "+staleCommand) {
		t.Fatalf("summary/log missing active command update:\n%s", out)
	}
	if strings.Contains(out, "older install") {
		t.Fatalf("refreshed active command was still reported as older:\n%s", out)
	}
}

func TestInstallSH_RerunUpdatesPersistentManagedCheckout(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	if err := os.MkdirAll(filepath.Join(checkout, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir checkout: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(checkout, "cmd", "gormes"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/gormes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "go.mod"), []byte("module github.com/TrebuchetDynamics/gormes-agent\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	fakebin, logPath := writeFakeUnixToolchain(t, root)
	writeFakeDownloadTools(t, fakebin)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_INSTALL_FROM_SOURCE=1",
		"TMPDIR="+filepath.Join(root, "tmp"),
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
		"git status --porcelain",
		"git fetch origin",
		"git checkout main",
		"git pull --ff-only origin main",
		"-o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("managed checkout rerun missing %q:\n%s", want, log)
		}
	}
	for _, reject := range []string{"git clone --branch main", "api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest", "releases/download"} {
		if strings.Contains(log, reject) {
			t.Fatalf("managed checkout rerun should not use %q:\n%s", reject, log)
		}
	}
}

func TestInstallSH_RerunRestartsLiveGatewayWithPublishedBinary(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	if err := os.MkdirAll(filepath.Join(checkout, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir checkout: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(checkout, "cmd", "gormes"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/gormes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "go.mod"), []byte("module github.com/TrebuchetDynamics/gormes-agent\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	activeBin := filepath.Join(root, "activebin")
	activeCommand := filepath.Join(activeBin, "gormes")
	if err := os.MkdirAll(activeBin, 0o755); err != nil {
		t.Fatalf("mkdir active bin: %v", err)
	}
	writeExecutable(t, activeCommand, `#!/bin/sh
set -eu
printf 'active-gormes' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"
if [ "${1:-}" = "gateway" ] && [ "${2:-}" = "status" ]; then
  printf 'Gateway status\nruntime: running (pid=4242 active_agents=0)\n'
fi
`)
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+activeBin+string(os.PathListSeparator)+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_GATEWAY_STATUS_PID=7777",
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
		"active-gormes gateway status",
		"built-gormes gateway stop",
		"built-gormes gateway\n",
		"built-gormes gateway status",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	if strings.Index(log, "built-gormes gateway stop") > strings.Index(log, "built-gormes gateway\n") {
		t.Fatalf("gateway start occurred before stop:\n%s", log)
	}
	for _, want := range []string{
		"restarting live gateway pid=4242",
		"gateway restarted pid=4242 -> 7777",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("install output missing %q\n%s", want, out)
		}
	}
	ledger, err := os.ReadFile(filepath.Join(home, ".gormes", "install.log.jsonl"))
	if err != nil {
		t.Fatalf("read install ledger: %v", err)
	}
	for _, want := range []string{
		`"event":"install"`,
		`"old_gateway_pid":4242`,
		`"new_gateway_pid":7777`,
		`"binary_sha256"`,
	} {
		if !strings.Contains(string(ledger), want) {
			t.Fatalf("install ledger missing %q\n%s", want, ledger)
		}
	}
}

func TestInstallSH_NoRestartSkipsLiveGatewayRestart(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	writeMinimalGoModule(t, checkout)
	activeBin := filepath.Join(root, "activebin")
	if err := os.MkdirAll(activeBin, 0o755); err != nil {
		t.Fatalf("mkdir active bin: %v", err)
	}
	writeExecutable(t, filepath.Join(activeBin, "gormes"), `#!/bin/sh
printf 'active-gormes' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"; done
printf '\n' >> "$GORMES_FAKE_LOG"
if [ "${1:-}" = "gateway" ] && [ "${2:-}" = "status" ]; then
  printf '{"runtime":{"gateway_state":"running","pid":4242,"active_agents":0},"validation":{"status":"live","live":true,"pid":4242}}\n'
fi
`)
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScriptWithArgs(t,
		[]string{"--no-restart"},
		"HOME="+home,
		"PATH="+activeBin+string(os.PathListSeparator)+fakebin,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}
	log := readTextFile(t, logPath)
	if strings.Contains(log, "built-gormes gateway stop") || strings.Contains(log, "built-gormes gateway\n") {
		t.Fatalf("--no-restart still stopped/started gateway:\n%s", log)
	}
	if !strings.Contains(out, "gateway restart skipped by policy=never") {
		t.Fatalf("--no-restart output missing skip evidence:\n%s", out)
	}
}

func TestInstallSH_DryRunDoesNotBuildPublishOrRestart(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScriptWithArgs(t,
		[]string{"--dry-run", "--branch", "development"},
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("dry-run failed: %v\n%s", err, out)
	}
	if got := readTextFile(t, logPath); got != "" {
		t.Fatalf("dry-run invoked toolchain:\n%s", got)
	}
	if !strings.Contains(out, "dry run") || !strings.Contains(out, "branch: development") {
		t.Fatalf("dry-run summary missing plan:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".gormes")); err == nil {
		t.Fatalf("dry-run created install home")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat install home: %v", err)
	}
}

func TestInstallSH_VerboseDryRunShowsResolvedPlan(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	binDir := filepath.Join(root, "bin")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScriptWithArgs(t,
		[]string{"--dry-run", "--verbose", "--branch", "development", "--home", home, "--bin-dir", binDir, "--restart-gateway", "never"},
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("verbose dry-run failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"resolved install plan",
		"verbose: true",
		"platform:",
		"install_home: " + home,
		"source_mode: managed-checkout",
		"checkout: " + filepath.Join(home, "gormes-agent"),
		"managed_binary: " + filepath.Join(home, "bin", "gormes"),
		"published_binary: " + filepath.Join(binDir, "gormes"),
		"restart_gateway: never",
		"dry run",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("verbose dry-run output missing %q\n%s", want, out)
		}
	}
	if got := readTextFile(t, logPath); got != "" {
		t.Fatalf("verbose dry-run invoked toolchain:\n%s", got)
	}
}

func TestInstallSH_UninstallDelegatesToPublishedGormesWithoutBuild(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	binDir := filepath.Join(root, "bin")
	logPath := filepath.Join(root, "uninstall.log")
	writeExecutable(t, filepath.Join(binDir, "gormes"), `#!/bin/sh
set -eu
printf 'gormes' >> "$GORMES_FAKE_LOG"
for arg in "$@"; do
  printf ' %s' "$arg" >> "$GORMES_FAKE_LOG"
done
printf '\n' >> "$GORMES_FAKE_LOG"
if [ "${1:-}" = "uninstall" ]; then
  printf 'uninstall dry-run: 0 artifact(s)\n'
  exit 0
fi
exit 99
`)

	out, err := runInstallScriptWithArgs(t,
		[]string{"--bin-dir", binDir, "--uninstall", "--keep-config", "--dry-run=false", "--yes"},
		"HOME="+home,
		"PATH="+binDir,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("uninstall failed: %v\n%s", err, out)
	}
	if got, want := readTextFile(t, logPath), "gormes uninstall --keep-config --dry-run=false --yes\n"; got != want {
		t.Fatalf("uninstall command = %q, want %q", got, want)
	}
	if strings.Contains(out, "Gormes installed") || strings.Contains(out, "cloning Gormes") || strings.Contains(out, "building gormes") {
		t.Fatalf("--uninstall ran install flow:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".gormes", "install.lock")); err == nil {
		t.Fatalf("--uninstall created install lock")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat install lock: %v", err)
	}
}

func TestInstallSH_LocalBuildUsesCurrentCheckoutWithoutGitUpdate(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	local := filepath.Join(root, "local-source")
	writeMinimalGoModule(t, local)
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScriptWithArgsInDir(t, local,
		[]string{"--local"},
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err != nil {
		t.Fatalf("local install failed: %v\n%s", err, out)
	}
	log := readTextFile(t, logPath)
	if strings.Contains(log, "git clone") || strings.Contains(log, "git fetch") || strings.Contains(log, "git pull") {
		t.Fatalf("--local touched remote git checkout:\n%s", log)
	}
	if !strings.Contains(log, "-o "+filepath.Join(home, ".gormes", "bin", "gormes")+" ./cmd/gormes") {
		t.Fatalf("--local did not build gormes:\n%s", log)
	}
	if !strings.Contains(out, "source: "+local) {
		t.Fatalf("--local summary did not name local source:\n%s", out)
	}
}

func TestInstallSH_ExistingInstallLockFailsBeforeMutation(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".gormes", "install.lock"), 0o755); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
	)
	if err == nil {
		t.Fatalf("install with existing lock succeeded\n%s", out)
	}
	if got := readTextFile(t, logPath); got != "" {
		t.Fatalf("locked installer mutated toolchain:\n%s", got)
	}
	if !strings.Contains(out, "another install is already running") {
		t.Fatalf("lock failure missing evidence:\n%s", out)
	}
}

func TestInstallSH_RollsBackPublishedCommandWhenVerificationFails(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	checkout := filepath.Join(home, ".gormes", "gormes-agent")
	writeMinimalGoModule(t, checkout)
	published := filepath.Join(home, ".local", "bin", "gormes")
	writeExecutable(t, published, "#!/bin/sh\nprintf 'old-ok\\n'\n")
	fakebin, logPath := writeFakeUnixToolchain(t, root)

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_BUILT_VERSION_FAIL=1",
	)
	if err == nil {
		t.Fatalf("install succeeded despite verification failure\n%s", out)
	}
	body, readErr := os.ReadFile(published)
	if readErr != nil {
		t.Fatalf("read published after rollback: %v", readErr)
	}
	if !strings.Contains(string(body), "old-ok") {
		t.Fatalf("published command was not rolled back:\n%s", body)
	}
	if !strings.Contains(out, "rolled back") {
		t.Fatalf("rollback evidence missing:\n%s", out)
	}
	if !strings.Contains(readTextFile(t, logPath), "go build") {
		t.Fatalf("rollback test did not reach build")
	}
}

func TestInstallSH_ManagedGoDownloadVerifiesExpectedSHA256(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	fakebin := filepath.Join(root, "fakebin")
	logPath := filepath.Join(root, "toolchain.log")
	if err := os.MkdirAll(fakebin, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	linkBasicUnixTools(t, fakebin)
	writeExecutable(t, filepath.Join(fakebin, "git"), fakeGitScript())
	writeFakeDownloadTools(t, fakebin)
	_, _, managedGoTemplate := writeBootstrapTemplates(t, root, "go1.26.0")
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte("fake go tarball\n")))

	out, err := runInstallScript(t,
		"HOME="+home,
		"PATH="+fakebin,
		"GORMES_FAKE_LOG="+logPath,
		"GORMES_FAKE_MANAGED_GO_TEMPLATE="+managedGoTemplate,
		"GORMES_GO_SHA256="+expected,
		"UNAME=Linux",
	)
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"verifying Go download sha256",
		"Go download sha256 verified",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("checksum output missing %q\n%s", want, out)
		}
	}
}

func writeMinimalGoModule(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "gormes"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/gormes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/TrebuchetDynamics/gormes-agent\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
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
	gitTemplate, goTemplate, managedGoTemplate := writeBootstrapTemplates(t, root, "go1.26.0")
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
		"-o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	if !strings.Contains(out, "Gormes installed") {
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
	_, _, managedGoTemplate := writeBootstrapTemplates(t, root, "go1.26.0")
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
		"-o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
	if !strings.Contains(out, "Gormes installed") {
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
		"-o " + filepath.Join(home, ".gormes", "bin", "gormes") + " ./cmd/gormes",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("toolchain log missing %q\n%s", want, log)
		}
	}
}
