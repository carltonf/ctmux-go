package cli

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"
)

func newTestApp() (*App, *bytes.Buffer, *bytes.Buffer) {
	mockFS := fstest.MapFS{
		"center.conf": &fstest.MapFile{Data: []byte("# center conf\n")},
		"hub.conf":    &fstest.MapFile{Data: []byte("# hub conf\n")},
		"node.conf":   &fstest.MapFile{Data: []byte("# node conf\n")},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	mockRunner := func(binary string, args []string, env []string) error {
		return nil
	}

	app := &App{
		ConfigFS: mockFS,
		BuildInfo: BuildInfo{
			Version: "1.0.0-test",
			Commit:  "abc1234",
			Date:    "2026-11-26",
		},
		Stdout: stdout,
		Stderr: stderr,
		Runner: mockRunner,
	}

	return app, stdout, stderr
}

func TestAppNoArgs(t *testing.T) {
	app, _, stderr := newTestApp()
	err := app.Run([]string{})
	if err == nil {
		t.Errorf("expected error when no args provided")
	}
	if !strings.Contains(stderr.String(), "No arguments provided") {
		t.Errorf("expected error message in stderr, got: %s", stderr.String())
	}
}

func TestAppUnknownCommand(t *testing.T) {
	app, _, stderr := newTestApp()
	err := app.Run([]string{"unknown_cmd"})
	if err == nil {
		t.Errorf("expected error for unknown command")
	}
	if !strings.Contains(stderr.String(), "Invalid argument") {
		t.Errorf("expected invalid argument error in stderr, got: %s", stderr.String())
	}
}

func TestAppHelp(t *testing.T) {
	for _, flag := range []string{"help", "-h", "--help"} {
		app, stdout, _ := newTestApp()
		err := app.Run([]string{flag})
		if err != nil {
			t.Errorf("unexpected error on %s: %v", flag, err)
		}
		if !strings.Contains(stdout.String(), "ctmux - Layered tmux session manager") {
			t.Errorf("expected help text on %s, got: %s", flag, stdout.String())
		}
	}
}

func TestAppVersion(t *testing.T) {
	for _, flag := range []string{"version", "-v", "--version"} {
		app, stdout, _ := newTestApp()
		err := app.Run([]string{flag})
		if err != nil {
			t.Errorf("unexpected error on %s: %v", flag, err)
		}
		if !strings.Contains(stdout.String(), "1.0.0-test") {
			t.Errorf("expected version output on %s, got: %s", flag, stdout.String())
		}
	}
}

func TestAppConfigShow(t *testing.T) {
	app, stdout, _ := newTestApp()
	err := app.Run([]string{"config", "center"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != "# center conf\n" {
		t.Errorf("expected config output, got: %q", stdout.String())
	}

	// invalid tier
	app2, _, stderr2 := newTestApp()
	err2 := app2.Run([]string{"config", "invalid"})
	if err2 == nil {
		t.Errorf("expected error for invalid tier in config")
	}
	if !strings.Contains(stderr2.String(), "Unknown tier") {
		t.Errorf("expected Unknown tier error, got: %s", stderr2.String())
	}
}

func TestAppSessionDryRun(t *testing.T) {
	tempCache := t.TempDir()
	app, stdout, _ := newTestApp()
	err := app.Run([]string{"center", "-dry-run", "-cache-dir", tempCache})
	if err != nil {
		t.Fatalf("unexpected error on dry run: %v", err)
	}
	if !strings.Contains(stdout.String(), "Dry run: tmux -S") {
		t.Errorf("expected dry run message, got: %s", stdout.String())
	}
}

func TestAppReset(t *testing.T) {
	tempCache := t.TempDir()
	app, stdout, _ := newTestApp()
	err := app.Run([]string{"reset", "-cache-dir", tempCache})
	if err != nil {
		t.Fatalf("unexpected error on reset: %v", err)
	}
	if !strings.Contains(stdout.String(), "Cache is already clean") {
		t.Errorf("expected clean cache message, got: %s", stdout.String())
	}
}

func TestAppStatus(t *testing.T) {
	tempCache := t.TempDir()
	app, stdout, _ := newTestApp()
	err := app.Run([]string{"status", "-cache-dir", tempCache})
	if err != nil {
		t.Fatalf("unexpected error on status: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "TIER") || !strings.Contains(out, "center") {
		t.Errorf("expected status table, got: %s", out)
	}
}

func TestAppKill(t *testing.T) {
	tempCache := t.TempDir()
	app, stdout, _ := newTestApp()
	err := app.Run([]string{"kill", "all", "-cache-dir", tempCache})
	if err != nil {
		t.Fatalf("unexpected error on kill: %v", err)
	}
	if !strings.Contains(stdout.String(), "Terminated center") {
		t.Errorf("expected termination message, got: %s", stdout.String())
	}
}
