package main

import (
	"bytes"
	"strings"
	"testing"

	"ctmux/internal/cli"
	"ctmux/internal/config"
)

func TestEmbeddedConfigs(t *testing.T) {
	tiers := []string{"center", "hub", "node"}
	for _, tier := range tiers {
		t.Run(tier, func(t *testing.T) {
			data, err := config.ReadEmbeddedConfig(configFS, tier)
			if err != nil {
				t.Fatalf("failed to read embedded config for %s: %v", tier, err)
			}
			if len(data) == 0 {
				t.Fatalf("embedded config for %s is empty", tier)
			}
			// Verify it contains expected tmux directives
			if !strings.Contains(string(data), "new-session") {
				t.Errorf("embedded config %s does not contain 'new-session'", tier)
			}
		})
	}
}

func TestAppIntegrationHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := cli.NewApp(configFS, cli.BuildInfo{
		Version: "1.0.0",
		Commit:  "test-commit",
		Date:    "2026-11-26",
	})
	app.Stdout = &stdout
	app.Stderr = &stderr

	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("unexpected error running --help: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "center") || !strings.Contains(output, "hub") || !strings.Contains(output, "node") {
		t.Errorf("expected help output to mention center, hub, node; got:\n%s", output)
	}
}
