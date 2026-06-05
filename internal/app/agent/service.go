package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agenttemplate"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type Options struct {
	DefaultResetTarget string
	BuildProvenance    func() BuildProvenance
	OpenRegistry       func() (Registry, func(), error)
}

type Registry interface {
	Bind(context.Context, string, goncho.BindingMatch) error
	Unbind(context.Context, goncho.BindingMatch) error
	Resolve(context.Context, goncho.BindingMatch) (string, bool, error)
	Create(context.Context, goncho.CreateAgentOptions) (goncho.AgentRecord, error)
	List(context.Context) ([]goncho.AgentRecord, error)
}

type exitCodeError struct {
	code int
	err  error
}

func NewExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return exitCodeError{code: code, err: err}
}

func (e exitCodeError) Error() string { return e.err.Error() }
func (e exitCodeError) Unwrap() error { return e.err }
func (e exitCodeError) ExitCode() int { return e.code }

type ResetOptions struct {
	Target string
	Force  bool
	DryRun bool
	JSON   bool
}

type SpawnOptions struct {
	Persona string
	JSON    bool
}

type BindingMatch struct {
	Channel  string
	PeerKind string
	PeerID   string
	ThreadID string
}

type resetReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Target string          `json:"target"`
	DryRun bool            `json:"dry_run"`
	Files  []resetFileJSON `json:"files"`
}

type resetFileJSON struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

type bindingReportJSON struct {
	Build   BuildProvenance  `json:"build"`
	Match   bindingMatchJSON `json:"match"`
	AgentID string           `json:"agent_id,omitempty"`
	Bound   *bool            `json:"bound,omitempty"`
	Removed *bool            `json:"removed,omitempty"`
}

type bindingMatchJSON struct {
	Channel  string `json:"channel"`
	PeerKind string `json:"peer_kind"`
	PeerID   string `json:"peer_id"`
	ThreadID string `json:"thread_id,omitempty"`
}

type recordJSON struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Persona   string `json:"persona,omitempty"`
	CreatedAt string `json:"created_at"`
}

type spawnReportJSON struct {
	Build BuildProvenance `json:"build"`
	Agent recordJSON      `json:"agent"`
}

type listReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Agents []recordJSON    `json:"agents"`
}

func RunBind(ctx context.Context, out io.Writer, agentID string, match BindingMatch, asJSON bool, opts Options) error {
	reg, cleanup, err := openRegistry(opts)
	if err != nil {
		return err
	}
	defer cleanup()

	m := BindingMatchToGoncho(match)
	if err := reg.Bind(ctx, agentID, m); err != nil {
		return fmt.Errorf("gormes agent bind: %w", err)
	}
	if asJSON {
		return writeJSON(out, bindingReportJSON{Build: build(opts), Match: bindingMatchToJSON(m), AgentID: agentID})
	}
	fmt.Fprintf(out, "bound %s -> %s/%s/%s/%s\n", agentID, m.Channel, m.PeerKind, m.PeerID, m.ThreadID)
	return nil
}

func RunUnbind(ctx context.Context, out io.Writer, match BindingMatch, asJSON bool, opts Options) error {
	reg, cleanup, err := openRegistry(opts)
	if err != nil {
		return err
	}
	defer cleanup()

	m := BindingMatchToGoncho(match)
	if err := reg.Unbind(ctx, m); err != nil {
		return fmt.Errorf("gormes agent unbind: %w", err)
	}
	if asJSON {
		removed := true
		return writeJSON(out, bindingReportJSON{Build: build(opts), Match: bindingMatchToJSON(m), Removed: &removed})
	}
	fmt.Fprintf(out, "unbound %s/%s/%s/%s\n", m.Channel, m.PeerKind, m.PeerID, m.ThreadID)
	return nil
}

func RunInspect(ctx context.Context, out, errOut io.Writer, match BindingMatch, asJSON bool, opts Options) error {
	reg, cleanup, err := openRegistry(opts)
	if err != nil {
		return err
	}
	defer cleanup()

	m := BindingMatchToGoncho(match)
	agentID, found, err := reg.Resolve(ctx, m)
	if err != nil {
		return fmt.Errorf("gormes agent inspect: %w", err)
	}
	if asJSON {
		bound := found
		report := bindingReportJSON{Build: build(opts), Match: bindingMatchToJSON(m), Bound: &bound}
		if found {
			report.AgentID = agentID
		}
		return writeJSON(out, report)
	}
	if !found {
		fmt.Fprintln(errOut, "agent_not_bound")
		return NewExitCodeError(1, fmt.Errorf("agent_not_bound"))
	}
	fmt.Fprintf(out, "agent: %s\n", agentID)
	return nil
}

