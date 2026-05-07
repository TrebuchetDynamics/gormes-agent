package cli

import (
	"strings"
	"testing"
)

func TestSystemdUnitPATHPreservesWSLInteropOnlyUnderWSL(t *testing.T) {
	hostPath := strings.Join([]string{
		"/usr/local/bin",
		"/mnt/c/WINDOWS/system32",
		"/mnt/c/WINDOWS/System32/WindowsPowerShell/v1.0/",
	}, ":")

	wslLine := SystemdPATHEnvironment(SystemdPATHOptions{
		BasePath:    "%h/.local/bin:/usr/local/bin:/usr/bin:/bin",
		HostPath:    hostPath,
		WSLDetected: true,
	})
	for _, want := range []string{
		"Environment=PATH=",
		"/mnt/c/WINDOWS/system32",
		"/mnt/c/WINDOWS/System32/WindowsPowerShell/v1.0/",
	} {
		if !strings.Contains(wslLine, want) {
			t.Fatalf("WSL PATH line missing %q:\n%s", want, wslLine)
		}
	}

	nonWSLLine := SystemdPATHEnvironment(SystemdPATHOptions{
		BasePath:    "%h/.local/bin:/usr/local/bin:/usr/bin:/bin",
		HostPath:    hostPath,
		WSLDetected: false,
	})
	if strings.Contains(nonWSLLine, "/mnt/c/WINDOWS") {
		t.Fatalf("non-WSL PATH line inherited WSL interop entries:\n%s", nonWSLLine)
	}
}
