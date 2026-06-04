package jsoninput

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestEmitWritesStructuredErrorAndExitCode(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{Use: "fixture"}
	cmd.SetOut(&out)

	err := Emit(cmd, "missing_argument", "requires <name>", BuildProvenance{Version: "v-test", GitCommit: "g-test"})
	coded, ok := err.(interface{ ExitCode() int })
	if !ok || coded.ExitCode() != 1 {
		t.Fatalf("ExitCode = %#v, want 1", coded)
	}

	var got ErrorReportJSON
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\n%s", jsonErr, out.String())
	}
	if got.Build.Version != "v-test" || got.Build.GitCommit != "g-test" || got.Action != "missing_argument" || got.Error != "requires <name>" {
		t.Fatalf("report = %+v", got)
	}
}

func TestArgsIncludeJSONFlag(t *testing.T) {
	if !ArgsIncludeJSONFlag([]string{"config", "--json"}) {
		t.Fatal("--json not detected")
	}
	if !ArgsIncludeJSONFlag([]string{"config", "--json=true"}) {
		t.Fatal("--json=true not detected")
	}
	if ArgsIncludeJSONFlag([]string{"config", "json"}) {
		t.Fatal("bare json should not count as flag")
	}
}