func RunSpawn(ctx context.Context, out io.Writer, name string, spawn SpawnOptions, opts Options) error {
	reg, cleanup, err := openRegistry(opts)
	if err != nil {
		return err
	}
	defer cleanup()

	record, err := reg.Create(ctx, goncho.CreateAgentOptions{Name: name, Persona: spawn.Persona})
	if err != nil {
		if errors.Is(err, goncho.ErrAgentIDInvalid) {
			return NewExitCodeError(2, fmt.Errorf("agent_id_invalid: name %q does not normalize to a valid agent id (^[a-z][a-z0-9_-]{0,63}$)", name))
		}
		return fmt.Errorf("gormes agent spawn: %w", err)
	}
	if spawn.JSON {
		return writeJSON(out, spawnReportJSON{Build: build(opts), Agent: recordToJSON(record)})
	}
	fmt.Fprintf(out, "spawned agent %s (%s)\n", record.ID, record.Name)
	return nil
}

func RunList(ctx context.Context, out io.Writer, asJSON bool, opts Options) error {
	reg, cleanup, err := openRegistry(opts)
	if err != nil {
		return err
	}
	defer cleanup()

	records, err := reg.List(ctx)
	if err != nil {
		return fmt.Errorf("gormes agent list: %w", err)
	}
	if asJSON {
		report := listReportJSON{Build: build(opts), Agents: make([]recordJSON, 0, len(records))}
		for _, r := range records {
			report.Agents = append(report.Agents, recordToJSON(r))
		}
		return writeJSON(out, report)
	}
	if len(records) == 0 {
		fmt.Fprintln(out, "no runtime-spawned agents")
		return nil
	}
	for _, r := range records {
		fmt.Fprintf(out, "%s\t%s\n", r.ID, r.Name)
	}
	return nil
}

func RunReset(out io.Writer, opts ResetOptions, build BuildProvenance) error {
	result, err := agenttemplate.ApplyDefaultTemplates(agenttemplate.WriteOptions{TargetDir: opts.Target, Force: opts.Force, DryRun: opts.DryRun})
	if err != nil {
		return fmt.Errorf("gormes agent reset: %w", err)
	}
	if opts.JSON {
		report := resetReportJSON{Build: build, Target: result.TargetDir, DryRun: opts.DryRun, Files: make([]resetFileJSON, len(result.Files))}
		for i, f := range result.Files {
			report.Files[i] = resetFileJSON{Path: f.Path, Action: string(f.Action)}
		}
		return writeJSON(out, report)
	}
	fmt.Fprintf(out, "target: %s\n", result.TargetDir)
	for _, file := range result.Files {
		fmt.Fprintf(out, "%s %s\n", file.Action, file.Path)
	}
	return nil
}

func BindingMatchToGoncho(f BindingMatch) goncho.BindingMatch {
	return goncho.BindingMatch{Channel: f.Channel, PeerKind: f.PeerKind, PeerID: f.PeerID, ThreadID: f.ThreadID}
}

func bindingMatchToJSON(m goncho.BindingMatch) bindingMatchJSON {
	return bindingMatchJSON{Channel: m.Channel, PeerKind: m.PeerKind, PeerID: m.PeerID, ThreadID: m.ThreadID}
}

func recordToJSON(r goncho.AgentRecord) recordJSON {
	return recordJSON{ID: r.ID, Name: r.Name, Persona: r.Persona, CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339)}
}

func openRegistry(opts Options) (Registry, func(), error) {
	if opts.OpenRegistry == nil {
		return nil, func() {}, fmt.Errorf("gormes agent: registry opener is not configured")
	}
	reg, cleanup, err := opts.OpenRegistry()
	if cleanup == nil {
		cleanup = func() {}
	}
	return reg, cleanup, err
}

func build(opts Options) BuildProvenance {
	if opts.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return opts.BuildProvenance()
}

func writeJSON(out io.Writer, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(body))
	return err
}
