package dashboard

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver"
)

type CommandOptions struct {
	APIKey    string
	Version   string
	GitCommit string
	GitDirty  bool
	GoVersion string
	Stderr    io.Writer
	OpenURL   func(string) error
	// BuildTurnLoop, when set, constructs the native turn loop that backs
	// dashboard chat. It is invoked once at server start with the command
	// context; the returned cleanup runs at shutdown. When nil (or when it
	// returns an error), chat degrades to display-only task injection. The
	// factory lives in the CLI layer because building a kernel needs the
	// provider client and tool registry, which the app package must not import.
	BuildTurnLoop func(ctx context.Context) (apiserver.TurnLoop, func(), error)
}

type dashboardOptions struct {
	Port   int
	NoOpen bool
}

func NewCommand(options CommandOptions) *cobra.Command {
	opts := dashboardOptions{}
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Start the Gormes web dashboard",
		Long:  "Starts an HTTP server with an htmx-based web dashboard for managing sessions, config, skills, and logs.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCommand(cmd.Context(), opts, options)
		},
	}
	cmd.Flags().IntVar(&opts.Port, "port", 43827, "Dashboard HTTP server port")
	cmd.Flags().BoolVar(&opts.NoOpen, "no-open", false, "do not open the dashboard in a browser")
	return cmd
}

func DefaultCommandOptions(version string, gitCommit string, gitDirty bool) CommandOptions {
	return CommandOptions{
		APIKey:    os.Getenv("GORMES_DASHBOARD_API_KEY"),
		Version:   version,
		GitCommit: gitCommit,
		GitDirty:  gitDirty,
		GoVersion: runtime.Version(),
		Stderr:    os.Stderr,
		OpenURL:   OpenURL,
	}
}

func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func runCommand(ctx context.Context, opts dashboardOptions, options CommandOptions) error {
	apiKey := options.APIKey
	if apiKey == "" {
		apiKey = "gormes-dashboard-dev"
	}
	cfg := apiserver.Config{
		APIKey:             apiKey,
		DashboardBoundHost: "127.0.0.1",
		BuildInfo: apiserver.BuildInfo{
			Version:   options.Version,
			GitCommit: options.GitCommit,
			GitDirty:  options.GitDirty,
			GoVersion: options.GoVersion,
		},
		ConfigSummary: buildConfigSummary,
		EnvStatus:     buildEnvStatus,
		SkillsList:    buildSkillsList,
	}

	stderr := options.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	// Wire the native turn loop FIRST so dashboard chat runs real agent turns
	// and the transcript store (memory.db) exists before the session directory
	// is opened below. A build failure (e.g. missing provider credentials)
	// degrades to display-only chat rather than blocking dashboard startup.
	if options.BuildTurnLoop != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		loop, cleanup, err := options.BuildTurnLoop(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "Dashboard chat disabled (turn loop unavailable): %v\n", err)
		} else {
			cfg.Loop = loop
			if cleanup != nil {
				defer cleanup()
			}
		}
	}

	// Live persistent sessions (memory.db, concurrent-safe). Opened after the
	// turn loop so a dashboard-created transcript DB is visible immediately.
	if lister, closer, ok := newSessionsLister(); ok {
		cfg.SessionsList = lister
		defer closer()
	}
	// Live cron jobs (sessions.db / bbolt; best-effort — skipped when the
	// gateway holds the exclusive lock).
	if reader, closer, ok := newCronReader(); ok {
		cfg.CronJobs = reader
		defer closer()
	} else {
		fmt.Fprintln(stderr, "Dashboard cron panel disabled (cron store unavailable; gateway may hold sessions.db)")
	}

	srv, err := apiserver.NewServerChecked(cfg)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", opts.Port)
	url := fmt.Sprintf("http://%s/dashboard", addr)
	fmt.Fprintf(stderr, "Gormes dashboard starting at %s\n", url)
	if !opts.NoOpen {
		openURL := options.OpenURL
		go func() {
			time.Sleep(250 * time.Millisecond)
			if openURL == nil {
				fmt.Fprintf(stderr, "Open %s in your browser (no opener configured)\n", url)
				return
			}
			if err := openURL(url); err != nil {
				fmt.Fprintf(stderr, "Open %s in your browser (%v)\n", url, err)
			}
		}()
	}

	server := &http.Server{Addr: addr, Handler: srv.Handler()}
	return server.ListenAndServe()
}
