package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/goncho"
)

func newGonchoMemoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Search and inspect Goncho memories",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newGonchoMemorySearchCommand())
	cmd.AddCommand(newGonchoMemoryInspectCommand())
	return cmd
}

func newGonchoMemorySearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Goncho memories by keyword",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runGonchoMemorySearch,
	}
	cmd.Flags().String("peer", "operator", "peer ID for search scope")
	cmd.Flags().String("session", "", "optional session key to scope search")
	cmd.Flags().Int("limit", 10, "max results to return")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}

func newGonchoMemoryInspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a single memory entry",
		Args:  cobra.ExactArgs(1),
		RunE:  runGonchoMemoryInspect,
	}
	cmd.Flags().String("peer", "operator", "peer ID for search scope")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}

func runGonchoMemorySearch(cmd *cobra.Command, args []string) error {
	emitJSON, _ := cmd.Flags().GetBool("json")
	peer, _ := cmd.Flags().GetString("peer")
	sessionKey, _ := cmd.Flags().GetString("session")
	limit, _ := cmd.Flags().GetInt("limit")

	query := strings.Join(args, " ")

	db, svc, err := openGonchoService()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	result, err := svc.Search(ctx, goncho.SearchParams{
		Peer:       peer,
		Query:      query,
		SessionKey: sessionKey,
		Limit:      limit,
	})
	if err != nil {
		return fmt.Errorf("goncho memory search: %w", err)
	}

	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(result)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Goncho memory search: %q\n", query)
	fmt.Fprintf(cmd.OutOrStdout(), "results: %d\n\n", len(result.Results))
	for i, hit := range result.Results {
		fmt.Fprintf(cmd.OutOrStdout(), "%d. [%s] session=%s\n   %s\n\n",
			i+1, hit.Source, valueOrNone(hit.SessionKey), truncateString(hit.Content, 200))
	}
	return nil
}

func runGonchoMemoryInspect(cmd *cobra.Command, args []string) error {
	emitJSON, _ := cmd.Flags().GetBool("json")
	peer, _ := cmd.Flags().GetString("peer")

	db, svc, err := openGonchoService()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	result, err := svc.Search(ctx, goncho.SearchParams{
		Peer:  peer,
		Query: args[0],
		Limit: 1,
	})
	if err != nil {
		return fmt.Errorf("goncho memory inspect: %w", err)
	}
	if len(result.Results) == 0 {
		return fmt.Errorf("no memory found matching %q", args[0])
	}

	hit := result.Results[0]
	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		return enc.Encode(hit)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Goncho memory inspection\n")
	fmt.Fprintf(cmd.OutOrStdout(), "source: %s\n", hit.Source)
	fmt.Fprintf(cmd.OutOrStdout(), "session_key: %s\n", valueOrNone(hit.SessionKey))
	fmt.Fprintf(cmd.OutOrStdout(), "content:\n%s\n", hit.Content)
	return nil
}

func openGonchoService() (*sql.DB, *goncho.Service, error) {
	memoryPath := config.MemoryDBPath()
	if _, err := os.Stat(memoryPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("memory database not found at %s — run `gormes setup` first", memoryPath)
		}
		return nil, nil, err
	}

	db, err := sqlOpenGoncho(memoryPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open memory db: %w", err)
	}
	db.SetMaxOpenConns(1)

	cfg, err := config.Load(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	gonchoCfg := cfg.Goncho.RuntimeConfig()
	svc := goncho.NewService(db, gonchoCfg, nil)
	return db, svc, nil
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
