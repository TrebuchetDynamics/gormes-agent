package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/config"
	"github.com/XelHaku/golang-hermes-agent/gormes/internal/hermes"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify that the Python api_server is reachable",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load(nil)
		if err != nil {
			return err
		}
		c := hermes.NewHTTPClient(cfg.Hermes.Endpoint, cfg.Hermes.APIKey)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := c.Health(ctx); err != nil {
			fmt.Fprintf(os.Stderr,
				"✗ api_server NOT reachable at %s: %v\n\nStart it with:\n  API_SERVER_ENABLED=true hermes gateway start\n",
				cfg.Hermes.Endpoint, err)
			os.Exit(1)
		}
		fmt.Printf("✓ api_server reachable at %s\n", cfg.Hermes.Endpoint)
		return nil
	},
}
