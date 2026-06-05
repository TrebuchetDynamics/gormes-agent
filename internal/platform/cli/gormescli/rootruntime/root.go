package rootruntime

import (
	"fmt"

	"github.com/spf13/cobra"
)

type RootOptions struct {
	Version           string
	PersistentPreRunE func(*cobra.Command, []string) error
	RunE              func(*cobra.Command, []string) error
	Finalizers        []func(*cobra.Command)
}

type CommandFactories map[string]func() *cobra.Command

var RootCommandOrder = []string{
	"doctor",
	"version",
	"telegram",
	"gateway",
	"channels",
	"whatsapp",
	"slack",
	"session",
	"memory",
	"goncho",
	"kanban",
	"chat",
	"send",
	"curator",
	"acp",
	"system",
	"agent",
	"navivox",
	"usage",
	"tts",
	"status",
	"auth",
	"providers",
	"logout",
	"config",
	"fallback",
	"router",
	"fidelity",
	"secrets",
	"security",
	"migrate",
	"claw",
	"profile",
	"model",
	"setup",
	"skills",
	"plugins",
	"mcp",
	"dashboard",
	"update",
	"restore",
	"uninstall",
	"logs",
	"checkpoints",
	"completion",
	"cron",
	"webhook",
	"hooks",
	"dump",
	"debug",
	"backup",
	"import",
	"pairing",
	"tools",
	"insights",
	"admin",
}

