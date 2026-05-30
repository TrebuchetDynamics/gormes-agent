package rpcmode

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunRPCModeWritesHeaderAndHandlesPromptStream(t *testing.T) {
	rt := &fakeRPCRuntime{
		header: RPCRecord{"session_id": "s-1"},
		promptEvents: []RPCRecord{
			{"type": "assistant", "content": "hello"},
		},
	}
	input := `{"id":"p1","type":"prompt","message":"hi","images":["img"],"streamingBehavior":"stream"}` + "\n"
	var out strings.Builder

	if err := RunRPCMode(context.Background(), RPCModeOptions{In: strings.NewReader(input), Out: &out, Runtime: rt}); err != nil {
		t.Fatalf("RunRPCMode: %v", err)
	}

	records := decodeJSONL(t, out.String())
	if len(records) != 3 {
		t.Fatalf("records = %#v, want header, response, stream event", records)
	}
	if records[0]["type"] != "session" || records[0]["version"].(float64) != 1 || records[0]["session_id"] != "s-1" {
		t.Fatalf("header = %#v, want defaulted session header", records[0])
	}
	if records[1]["type"] != "response" || records[1]["id"] != "p1" || records[1]["command"] != "prompt" || records[1]["success"] != true {
		t.Fatalf("prompt response = %#v", records[1])
	}
	if records[2]["type"] != "assistant" || records[2]["content"] != "hello" {
		t.Fatalf("stream event = %#v", records[2])
	}
	if rt.lastPrompt.Message != "hi" || rt.lastPrompt.StreamingBehavior != "stream" || len(rt.lastPrompt.Images) != 1 {
		t.Fatalf("prompt request = %+v, want message/images/streaming behavior", rt.lastPrompt)
	}
}

func TestRunRPCModeReportsParseUnsupportedAndQueueUpdates(t *testing.T) {
	rt := &fakeRPCRuntime{queue: RPCQueueState{Steering: []string{"nudge"}, FollowUp: []string{"next"}}}
	input := strings.Join([]string{
		`not-json`,
		`{"id":"x","type":"set_model"}`,
		`{"id":"s","type":"steer","message":"nudge"}`,
		`{"id":"f","type":"follow_up","message":"next"}`,
	}, "\n") + "\n"
	var out strings.Builder

	if err := RunRPCMode(context.Background(), RPCModeOptions{In: strings.NewReader(input), Out: &out, Runtime: rt}); err != nil {
		t.Fatalf("RunRPCMode: %v", err)
	}

	records := decodeJSONL(t, out.String())
	if len(records) != 7 {
		t.Fatalf("records = %#v, want header plus six command records", records)
	}
	if records[1]["command"] != "parse" || records[1]["success"] != false {
		t.Fatalf("parse error record = %#v", records[1])
	}
	if records[2]["command"] != "set_model" || records[2]["success"] != false || !strings.Contains(records[2]["error"].(string), "unsupported") {
		t.Fatalf("unsupported record = %#v", records[2])
	}
	if records[3]["command"] != "steer" || records[3]["success"] != true {
		t.Fatalf("steer response = %#v", records[3])
	}
	if records[4]["type"] != "queue_update" || records[4]["steering"].([]any)[0] != "nudge" || records[4]["followUp"].([]any)[0] != "next" {
		t.Fatalf("steer queue update = %#v", records[4])
	}
	if records[5]["command"] != "follow_up" || records[5]["success"] != true {
		t.Fatalf("follow_up response = %#v", records[5])
	}
	if records[6]["type"] != "queue_update" || records[6]["steering"].([]any)[0] != "nudge" || records[6]["followUp"].([]any)[0] != "next" {
		t.Fatalf("follow_up queue update = %#v", records[6])
	}
}

func decodeJSONL(t *testing.T, raw string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("decode %q: %v", line, err)
		}
		records = append(records, rec)
	}
	return records
}

type fakeRPCRuntime struct {
	header       RPCRecord
	state        RPCRecord
	messages     []RPCRecord
	promptEvents []RPCRecord
	queue        RPCQueueState
	lastPrompt   RPCPromptRequest
}

func (f *fakeRPCRuntime) Header(context.Context) RPCRecord { return f.header }

func (f *fakeRPCRuntime) State(context.Context) (RPCRecord, error) { return f.state, nil }

func (f *fakeRPCRuntime) Messages(context.Context) ([]RPCRecord, error) { return f.messages, nil }

func (f *fakeRPCRuntime) Prompt(_ context.Context, req RPCPromptRequest) (<-chan RPCRecord, error) {
	f.lastPrompt = req
	ch := make(chan RPCRecord, len(f.promptEvents))
	for _, ev := range f.promptEvents {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (f *fakeRPCRuntime) Steer(context.Context, string) (RPCQueueState, error) { return f.queue, nil }

func (f *fakeRPCRuntime) FollowUp(context.Context, string) (RPCQueueState, error) {
	return f.queue, nil
}

func (f *fakeRPCRuntime) Abort(context.Context) error { return nil }
