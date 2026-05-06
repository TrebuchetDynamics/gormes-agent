package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

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