func NewRootCommand(opts RootOptions, factories CommandFactories) *cobra.Command {
	root := &cobra.Command{
		Use:     "gormes",
		Version: opts.Version,
		Short:   "Go-native Hermes-compatible agent runtime",
		Long: `Gormes is the Go-native Hermes-compatible agent runtime: one static binary for
terminal chat, scripted queries, local memory, skills, tools, and messaging
gateways. It does not require Python, Docker, or a separate Hermes process.

Fast paths:
  Fresh install:       gormes setup -> gormes chat
  Targeted setup:      gormes setup --quick --target terminal
  Scripted chat:       gormes chat -q "summarize this repo"
  Interactive TUI:     gormes --profile work
  Messaging gateway:   gormes gateway status -> gormes gateway -> gormes logs

Operator workflows:
  First run and setup
    gormes setup                      run guided setup
    gormes setup --quick              configure missing setup items only
    gormes setup provider             configure endpoint, model, and API key
    gormes setup model                pick the default provider/model
    gormes doctor --offline           check local readiness without network calls

  Provider/auth/debug
    gormes config show                show config with secrets redacted
    gormes config check --json        machine-check config and dotenv readiness
    gormes config edit                open config.toml in your editor
    gormes auth add <provider>        add provider credentials
    gormes providers setup <provider> show provider-specific setup commands
    gormes logout <provider>          clear stored provider auth
    gormes usage                      show provider account usage
    gormes router --dry-run           inspect local OpenAI-compatible Router config
    gormes debug share                collect and share a debug bundle

  Session and memory
    gormes                            open the TUI
    gormes --offline                  smoke test the TUI without provider calls or network submits
    gormes chat -q "hello"            send one chat query and exit
    gormes session list               list past sessions
    gormes session export <id>        export a session transcript
    gormes memory status              inspect memory store
    gormes goncho doctor --json       inspect Goncho memory storage

  Agents, profiles, and workspace
    gormes agent reset                seed default agent context templates
    gormes setup agent                print multi-agent setup guidance
    gormes setup workspace            print workspace setup guidance
    gormes setup bindings             print channel-to-agent binding guidance
    gormes profile list               list known profiles
    gormes profile use <name>         switch active profile
    gormes kanban list                inspect durable multi-agent work

  Automation and integrations
    gormes dashboard                  start http://127.0.0.1:43827/dashboard
    gormes gateway                    start the configured gateway
    gormes gateway status             check gateway runtime state
    gormes gateway reload             reload swappable live gateway config
    gormes gateway stop               stop a running gateway
    gormes send --to telegram "done"  send or dry-run platform delivery
    gormes whatsapp                   set up WhatsApp Baileys pairing
    gormes telegram                   start Telegram-only mode
    gormes acp client                 connect a debug ACP client to Gormes
    gormes system event "note"        enqueue a system event and heartbeat wake
    gormes mcp login <server>         refresh OAuth for one MCP server

  Tools, skills, and maintenance
    gormes skills list                list installed skills
    gormes skills install <url>       install a direct SKILL.md URL
    gormes curator status             inspect background skill maintenance
    gormes plugins list               list installed plugins
    gormes secrets audit --plan file  audit SecretRef runtime plans
    gormes security audit --deep      inspect gateway, channel, tool, and state security
    gormes status                     show runtime and progress blockers
    gormes migrate hermes             import state from Hermes (dry-run)
    gormes update                     update a managed source checkout
    gormes version                    print version
    gormes uninstall                  remove Gormes artifacts

Examples:
  gormes --offline --profile test
  gormes chat -q "write a release note"
  gormes chat
  gormes gateway status --json
  gormes config check --json
  gormes session export <id> --format json

Config and state:
  home:     $GORMES_HOME (default ~/.gormes)
  config:   ~/.gormes/config.toml
  secrets:  ~/.gormes/.env
  profiles: gormes profile list

Environment:
  GORMES_HOME                       runtime home (default ~/.gormes)
  GORMES_API_KEY                    provider API key
  GORMES_ENDPOINT                   provider endpoint URL
  GORMES_INFERENCE_MODEL            default model override
  GORMES_INFERENCE_PROVIDER         default provider override
  GORMES_SKILLS_ROOT                custom skills directory

Need more detail:
  gormes help <command>
  gormes <command> --help
  gormes completion <shell>
  docs: https://docs.gormes.ai`,
		SilenceUsage:      true,
		PersistentPreRunE: opts.PersistentPreRunE,
		RunE:              opts.RunE,
	}
	root.Flags().BoolP("version", "V", false, "version for gormes")
	root.PersistentFlags().StringP("profile", "p", "", "profile name for this invocation")
	root.PersistentFlags().StringArray("skills", nil, "runtime skill allowlist for this invocation; repeat or comma-separate")
	root.PersistentFlags().StringP("model", "m", "", "model override for chat or TUI startup; also settable via GORMES_INFERENCE_MODEL")
	root.PersistentFlags().String("provider", "", "provider override for chat or TUI startup; also settable via GORMES_INFERENCE_PROVIDER")
	root.PersistentFlags().String("endpoint", "", "provider endpoint override for chat or TUI startup; invocation-only and also settable via GORMES_ENDPOINT")
	root.PersistentFlags().String("api-key", "", "provider API key override for chat or TUI startup; invocation-only and never persisted")
	root.PersistentFlags().Bool("offline", false, "run the TUI as a local smoke test without provider health checks or network submits")
	if flag := root.PersistentFlags().Lookup("offline"); flag != nil && root.Flags().Lookup("offline") == nil {
		root.Flags().AddFlag(flag)
	}
	root.Flags().String("resume", "", "override persisted session_id for the TUI's default key")
	root.Flags().StringP("continue", "c", "", "resolve a session id or unique prefix and resume it")
	if flag := root.Flags().Lookup("continue"); flag != nil {
		flag.NoOptDefVal = "last"
	}
	root.Flags().String("remote", "", "connect the TUI to a remote Gormes gateway over SSE (consumes /events; bypasses local kernel and provider setup)")

	for _, name := range RootCommandOrder {
		factory := factories[name]
		if factory == nil {
			panic(fmt.Sprintf("gormes root command factory %q is not configured", name))
		}
		root.AddCommand(factory())
	}
	for _, finalize := range opts.Finalizers {
		if finalize != nil {
			finalize(root)
		}
	}
	return root
}
