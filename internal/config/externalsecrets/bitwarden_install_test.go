package externalsecrets

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallBitwardenBWSVerifiesChecksumAndInstallsAtomically(t *testing.T) {
	home := t.TempDir()
	zipBytes := bitwardenTestZip(t, "bws", "#!/bin/sh\necho bws 2.0.0\n")
	asset := "bws-x86_64-unknown-linux-gnu-2.0.0.zip"
	checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(zipBytes), asset)

	path, err := InstallBitwardenBWS(context.Background(), BitwardenInstallOptions{
		HomeDir: home,
		System:  "linux",
		Machine: "amd64",
		Libc:    "gnu",
		Download: func(_ context.Context, url string) ([]byte, error) {
			if strings.HasSuffix(url, asset) {
				return zipBytes, nil
			}
			if strings.HasSuffix(url, BitwardenChecksumName) {
				return []byte(checksum), nil
			}
			return nil, fmt.Errorf("unexpected url %s", url)
		},
	})
	if err != nil {
		t.Fatalf("InstallBitwardenBWS: %v", err)
	}
	if path != filepath.Join(home, "bin", "bws") {
		t.Fatalf("path = %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat installed bws: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("installed bws is not executable: %v", info.Mode())
	}
	body, _ := os.ReadFile(path)
	if !bytes.Contains(body, []byte("bws 2.0.0")) {
		t.Fatalf("installed body mismatch: %q", body)
	}
}

func TestInstallBitwardenBWSRejectsChecksumMismatchBeforeReplacingExisting(t *testing.T) {
	home := t.TempDir()
	managed := filepath.Join(home, "bin", "bws")
	if err := os.MkdirAll(filepath.Dir(managed), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managed, []byte("existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	zipBytes := bitwardenTestZip(t, "bws", "new")
	_, err := InstallBitwardenBWS(context.Background(), BitwardenInstallOptions{
		HomeDir: home,
		Force:   true,
		System:  "linux",
		Machine: "amd64",
		Libc:    "gnu",
		Download: func(_ context.Context, url string) ([]byte, error) {
			if strings.HasSuffix(url, ".zip") {
				return zipBytes, nil
			}
			return []byte("deadbeef  bws-x86_64-unknown-linux-gnu-2.0.0.zip\n"), nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
	body, _ := os.ReadFile(managed)
	if string(body) != "existing" {
		t.Fatalf("existing binary replaced on checksum failure: %q", body)
	}
}

func TestInstallBitwardenBWSRejectsZipSlip(t *testing.T) {
	home := t.TempDir()
	zipBytes := bitwardenTestZip(t, "../bws", "evil")
	asset := "bws-x86_64-unknown-linux-gnu-2.0.0.zip"
	checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(zipBytes), asset)
	_, err := InstallBitwardenBWS(context.Background(), BitwardenInstallOptions{
		HomeDir: home,
		System:  "linux",
		Machine: "amd64",
		Libc:    "gnu",
		Download: func(_ context.Context, url string) ([]byte, error) {
			if strings.HasSuffix(url, ".zip") {
				return zipBytes, nil
			}
			return []byte(checksum), nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe archive member") {
		t.Fatalf("err = %v, want unsafe archive member", err)
	}
}

func TestInstallBitwardenBWSNoForceKeepsExistingAndPlatformAssets(t *testing.T) {
	home := t.TempDir()
	managed := filepath.Join(home, "bin", "bws")
	if err := os.MkdirAll(filepath.Dir(managed), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managed, []byte("existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	called := false
	path, err := InstallBitwardenBWS(context.Background(), BitwardenInstallOptions{
		HomeDir: home,
		Download: func(context.Context, string) ([]byte, error) {
			called = true
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("InstallBitwardenBWS existing: %v", err)
	}
	if path != managed || called {
		t.Fatalf("existing path=%q downloadCalled=%v", path, called)
	}

	cases := []struct{ goos, goarch, libc, want string }{
		{"linux", "amd64", "gnu", "bws-x86_64-unknown-linux-gnu-2.0.0.zip"},
		{"linux", "arm64", "musl", "bws-aarch64-unknown-linux-musl-2.0.0.zip"},
		{"darwin", "arm64", "", "bws-macos-universal-2.0.0.zip"},
		{"windows", "amd64", "", "bws-x86_64-pc-windows-msvc-2.0.0.zip"},
		{"windows", "arm64", "", "bws-aarch64-pc-windows-msvc-2.0.0.zip"},
	}
	for _, tc := range cases {
		got, err := BitwardenAssetName(BitwardenInstallOptions{System: tc.goos, Machine: tc.goarch, Libc: tc.libc})
		if err != nil || got != tc.want {
			t.Fatalf("asset %s/%s/%s = %q, %v; want %q", tc.goos, tc.goarch, tc.libc, got, err, tc.want)
		}
	}
	if _, err := BitwardenAssetName(BitwardenInstallOptions{System: "plan9", Machine: "amd64"}); err == nil || !strings.Contains(err.Error(), "unsupported platform") {
		t.Fatalf("unsupported err = %v", err)
	}
}

func bitwardenTestZip(t *testing.T, name, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}
