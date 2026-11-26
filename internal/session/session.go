package session

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Runner represents a function that executes the tmux process.
type Runner func(binary string, args []string, env []string) error

// Session represents a configured ctmux session ready for execution.
type Session struct {
	Tier        string
	CacheDir    string
	ConfigPath  string
	SocketPath  string
	SessionName string // Default: "zero"
	Detach      bool   // Start in background without attaching
	DryRun      bool   // Print command only
	ExtraArgs   []string
}

// SessionStatus reports the current state of a tier's tmux server.
type SessionStatus struct {
	Tier         string
	SocketPath   string
	SocketExists bool
	IsRunning    bool
	SessionName  string
	Windows      int
	Attached     bool
	RawInfo      string
}

// FindTmuxBinary locates the tmux executable in PATH.
func FindTmuxBinary() (string, error) {
	binary, err := exec.LookPath("tmux")
	if err != nil {
		return "", fmt.Errorf("tmux executable not found in PATH; please ensure tmux is installed and available in PATH")
	}
	return binary, nil
}

// BuildArgs constructs the command-line arguments for tmux.
func (s *Session) BuildArgs() []string {
	sessionName := s.SessionName
	if sessionName == "" {
		sessionName = "zero"
	}

	args := []string{"tmux", "-S", s.SocketPath, "-f", s.ConfigPath}

	if s.Detach {
		args = append(args, "new-session", "-d", "-A", "-s", sessionName)
	} else {
		// "at" is short for "attach-session"
		// If session does not exist, the 'new-session -A -s zero' in config will create it.
		args = append(args, "at")
	}

	if len(s.ExtraArgs) > 0 {
		args = append(args, s.ExtraArgs...)
	}

	return args
}

// DefaultRunner is the platform-aware process executor.
var DefaultRunner = func(binary string, args []string, env []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command(binary, args[1:]...)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return syscall.Exec(binary, args, env)
}

// Run launches the tmux session using the provided runner.
func (s *Session) Run(runner Runner) error {
	if runner == nil {
		runner = DefaultRunner
	}

	binary, err := FindTmuxBinary()
	if err != nil {
		return err
	}

	args := s.BuildArgs()

	if s.DryRun {
		return nil
	}

	if s.Detach {
		// When detaching, run via exec.Command even on Unix so ctmux can return cleanly.
		cmd := exec.Command(binary, args[1:]...)
		cmd.Env = os.Environ()
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to start detached tmux session: %w\nOutput: %s", err, string(output))
		}
		return nil
	}

	if err := runner(binary, args, os.Environ()); err != nil {
		return fmt.Errorf("failed to execute tmux (%s): %w", binary, err)
	}

	return nil
}

// GetStatus checks the running state of a tmux session for the given socket.
func GetStatus(tier, socketPath string) SessionStatus {
	status := SessionStatus{
		Tier:       tier,
		SocketPath: socketPath,
	}

	if fi, err := os.Stat(socketPath); err != nil || fi.IsDir() {
		status.SocketExists = false
		status.IsRunning = false
		return status
	}
	status.SocketExists = true

	binary, err := FindTmuxBinary()
	if err != nil {
		// Tmux binary not available to query socket
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "-S", socketPath, "list-sessions", "-F", "#{session_name}:#{session_windows}:#{session_attached}")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// Server not responding or socket is stale
		status.IsRunning = false
		return status
	}

	status.IsRunning = true
	output := strings.TrimSpace(out.String())
	status.RawInfo = output

	lines := strings.Split(output, "\n")
	if len(lines) > 0 && lines[0] != "" {
		parts := strings.Split(lines[0], ":")
		if len(parts) >= 1 {
			status.SessionName = parts[0]
		}
		if len(parts) >= 2 {
			var w int
			if _, err := fmt.Sscanf(parts[1], "%d", &w); err == nil {
				status.Windows = w
			}
		}
		if len(parts) >= 3 {
			status.Attached = parts[2] == "1"
		}
	}

	return status
}

// Kill stops the tmux server running on the specified socket and cleans up the socket.
func Kill(socketPath string) error {
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil
	}

	binary, err := FindTmuxBinary()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-S", socketPath, "kill-server")
		_ = cmd.Run() // Ignore error if already dead
	}

	// Always ensure socket file is removed
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove socket file %q: %w", socketPath, err)
	}

	return nil
}
