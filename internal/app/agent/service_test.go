package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
)

type fakeRegistry struct {
	records []goncho.AgentRecord
	bound   map[goncho.BindingMatch]string
}

func (f *fakeRegistry) Bind(_ context.Context, agentID string, match goncho.BindingMatch) error {
	if f.bound == nil {
		f.bound = map[goncho.BindingMatch]string{}
	}
	f.bound[match] = agentID
	return nil
}
func (f *fakeRegistry) Unbind(_ context.Context, match goncho.BindingMatch) error {
	delete(f.bound, match)
	return nil
}
func (f *fakeRegistry) Resolve(_ context.Context, match goncho.BindingMatch) (string, bool, error) {
	id, ok := f.bound[match]
	return id, ok, nil
}
func (f *fakeRegistry) Create(_ context.Context, opts goncho.CreateAgentOptions) (goncho.AgentRecord, error) {
	if opts.Name == "!!!" {
		return goncho.AgentRecord{}, goncho.ErrAgentIDInvalid
	}
	record := goncho.AgentRecord{ID: "research-bot", Name: opts.Name, Persona: opts.Persona, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	f.records = append(f.records, record)
	return record, nil
}
func (f *fakeRegistry) List(context.Context) ([]goncho.AgentRecord, error) { return f.records, nil }

func TestRunSpawnWritesJSONWithBuildProvenance(t *testing.T) {
	reg := &fakeRegistry{}
	var out bytes.Buffer
	err := RunSpawn(context.Background(), &out, "Research Bot", SpawnOptions{Persona: "literature review", JSON: true}, testOptions(reg))
	if err != nil {
		t.Fatalf("RunSpawn: %v", err)
	}
	var got struct {
		Build BuildProvenance `json:"build"`
		Agent struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Persona string `json:"persona"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.Build.Version != "v-test" || got.Agent.ID != "research-bot" || got.Agent.Persona != "literature review" {
		t.Fatalf("report = %+v", got)
	}
}

func TestRunInspectUnboundTextReturnsExitCode(t *testing.T) {
	reg := &fakeRegistry{bound: map[goncho.BindingMatch]string{}}
	var stdout, stderr bytes.Buffer
	err := RunInspect(context.Background(), &stdout, &stderr, BindingMatch{Channel: "telegram", PeerKind: "group", PeerID: "-100"}, false, testOptions(reg))
	if err == nil {
		t.Fatal("RunInspect error = nil")
	}
	coded, ok := err.(interface{ ExitCode() int })
	if !ok || coded.ExitCode() != 1 {
		t.Fatalf("exit code = %#v, want 1", coded)
	}
	if !strings.Contains(stderr.String(), "agent_not_bound") || !strings.Contains(err.Error(), "agent_not_bound") {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout.String(), stderr.String(), err)
	}
}

func TestRunSpawnInvalidNameReturnsExitCode2(t *testing.T) {
	var out bytes.Buffer
	err := RunSpawn(context.Background(), &out, "!!!", SpawnOptions{}, testOptions(&fakeRegistry{}))
	if err == nil || !errors.Is(err, goncho.ErrAgentIDInvalid) && !strings.Contains(err.Error(), "agent_id_invalid") {
		t.Fatalf("err = %v, want agent_id_invalid", err)
	}
	coded, ok := err.(interface{ ExitCode() int })
	if !ok || coded.ExitCode() != 2 {
		t.Fatalf("exit code = %#v, want 2", coded)
	}
}

func testOptions(reg *fakeRegistry) Options {
	return Options{
		BuildProvenance: func() BuildProvenance { return BuildProvenance{Version: "v-test", GitCommit: "g-test"} },
		OpenRegistry:    func() (Registry, func(), error) { return reg, func() {}, nil },
	}
}
