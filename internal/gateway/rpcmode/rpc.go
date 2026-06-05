package rpcmode

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

// RPCRecord is one JSONL object on the Gormes-owned stdio RPC stream.
type RPCRecord map[string]any

// RPCPromptRequest is the prompt subset accepted by the first Gormes RPC slice.
type RPCPromptRequest struct {
	ID                string
	Message           string
	Images            []any
	StreamingBehavior string
}

// RPCQueueState mirrors Pi's queue_update shape while remaining Gormes-owned.
type RPCQueueState struct {
	Steering []string `json:"steering"`
	FollowUp []string `json:"followUp"`
}

// RPCRuntime is the small runtime seam behind the stdio JSONL loop. Tests can
// provide a fake runtime; production binds this to the local kernel from
// cmd/gormes without starting an HTTP listener or a Pi subprocess.
type RPCRuntime interface {
	Header(context.Context) RPCRecord
	State(context.Context) (RPCRecord, error)
	Messages(context.Context) ([]RPCRecord, error)
	Prompt(context.Context, RPCPromptRequest) (<-chan RPCRecord, error)
	Steer(context.Context, string) (RPCQueueState, error)
	FollowUp(context.Context, string) (RPCQueueState, error)
	Abort(context.Context) error
}

type RPCModeOptions struct {
	In      io.Reader
	Out     io.Writer
	Runtime RPCRuntime
}

// RunRPCMode runs a strict LF-framed stdin/stdout JSONL protocol. Stderr is not
// touched; malformed input and unsupported commands are reported as structured
// response records on stdout and do not terminate the loop.
func RunRPCMode(ctx context.Context, opts RPCModeOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.In == nil {
		opts.In = strings.NewReader("")
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	if opts.Runtime == nil {
		return errors.New("rpc runtime is required")
	}
	writer := &rpcJSONLWriter{out: opts.Out}
	if err := writer.write(rpcDefaultHeader(opts.Runtime.Header(ctx))); err != nil {
		return err
	}

	var wg sync.WaitGroup
	defer wg.Wait()
	reader := bufio.NewReader(opts.In)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			if handleErr := handleRPCInputLine(ctx, line, opts.Runtime, writer, &wg); handleErr != nil {
				return handleErr
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func handleRPCInputLine(ctx context.Context, line string, runtime RPCRuntime, writer *rpcJSONLWriter, wg *sync.WaitGroup) error {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return writer.write(rpcError("", "parse", fmt.Sprintf("Failed to parse command: %v", err)))
	}
	id := rpcString(raw["id"])
	command := rpcString(raw["type"])
	switch command {
	case "prompt":
		message := rpcString(raw["message"])
		if strings.TrimSpace(message) == "" {
			return writer.write(rpcError(id, "prompt", "message is required"))
		}
		events, err := runtime.Prompt(ctx, RPCPromptRequest{
			ID:                id,
			Message:           message,
			Images:            rpcAnySlice(raw["images"]),
			StreamingBehavior: rpcString(raw["streamingBehavior"]),
		})
		if err != nil {
			return writer.write(rpcError(id, "prompt", err.Error()))
		}
		if err := writer.write(rpcSuccess(id, "prompt", nil)); err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ev := range events {
				_ = writer.write(ev)
			}
		}()
		return nil
	case "steer":
		state, err := runtime.Steer(ctx, rpcString(raw["message"]))
		if err != nil {
			return writer.write(rpcError(id, "steer", err.Error()))
		}
		if err := writer.write(rpcSuccess(id, "steer", nil)); err != nil {
			return err
		}
		return writer.write(RPCRecord{"type": "queue_update", "steering": state.Steering, "followUp": state.FollowUp})
	case "follow_up":
		state, err := runtime.FollowUp(ctx, rpcString(raw["message"]))
		if err != nil {
			return writer.write(rpcError(id, "follow_up", err.Error()))
		}
		if err := writer.write(rpcSuccess(id, "follow_up", nil)); err != nil {
			return err
		}
		return writer.write(RPCRecord{"type": "queue_update", "steering": state.Steering, "followUp": state.FollowUp})
	case "abort":
		if err := runtime.Abort(ctx); err != nil {
			return writer.write(rpcError(id, "abort", err.Error()))
		}
		return writer.write(rpcSuccess(id, "abort", nil))
	case "get_state":
		state, err := runtime.State(ctx)
		if err != nil {
			return writer.write(rpcError(id, "get_state", err.Error()))
		}
		return writer.write(rpcSuccess(id, "get_state", state))
	case "get_messages":
		messages, err := runtime.Messages(ctx)
		if err != nil {
			return writer.write(rpcError(id, "get_messages", err.Error()))
		}
		return writer.write(rpcSuccess(id, "get_messages", RPCRecord{"messages": messages}))
	case "set_model", "cycle_model", "get_available_models", "set_thinking_level", "cycle_thinking_level", "set_steering_mode", "set_follow_up_mode", "compact", "set_auto_compaction", "set_auto_retry", "abort_retry", "bash", "abort_bash", "get_session_stats", "export_html", "new_session", "switch_session", "fork", "clone", "get_fork_messages", "get_last_assistant_text", "set_session_name", "get_commands":
		return writer.write(rpcError(id, command, fmt.Sprintf("unsupported command %q in this Gormes RPC slice", command)))
	default:
		if command == "" {
			command = "unknown"
		}
		return writer.write(rpcError(id, command, "Unknown command: "+command))
	}
}

type rpcJSONLWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func (w *rpcJSONLWriter) write(record RPCRecord) error {
	if record == nil {
		return nil
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.out.Write(raw); err != nil {
		return err
	}
	_, err = w.out.Write([]byte("\n"))
	return err
}

func rpcDefaultHeader(header RPCRecord) RPCRecord {
	if header == nil {
		header = RPCRecord{}
	}
	if _, ok := header["type"]; !ok {
		header["type"] = "session"
	}
	if _, ok := header["version"]; !ok {
		header["version"] = 1
	}
	return header
}

func rpcSuccess(id, command string, data any) RPCRecord {
	rec := RPCRecord{"type": "response", "command": command, "success": true}
	if id != "" {
		rec["id"] = id
	}
	if data != nil {
		rec["data"] = data
	}
	return rec
}

func rpcError(id, command, message string) RPCRecord {
	rec := RPCRecord{"type": "response", "command": command, "success": false, "error": message}
	if id != "" {
		rec["id"] = id
	}
	return rec
}

func rpcString(v any) string {
	s, _ := v.(string)
	return s
}

func rpcAnySlice(v any) []any {
	items, _ := v.([]any)
	if len(items) == 0 {
		return nil
	}
	return items
}
