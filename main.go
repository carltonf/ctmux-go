package main

import (
	"embed"
	"os"

	"ctmux/internal/cli"
)

// Embed tmux configuration files directly into the ctmux binary.
//
//go:embed center.conf hub.conf node.conf
var configFS embed.FS

// Build metadata injected at compile time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := cli.NewApp(configFS, cli.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})

	if err := app.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
