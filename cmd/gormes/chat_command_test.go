package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestChatWelcomeStartupSeedUsesRealVersionAndToolCount(t *testing.T) {
	if Version == "" {
		t.Fatal("main.Version must be non-empty for the welcome panel seed")
	}

	v, n, toolsets := welcomeStartupSeed(tools.NewRegistry())
	if v != Version {
		t.Fatalf("welcomeStartupSeed version = %q, want main.Version %q", v, Version)
	}
	if n != 0 {
		t.Fatalf("empty registry tool count = %d, want 0", n)
	}
	if len(toolsets) != 0 {
		t.Fatalf("empty registry toolsets = %v, want none", toolsets)
	}

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "terminal",
		ExecuteFn: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		},
	})
	if _, n2, toolsets2 := welcomeStartupSeed(reg); n2 != 1 {
		t.Fatalf("one-tool registry count = %d, want 1", n2)
	} else if !slices.Contains(toolsets2, "terminal") {
		t.Fatalf("one-tool registry toolsets = %v, want terminal", toolsets2)
	}
}

func TestChatCommandProfileFlagSetsGormesHomeBeforeConfigLoad(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	initialHome := config.GormesHome()
	wantProfileHome := filepath.Join(filepath.Dir(initialHome), "gormes", "profiles", "worker")
	hermesHome := os.Getenv("HERMES_HOME")

	var gotHome string
	var gotPrompt string
	cmd := newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			gotHome = os.Getenv("GORMES_HOME")
			gotPrompt = invocation.Prompt
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			t.Fatal("runResolvedTUI was called for chat -q")
			return nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "-p", "worker", "chat", "-q", "work kanban task t_123")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotHome != wantProfileHome {
		t.Fatalf("GORMES_HOME during chat = %q, want profile root %q", gotHome, wantProfileHome)
	}
	if os.Getenv("HERMES_HOME") != hermesHome {
		t.Fatalf("HERMES_HOME changed to %q, want %q", os.Getenv("HERMES_HOME"), hermesHome)
	}
	if gotPrompt != "work kanban task t_123" {
		t.Fatalf("Prompt = %q, want kanban task prompt", gotPrompt)
	}
}

func TestChatCommandPinsCurrentKanbanBoardDBForTools(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("HERMES_KANBAN_BOARD", "")
	t.Setenv("HERMES_KANBAN_DB", "")

	reg := kanban.NewBoardRegistry(config.KanbanHome())
	if err := reg.Create("alpha"); err != nil {
		t.Fatalf("create alpha board: %v", err)
	}
	if err := reg.Switch("alpha"); err != nil {
		t.Fatalf("switch alpha board: %v", err)
	}
	wantDB := reg.BoardPath("alpha")
	hermesHome := os.Getenv("HERMES_HOME")

	var gotDB string
	var gotHermesBoard string
	cmd := newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(_ *cobra.Command, _ oneshotInvocation) error {
			gotDB = os.Getenv("GORMES_KANBAN_DB")
			gotHermesBoard = os.Getenv("HERMES_KANBAN_BOARD")
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			t.Fatal("runResolvedTUI was called for chat -q")
			return nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "chat", "-q", "work kanban task t_123")
	if err != nil {
		t.Fatalf("chat -q error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotDB != wantDB {
		t.Fatalf("GORMES_KANBAN_DB during chat = %q, want current board DB %q", gotDB, wantDB)
	}
	if gotHermesBoard != "" {
		t.Fatalf("HERMES_KANBAN_BOARD changed to %q, want untouched", gotHermesBoard)
	}
	if os.Getenv("HERMES_HOME") != hermesHome {
		t.Fatalf("HERMES_HOME changed to %q, want %q", os.Getenv("HERMES_HOME"), hermesHome)
	}
	if got := os.Getenv("GORMES_KANBAN_DB"); got != "" {
		t.Fatalf("GORMES_KANBAN_DB after chat = %q, want restored empty", got)
	}
}

func TestChatCommandPreservesExplicitKanbanDBPin(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	explicitDB := filepath.Join(t.TempDir(), "explicit-kanban.db")
	t.Setenv("GORMES_KANBAN_DB", explicitDB)

	reg := kanban.NewBoardRegistry(config.KanbanHome())
	if err := reg.Create("alpha"); err != nil {
		t.Fatalf("create alpha board: %v", err)
	}
	if err := reg.Switch("alpha"); err != nil {
		t.Fatalf("switch alpha board: %v", err)
	}

	var gotDB string
	cmd := newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(_ *cobra.Command, _ oneshotInvocation) error {
			gotDB = os.Getenv("GORMES_KANBAN_DB")
			return nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "chat", "-q", "work kanban task t_456")
	if err != nil {
		t.Fatalf("chat -q error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotDB != explicitDB {
		t.Fatalf("GORMES_KANBAN_DB during chat = %q, want explicit pin %q", gotDB, explicitDB)
	}
	if got := os.Getenv("GORMES_KANBAN_DB"); got != explicitDB {
		t.Fatalf("GORMES_KANBAN_DB after chat = %q, want explicit pin preserved %q", got, explicitDB)
	}
}

func TestChatCommandQueryDispatchesOneshot(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	var gotPrompt string
	cmd := newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			gotPrompt = invocation.Prompt
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			t.Fatal("runResolvedTUI was called for chat query")
			return nil
		},
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "chat", "-q", "hello from query")
	if err != nil {
		t.Fatalf("chat -q error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotPrompt != "hello from query" {
		t.Fatalf("chat -q prompt = %q", gotPrompt)
	}

	gotPrompt = ""
	cmd = newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			gotPrompt = invocation.Prompt
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			t.Fatal("runResolvedTUI was called for chat args")
			return nil
		},
	})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "chat", "hello", "from", "args")
	if err != nil {
		t.Fatalf("chat args error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotPrompt != "hello from args" {
		t.Fatalf("chat args prompt = %q", gotPrompt)
	}

	rootHelp, _, err := executeOneshotFlagCommand(newRootCommandWithRuntime(rootRuntime{}), "--help")
	if err != nil {
		t.Fatalf("root help error = %v", err)
	}
	for _, want := range []string{"--profile", "--skills", "chat"} {
		if !strings.Contains(rootHelp, want) {
			t.Fatalf("root help missing %q:\n%s", want, rootHelp)
		}
	}

	chatHelp, _, err := executeOneshotFlagCommand(newRootCommandWithRuntime(rootRuntime{}), "chat", "--help")
	if err != nil {
		t.Fatalf("chat help error = %v", err)
	}
	if !strings.Contains(chatHelp, "-q") || !strings.Contains(chatHelp, "--query") {
		t.Fatalf("chat help missing query flag:\n%s", chatHelp)
	}
}

