package main

import "slices"

type gonchoCLIParityStatus string

const (
	gonchoCLIImplemented gonchoCLIParityStatus = "implemented"
	gonchoCLIRowBacked   gonchoCLIParityStatus = "row_backed"
)

type gonchoCLIParityKind string

const (
	gonchoCLICommand    gonchoCLIParityKind = "command"
	gonchoCLICommandSet gonchoCLIParityKind = "command_set"
	gonchoCLIGlobalFlag gonchoCLIParityKind = "global_flag"
)

type gonchoCLIParityEntry struct {
	Path       []string              `json:"path"`
	Kind       gonchoCLIParityKind   `json:"kind"`
	Status     gonchoCLIParityStatus `json:"status"`
	SourceRef  string                `json:"source_ref"`
	Target     string                `json:"target,omitempty"`
	Residual   string                `json:"residual,omitempty"`
	TestRef    string                `json:"test_ref,omitempty"`
	ExitCodes  bool                  `json:"exit_codes,omitempty"`
	JSONOutput bool                  `json:"json_output,omitempty"`
	Confirm    bool                  `json:"confirm,omitempty"`
}

func gonchoCLICompatibilityManifest() []gonchoCLIParityEntry {
	entries := []gonchoCLIParityEntry{
		cliGlobalFlag("--json", "global JSON mode is row-backed outside goncho doctor; Honcho applies it across every command"),
		cliGlobalFlag("--version", "Honcho --version banner/version callback is row-backed; Gormes has top-level version output but no goncho-scoped version compatibility"),
		cliGlobalFlag("-V", "short version alias is row-backed with the Honcho banner/version callback"),
		cliScopeFlag("--workspace", "workspace override validation and command/group/top-level resolution remain row-backed"),
		cliScopeFlag("-w", "short workspace override alias remains row-backed"),
		cliScopeFlag("--peer", "peer override validation and command/group/top-level resolution remain row-backed"),
		cliScopeFlag("-p", "short peer override alias remains row-backed"),
		cliScopeFlag("--session", "session override validation and command/group/top-level resolution remain row-backed"),
		cliScopeFlag("-s", "short session override alias remains row-backed"),
		cliCommand([]string{"init"}, "honcho-cli/src/honcho_cli/commands/setup.py:init", gonchoCLIRowBacked, "", "config preservation, connection test, JSON output, env handling, and prompts remain row-backed"),
		{
			Path:       []string{"doctor"},
			Kind:       gonchoCLICommand,
			Status:     gonchoCLIImplemented,
			SourceRef:  "honcho-cli/src/honcho_cli/commands/setup.py:doctor",
			Target:     "cmd/gormes goncho doctor",
			Residual:   "Honcho CLI connection checks and exact JSON row names remain row-backed; Gormes doctor currently proves local Goncho topology and degraded modes",
			TestRef:    "cmd/gormes/goncho_doctor_test.go",
			ExitCodes:  true,
			JSONOutput: true,
		},
		cliCommand([]string{"help"}, "honcho-cli/src/honcho_cli/main.py:help_cmd", gonchoCLIRowBacked, "", "hidden help alias parity remains row-backed"),
		cliCommandSet("config", "honcho-cli/src/honcho_cli/commands/config_cmd.py", "config JSON/table output and stored client settings remain row-backed"),
	}

	for _, group := range []struct {
		name     string
		source   string
		residual string
		commands []string
	}{
		{
			name:     "workspace",
			source:   "honcho-cli/src/honcho_cli/commands/workspace.py",
			residual: "workspace CRUD/search/queue-status command execution, JSON arrays, destructive confirms, and error codes remain row-backed",
			commands: []string{
				"list", "create", "inspect", "delete", "search", "queue-status",
			},
		},
		{
			name:     "peer",
			source:   "honcho-cli/src/honcho_cli/commands/peer.py",
			residual: "peer list/inspect/card/chat/search/create/metadata/representation execution and JSON/error contracts remain row-backed",
			commands: []string{
				"list", "inspect", "card", "chat", "search", "create", "get-metadata", "set-metadata", "representation",
			},
		},
		{
			name:     "session",
			source:   "honcho-cli/src/honcho_cli/commands/session.py",
			residual: "session list/create/inspect/context/summaries/delete/peers/search/representation/metadata execution and confirmation guards remain row-backed",
			commands: []string{
				"list", "create", "inspect", "context", "summaries", "delete", "peers", "add-peers", "remove-peers", "search", "representation", "get-metadata", "set-metadata",
			},
		},
		{
			name:     "message",
			source:   "honcho-cli/src/honcho_cli/commands/message.py",
			residual: "message list/create/get command execution and single-object JSON output remain row-backed",
			commands: []string{"list", "create", "get"},
		},
		{
			name:     "conclusion",
			source:   "honcho-cli/src/honcho_cli/commands/conclusion.py",
			residual: "conclusion list/search/create/delete execution, observer validation, and delete confirmation remain row-backed",
			commands: []string{"list", "search", "create", "delete"},
		},
	} {
		entries = append(entries, cliCommandSet(group.name, group.source, group.residual))
		for _, command := range group.commands {
			entry := cliCommand([]string{group.name, command}, group.source+":"+command, gonchoCLIRowBacked, "", group.residual)
			switch group.name + " " + command {
			case "workspace delete", "session delete", "conclusion delete":
				entry.Confirm = true
				entry.ExitCodes = true
			case "workspace list", "workspace search", "message get":
				entry.JSONOutput = true
			}
			entries = append(entries, entry)
		}
	}

	return cloneGonchoCLIParityEntries(entries)
}

func cliGlobalFlag(flag, residual string) gonchoCLIParityEntry {
	return gonchoCLIParityEntry{
		Path:      []string{flag},
		Kind:      gonchoCLIGlobalFlag,
		Status:    gonchoCLIRowBacked,
		SourceRef: "honcho-cli/src/honcho_cli/main.py",
		Residual:  residual,
	}
}

func cliScopeFlag(flag, residual string) gonchoCLIParityEntry {
	return gonchoCLIParityEntry{
		Path:      []string{flag},
		Kind:      gonchoCLIGlobalFlag,
		Status:    gonchoCLIRowBacked,
		SourceRef: "honcho-cli/src/honcho_cli/common.py",
		Residual:  residual,
	}
}

func cliCommandSet(name, sourceRef, residual string) gonchoCLIParityEntry {
	return gonchoCLIParityEntry{
		Path:      []string{name},
		Kind:      gonchoCLICommandSet,
		Status:    gonchoCLIRowBacked,
		SourceRef: sourceRef,
		Residual:  residual,
	}
}

func cliCommand(path []string, sourceRef string, status gonchoCLIParityStatus, target, residual string) gonchoCLIParityEntry {
	return gonchoCLIParityEntry{
		Path:      slices.Clone(path),
		Kind:      gonchoCLICommand,
		Status:    status,
		SourceRef: sourceRef,
		Target:    target,
		Residual:  residual,
	}
}

func cloneGonchoCLIParityEntries(in []gonchoCLIParityEntry) []gonchoCLIParityEntry {
	out := make([]gonchoCLIParityEntry, len(in))
	for i, entry := range in {
		out[i] = entry
		out[i].Path = slices.Clone(entry.Path)
	}
	return out
}
