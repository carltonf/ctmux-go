package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"ctmux/internal/cli"
)

// Embed tmux configuration files from the tmux-confs directory.
//
//go:embed tmux-confs/*.conf
var embeddedFS embed.FS

// getConfigFS returns an fs.FS rooted at the tmux-confs directory.
func getConfigFS() (fs.FS, error) {
	sub, err := fs.Sub(embeddedFS, "tmux-confs")
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded tmux-confs filesystem: %w", err)
	}
	return sub, nil
}

// Build metadata injected at compile time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	configFS, err := getConfigFS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	app := cli.NewApp(configFS, cli.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})

	if err := app.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
