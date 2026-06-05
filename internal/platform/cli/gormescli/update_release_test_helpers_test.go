package gormescli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func currentUpdateVersionForTest() string {
	return defaultUpdateBuildProvenance().Version
}

func nextPatchVersionForTest(t *testing.T) string {
	t.Helper()
	parts := strings.Split(currentUpdateVersionForTest(), ".")
	if len(parts) != 3 {
		t.Fatalf("current update version = %q, want major.minor.patch", currentUpdateVersionForTest())
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatalf("current update version patch = %q, want integer: %v", parts[2], err)
	}
	return fmt.Sprintf("%s.%s.%0*d", parts[0], parts[1], len(parts[2]), patch+1)
}

func nextPatchReleaseMetadataForTest(t *testing.T, commit string) cli.UpdateReleaseMetadata {
	t.Helper()
	version := nextPatchVersionForTest(t)
	return cli.UpdateReleaseMetadata{Version: version, Tag: "v" + version, GitCommit: commit}
}

func nextPatchReleaseArtifactForTest(t *testing.T, goos, goarch string) string {
	t.Helper()
	return fmt.Sprintf("gormes-%s-%s-%s.tar.gz", nextPatchVersionForTest(t), goos, goarch)
}
