package channelgoncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	"github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	gonchotools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/goncho"
)

// NewService constructs a goncho.Service wired for channel runtime use. It
// attaches a Hermes-backed DialecticCaller when a provider client is available,
// enabling in-process honcho_reasoning without an external Honcho process.
func NewService(db *sql.DB, cfg goncho.Config, log *slog.Logger, client llm.Client, model string) *goncho.Service {
	svc := goncho.NewService(db, cfg, log)
	if client != nil {
		svc.SetDialecticCaller(NewHermesDialecticCaller(client, model))
	}
	return svc
}

// RegisterTools wires honcho_* memory tools onto the agent registry backed by
// the given goncho service. This is the shared entry point all channels
// (Telegram, WhatsApp, Slack, Discord) call to enable memory.
func RegisterTools(reg *tools.Registry, svc *goncho.Service) {
	gonchotools.RegisterHonchoTools(reg, svc)
}

// RegisterGormesTools wires the public Goncho v0.1.x Gormes adapter tools onto
// the registry. These are the stable goncho_* tools exposed by the released
// github.com/TrebuchetDynamics/goncho/integration/gormes package.
func RegisterGormesTools(reg *tools.Registry, mem *gormesgoncho.Runtime) {
	if reg == nil || mem == nil {
		return
	}
	reg.MustRegister(mem.ContextTool)
	reg.MustRegister(mem.SearchTool)
	reg.MustRegister(mem.RememberTool)
	reg.MustRegister(mem.ReviewTool)
	reg.MustRegister(mem.HandoffTool)
}

func FormatStatus(status gormesgoncho.Status) string {
	ready := "unavailable"
	if status.Ready {
		ready = "ready"
	}
	tools := append([]string(nil), status.ToolNames...)
	sort.Strings(tools)
	return fmt.Sprintf("goncho: %s workspace_id=%s observer_id=%s database=%s tools=%s", ready, status.WorkspaceID, status.ObserverID, status.DatabasePath, strings.Join(tools, ","))
}

// HermesDialecticCaller adapts Gormes' native provider client to the Goncho
// DialecticCaller seam. It keeps honcho_reasoning fully in-process: no Python
// hermes-agent or external Honcho runtime is launched for synthesis.
type HermesDialecticCaller struct {
	client llm.Client
	model  string
}

// NewHermesDialecticCaller returns a DialecticCaller backed by the native
// Hermes-compatible chat client used elsewhere in Gormes.
func NewHermesDialecticCaller(client llm.Client, model string) *HermesDialecticCaller {
	return &HermesDialecticCaller{client: client, model: strings.TrimSpace(model)}
}

// Chat sends the supplied Goncho context prompt and query through the native
// provider client, collecting streamed text tokens into one synthesized answer.
func (c *HermesDialecticCaller) Chat(ctx context.Context, peer string, systemPrompt string, query string) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("goncho: dialectic caller client is nil")
	}
	stream, err := c.client.OpenStream(ctx, llm.ChatRequest{
		Model:     c.model,
		Stream:    true,
		SessionID: dialecticSessionID(peer),
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: query},
		},
	})
	if err != nil {
		return "", fmt.Errorf("goncho: dialectic provider stream: %w", err)
	}
	defer stream.Close()

	var answer strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("goncho: dialectic provider recv: %w", err)
		}
		if ev.Kind == llm.EventToken {
			answer.WriteString(ev.Token)
		}
	}
	out := strings.TrimSpace(answer.String())
	if out == "" {
		return "", errors.New("goncho: no dialectic answer from provider")
	}
	return out, nil
}

func dialecticSessionID(peer string) string {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		peer = "unknown"
	}
	return "goncho-dialectic:" + peer
}
