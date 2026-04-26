package tools

import (
	"strings"
	"testing"
)

func TestDetectHardlineRmRoot(t *testing.T) {
	cases := []string{
		"rm -rf /",
		"sudo rm -rf /",
		"rm -rf /etc",
		"rm -rf $HOME",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			matched, desc := DetectHardline(cmd)
			if !matched {
				t.Fatalf("DetectHardline(%q) = false, want true", cmd)
			}
			if desc == "" {
				t.Fatalf("DetectHardline(%q) returned empty description", cmd)
			}
		})
	}
}

func TestDetectHardlineMkfsAndDD(t *testing.T) {
	cases := []string{
		"mkfs.ext4 /dev/sda1",
		"dd if=/dev/zero of=/dev/sdb",
		"> /dev/sda",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			matched, desc := DetectHardline(cmd)
			if !matched {
				t.Fatalf("DetectHardline(%q) = false, want true", cmd)
			}
			if desc == "" {
				t.Fatalf("DetectHardline(%q) returned empty description", cmd)
			}
		})
	}
}

func TestDetectHardlineForkBomb(t *testing.T) {
	matched, desc := DetectHardline(":(){ :|: & };:")
	if !matched {
		t.Fatalf("DetectHardline(fork bomb) = false, want true")
	}
	if !strings.Contains(strings.ToLower(desc), "fork bomb") {
		t.Fatalf("DetectHardline fork bomb description = %q, want contains 'fork bomb'", desc)
	}
}

func TestDetectHardlineShutdownAndReboot(t *testing.T) {
	matchCases := []string{
		"reboot",
		"shutdown -h now",
		"systemctl poweroff",
		"init 0",
		"telinit 6",
	}
	noMatchCases := []string{
		"echo reboot",
		"grep shutdown logs",
		"systemctl status",
	}
	for _, cmd := range matchCases {
		t.Run("match/"+cmd, func(t *testing.T) {
			ok, desc := DetectHardline(cmd)
			if !ok {
				t.Fatalf("DetectHardline(%q) = false, want true", cmd)
			}
			if desc == "" {
				t.Fatalf("DetectHardline(%q) returned empty description", cmd)
			}
		})
	}
	for _, cmd := range noMatchCases {
		t.Run("nomatch/"+cmd, func(t *testing.T) {
			ok, desc := DetectHardline(cmd)
			if ok {
				t.Fatalf("DetectHardline(%q) = true (desc=%q), want false", cmd, desc)
			}
			if desc != "" {
				t.Fatalf("DetectHardline(%q) returned description %q, want empty", cmd, desc)
			}
		})
	}
}

func TestDetectHardlineKillAll(t *testing.T) {
	matchCases := []string{
		"kill -1",
		"sudo kill -1",
	}
	noMatchCases := []string{
		"kill -9 1234",
	}
	for _, cmd := range matchCases {
		t.Run("match/"+cmd, func(t *testing.T) {
			ok, desc := DetectHardline(cmd)
			if !ok {
				t.Fatalf("DetectHardline(%q) = false, want true", cmd)
			}
			if desc == "" {
				t.Fatalf("DetectHardline(%q) returned empty description", cmd)
			}
		})
	}
	for _, cmd := range noMatchCases {
		t.Run("nomatch/"+cmd, func(t *testing.T) {
			ok, desc := DetectHardline(cmd)
			if ok {
				t.Fatalf("DetectHardline(%q) = true (desc=%q), want false", cmd, desc)
			}
		})
	}
}

func TestDetectHardlineBenign(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
	}{
		{"empty", ""},
		{"ls", "ls"},
		{"echo hi", "echo hi"},
		{"mkfs_helper", "mkfs_helper"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, desc := DetectHardline(tc.cmd)
			if ok {
				t.Fatalf("DetectHardline(%q) = true (desc=%q), want false", tc.cmd, desc)
			}
			if desc != "" {
				t.Fatalf("DetectHardline(%q) returned description %q, want empty", tc.cmd, desc)
			}
		})
	}
}