func TestChatCommandSkillsFlagBuildsForcedSkillProvider(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	skillsRoot := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", skillsRoot)
	writeChatCommandSkill(t, skillsRoot, "kanban-worker")
	writeChatCommandSkill(t, skillsRoot, "review-tests")
	writeChatCommandSkill(t, skillsRoot, "ops")

	var got []string
	cmd := newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(_ *cobra.Command, invocation oneshotInvocation) error {
			got = append([]string(nil), invocation.ForcedSkills...)
			return nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(
		cmd,
		"--skills", "kanban-worker,review-tests",
		"--skills", "kanban-worker",
		"--skills", "ops",
		"chat", "-q", "work kanban task t_456",
	)
	if err != nil {
		t.Fatalf("chat --skills error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	want := []string{"kanban-worker", "review-tests", "ops"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ForcedSkills = %#v, want %#v", got, want)
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("chat --skills mutated config path %s: %v", config.ConfigPath(), err)
	}
}

func TestChatCommandSkillsFlagRejectsMissingSkillBeforeOneshot(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_SKILLS_ROOT", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(*cobra.Command, oneshotInvocation) error {
			t.Fatal("runOneshot was called for missing forced skill")
			return nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--skills", "missing-skill", "chat", "-q", "work")
	if err == nil {
		t.Fatalf("chat --skills missing-skill succeeded\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "skill_unavailable: missing-skill") {
		t.Fatalf("error = %v, want missing skill evidence", err)
	}
}

func TestChatCommandWithoutQueryDelegatesTUI(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	var tuiCalled bool
	cmd := newRootCommandWithRuntime(rootRuntime{
		runOneshot: func(*cobra.Command, oneshotInvocation) error {
			t.Fatal("runOneshot was called for bare chat")
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			tuiCalled = true
			return nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "chat")
	if err != nil {
		t.Fatalf("bare chat error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !tuiCalled {
		t.Fatal("runResolvedTUI was not called")
	}
}

func writeChatCommandSkill(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "active", name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create skill dir: %v", err)
	}
	body := "---\nname: " + name + "\ndescription: " + name + " fixture\n---\nUse " + name + ".\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write skill fixture: %v", err)
	}
}
