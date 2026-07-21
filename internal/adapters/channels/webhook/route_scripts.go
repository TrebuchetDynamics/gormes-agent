package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

const (
	defaultRouteScriptTimeout = 30 * time.Second
	maxRouteScriptStdout      = 1 << 20
	maxRouteScriptStderr      = 64 << 10
)

func clampRouteScriptTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 || timeout > defaultRouteScriptTimeout {
		return defaultRouteScriptTimeout
	}
	return timeout
}

func runRouteScript(ctx context.Context, profileHome, script string, payload map[string]any, timeout time.Duration) (map[string]any, bool) {
	path, ok := resolveRouteScriptPath(profileHome, script)
	if !ok {
		return nil, false
	}
	stdin, err := json.Marshal(payload)
	if err != nil {
		return nil, false
	}

	runCtx, cancel := context.WithTimeout(ctx, clampRouteScriptTimeout(timeout))
	defer cancel()
	cmd := exec.CommandContext(runCtx, path)
	cmd.Dir = filepath.Dir(path)
	cmd.Env = routeScriptEnvironment(profileHome)
	cmd.Stdin = bytes.NewReader(stdin)
	stdout := newRouteScriptBuffer(maxRouteScriptStdout)
	stderr := newRouteScriptBuffer(maxRouteScriptStderr)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil || runCtx.Err() != nil || stdout.overflow || stderr.overflow {
		return nil, false
	}

	output := strings.TrimSpace(redaction.RedactSecrets(stdout.String()))
	if output == "" || output == "[SILENT]" {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		transformed := cloneScriptPayload(payload)
		transformed["script_output"] = output
		return transformed, true
	}
	transformed, ok := decoded.(map[string]any)
	if !ok || scriptIgnored(transformed) {
		return nil, false
	}
	return transformed, true
}

func resolveRouteScriptPath(profileHome, script string) (string, bool) {
	profileHome = strings.TrimSpace(profileHome)
	raw := strings.TrimSpace(script)
	if profileHome == "" || raw == "" {
		return "", false
	}

	rel := raw
	for _, alias := range []string{"~/.hermes/scripts", "~/.gormes/scripts"} {
		if raw == alias {
			rel = "."
			break
		}
		if strings.HasPrefix(raw, alias+"/") {
			rel = strings.TrimPrefix(raw, alias+"/")
			break
		}
	}
	if strings.HasPrefix(rel, "~") || filepath.IsAbs(rel) {
		return "", false
	}

	home, err := filepath.Abs(profileHome)
	if err != nil {
		return "", false
	}
	root := filepath.Join(home, "scripts")
	candidate := filepath.Join(root, filepath.FromSlash(rel))
	if !pathWithin(root, candidate) {
		return "", false
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	rootInfo, err := os.Stat(resolvedRoot)
	if err != nil || !rootInfo.IsDir() {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil || !pathWithin(resolvedRoot, resolved) {
		return "", false
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", false
	}
	return resolved, true
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func routeScriptEnvironment(profileHome string) []string {
	env := []string{
		"HOME=" + profileHome,
		"GORMES_HOME=" + profileHome,
		"HERMES_HOME=" + profileHome,
	}
	if value := os.Getenv("PATH"); value != "" {
		env = append(env, "PATH="+value)
	}
	if runtime.GOOS == "windows" {
		env = append(env, "USERPROFILE="+profileHome)
		for _, key := range []string{"SystemRoot", "ComSpec", "PATHEXT"} {
			if value := os.Getenv(key); value != "" {
				env = append(env, key+"="+value)
			}
		}
	}
	return env
}

func scriptIgnored(payload map[string]any) bool {
	for _, key := range []string{"[SILENT]", "__hermes_ignore__"} {
		if ignored, ok := payload[key].(bool); ok && ignored {
			return true
		}
	}
	return false
}

func cloneScriptPayload(payload map[string]any) map[string]any {
	cloned := make(map[string]any, len(payload)+1)
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

type routeScriptBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
}

func newRouteScriptBuffer(limit int) *routeScriptBuffer {
	return &routeScriptBuffer{limit: limit}
}

func (buffer *routeScriptBuffer) String() string {
	return buffer.buffer.String()
}

func (buffer *routeScriptBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining < len(data) {
		buffer.overflow = true
		if remaining < 0 {
			remaining = 0
		}
		data = data[:remaining]
	}
	_, _ = buffer.buffer.Write(data)
	return original, nil
}
