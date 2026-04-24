package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/www.gormes.ai/internal/site"
)

func main() {
	out := flag.String("out", "dist", "output directory")
	flag.Parse()

	if err := site.ExportDir(*out); err != nil {
		slog.Error("export site", "out", *out, "err", err)
		os.Exit(1)
	}

	slog.Info("exported www.gormes.ai", "out", *out)
}
