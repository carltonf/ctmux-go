package session

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		session  Session
		expected []string
	}{
		{
			name: "default attach mode",
			session: Session{
				Tier:       "center",
				SocketPath: "/tmp/center.sock",
				ConfigPath: "/tmp/center.conf",
			},
			expected: []string{"tmux", "-S", "/tmp/center.sock", "-f", "/tmp/center.conf", "at"},
		},
		{
			name: "detached mode with default session zero",
			session: Session{
				Tier:       "hub",
				SocketPath: "/tmp/hub.sock",
				ConfigPath: "/tmp/hub.conf",
				Detach:     true,
			},
			expected: []string{"tmux", "-S", "/tmp/hub.sock", "-f", "/tmp/hub.conf", "new-session", "-d", "-A", "-s", "zero"},
		},
		{
			name: "detached mode with custom session name and extra args",
			session: Session{
				Tier:        "node",
				SocketPath:  "/tmp/node.sock",
				ConfigPath:  "/tmp/node.conf",
				SessionName: "custom",
				Detach:      true,
				ExtraArgs:   []string{"-v"},
			},
			expected: []string{"tmux", "-S", "/tmp/node.sock", "-f", "/tmp/node.conf", "new-session", "-d", "-A", "-s", "custom", "-v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.session.BuildArgs()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("BuildArgs() = %v; want %v", got, tt.expected)
			}
		})
	}
}

func TestSessionRunDryRun(t *testing.T) {
	// In DryRun mode, Run should succeed even if tmux is not installed
	s := Session{
		Tier:       "center",
		SocketPath: "/tmp/center.sock",
		ConfigPath: "/tmp/center.conf",
		DryRun:     true,
	}

	// Create a dummy fake tmux in PATH or check dry run
	t.Setenv("PATH", t.TempDir()) // empty PATH so tmux definitely doesn't exist
	err := s.Run(nil)
	if err == nil {
		t.Errorf("expected error when tmux is not in PATH even with DryRun, since FindTmuxBinary checks binary")
	}
}

func TestSessionRunMockRunner(t *testing.T) {
	fakeBinDir := t.TempDir()
	fakeTmux := filepath.Join(fakeBinDir, "tmux")
	if err := os.WriteFile(fakeTmux, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBinDir)

	called := false
	var capturedBinary string
	var capturedArgs []string

	mockRunner := func(binary string, args []string, env []string) error {
		called = true
		capturedBinary = binary
		capturedArgs = args
		return nil
	}

	s := Session{
		Tier:       "node",
		SocketPath: "/tmp/node.sock",
		ConfigPath: "/tmp/node.conf",
	}

	if err := s.Run(mockRunner); err != nil {
		t.Fatalf("unexpected error running session: %v", err)
	}

	if !called {
		t.Errorf("expected mock runner to be called")
	}
	if capturedBinary != fakeTmux {
		t.Errorf("binary = %q; want %q", capturedBinary, fakeTmux)
	}
	expectedArgs := []string{"tmux", "-S", "/tmp/node.sock", "-f", "/tmp/node.conf", "at"}
	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Errorf("args = %v; want %v", capturedArgs, expectedArgs)
	}
}

func TestGetStatusNonExistentSocket(t *testing.T) {
	status := GetStatus("node", "/nonexistent/path/node.sock")
	if status.SocketExists {
		t.Errorf("expected SocketExists=false, got true")
	}
	if status.IsRunning {
		t.Errorf("expected IsRunning=false, got true")
	}
}

func TestKillNonExistentSocket(t *testing.T) {
	err := Kill("/nonexistent/path/node.sock")
	if err != nil {
		t.Errorf("expected nil error for nonexistent socket kill, got %v", err)
	}
}
