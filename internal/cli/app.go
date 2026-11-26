package cli

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"

	"ctmux/internal/config"
	"ctmux/internal/session"
)

// BuildInfo holds compilation metadata.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// App encapsulates the CLI application state and dependencies.
type App struct {
	ConfigFS  fs.FS
	BuildInfo BuildInfo
	Stdout    io.Writer
	Stderr    io.Writer
	Runner    session.Runner
}

// NewApp creates a new CLI App instance with standard defaults.
func NewApp(configFS fs.FS, info BuildInfo) *App {
	return &App{
		ConfigFS:  configFS,
		BuildInfo: info,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		Runner:    session.DefaultRunner,
	}
}

// Run parses arguments and executes the requested command.
func (a *App) Run(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(a.Stderr, "Error: No arguments provided.")
		a.PrintUsage(a.Stderr)
		return fmt.Errorf("no arguments provided")
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "center", "c":
		return a.handleSession("center", cmdArgs)
	case "hub", "h":
		return a.handleSession("hub", cmdArgs)
	case "node", "n":
		return a.handleSession("node", cmdArgs)
	case "reset":
		return a.handleReset(cmdArgs)
	case "status", "ls", "list":
		return a.handleStatus(cmdArgs)
	case "kill":
		return a.handleKill(cmdArgs)
	case "config":
		return a.handleConfig(cmdArgs)
	case "version", "-v", "--version":
		return a.handleVersion(cmdArgs)
	case "help", "-h", "--help":
		a.PrintUsage(a.Stdout)
		return nil
	default:
		fmt.Fprintf(a.Stderr, "Error: Invalid argument %q.\n\n", cmd)
		a.PrintUsage(a.Stderr)
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func (a *App) handleSession(tier string, args []string) error {
	fs := flag.NewFlagSet(tier, flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	var (
		cacheDir    string
		detach      bool
		dryRun      bool
		sessionName string
	)

	fs.StringVar(&cacheDir, "cache-dir", "", "Custom directory for configs and sockets")
	fs.BoolVar(&detach, "d", false, "Start session in background (detached)")
	fs.BoolVar(&detach, "detach", false, "Start session in background (detached)")
	fs.BoolVar(&dryRun, "dry-run", false, "Print tmux command without executing")
	fs.StringVar(&sessionName, "s", "zero", "Tmux session name")
	fs.StringVar(&sessionName, "session", "zero", "Tmux session name")

	if err := fs.Parse(args); err != nil {
		return err
	}

	resolvedCacheDir, err := config.GetCacheDir(cacheDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error resolving cache directory: %v\n", err)
		return err
	}

	configPath, updated, err := config.EnsureConfigFile(a.ConfigFS, resolvedCacheDir, tier)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error preparing configuration: %v\n", err)
		return err
	}
	if updated {
		fmt.Fprintf(a.Stdout, "[ctmux] Synchronized %s configuration to %s\n", tier, configPath)
	}

	socketPath := config.GetSocketPath(resolvedCacheDir, tier)

	s := &session.Session{
		Tier:        tier,
		CacheDir:    resolvedCacheDir,
		ConfigPath:  configPath,
		SocketPath:  socketPath,
		SessionName: sessionName,
		Detach:      detach,
		DryRun:      dryRun,
		ExtraArgs:   fs.Args(),
	}

	if dryRun {
		tmuxCmd := strings.Join(s.BuildArgs(), " ")
		fmt.Fprintf(a.Stdout, "[ctmux] Dry run: %s\n", tmuxCmd)
		return nil
	}

	fmt.Fprintf(a.Stdout, "[CTMUX] Starting %s...\n", tier)

	if err := s.Run(a.Runner); err != nil {
		fmt.Fprintf(a.Stderr, "Error starting %s session: %v\n", tier, err)
		return err
	}

	return nil
}

func (a *App) handleReset(args []string) error {
	fs := flag.NewFlagSet("reset", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	var cacheDir string
	fs.StringVar(&cacheDir, "cache-dir", "", "Custom cache directory to reset")

	if err := fs.Parse(args); err != nil {
		return err
	}

	resolvedCacheDir, err := config.GetCacheDir(cacheDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error resolving cache directory: %v\n", err)
		return err
	}

	removed, err := config.ResetCache(resolvedCacheDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error resetting cache: %v\n", err)
		return err
	}

	if len(removed) == 0 {
		fmt.Fprintf(a.Stdout, "Cache is already clean (%s).\n", resolvedCacheDir)
	} else {
		for _, file := range removed {
			fmt.Fprintf(a.Stdout, "Removed %s\n", file)
		}
		fmt.Fprintf(a.Stdout, "Reset %d files in %s\n", len(removed), resolvedCacheDir)
	}

	return nil
}

func (a *App) handleStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	var cacheDir string
	fs.StringVar(&cacheDir, "cache-dir", "", "Custom cache directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	resolvedCacheDir, err := config.GetCacheDir(cacheDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error resolving cache directory: %v\n", err)
		return err
	}

	w := tabwriter.NewWriter(a.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TIER\tSTATUS\tWINDOWS\tATTACHED\tSOCKET")

	for _, tier := range config.SupportedTiers {
		sockPath := config.GetSocketPath(resolvedCacheDir, tier)
		status := session.GetStatus(tier, sockPath)

		statusStr := "stopped"
		windowsStr := "-"
		attachedStr := "-"

		if status.IsRunning {
			statusStr = "running"
			windowsStr = fmt.Sprintf("%d", status.Windows)
			if status.Attached {
				attachedStr = "yes"
			} else {
				attachedStr = "no"
			}
		} else if status.SocketExists {
			statusStr = "socket idle"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", tier, statusStr, windowsStr, attachedStr, sockPath)
	}

	w.Flush()
	return nil
}

func (a *App) handleKill(args []string) error {
	fs := flag.NewFlagSet("kill", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	var cacheDir string
	fs.StringVar(&cacheDir, "cache-dir", "", "Custom cache directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	target := "all"
	if len(fs.Args()) > 0 {
		target = fs.Args()[0]
	}

	resolvedCacheDir, err := config.GetCacheDir(cacheDir)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error resolving cache directory: %v\n", err)
		return err
	}

	var tiersToKill []string
	if target == "all" {
		tiersToKill = config.SupportedTiers
	} else if config.IsValidTier(target) {
		tiersToKill = []string{target}
	} else {
		fmt.Fprintf(a.Stderr, "Error: Unknown tier %q. Supported tiers: center, hub, node, all\n", target)
		return fmt.Errorf("invalid tier: %s", target)
	}

	for _, tier := range tiersToKill {
		sockPath := config.GetSocketPath(resolvedCacheDir, tier)
		if err := session.Kill(sockPath); err != nil {
			fmt.Fprintf(a.Stderr, "Failed to kill %s: %v\n", tier, err)
		} else {
			fmt.Fprintf(a.Stdout, "Terminated %s session (socket: %s)\n", tier, sockPath)
		}
	}

	return nil
}

func (a *App) handleConfig(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(a.Stderr, "Usage: ctmux config <center|hub|node>")
		return fmt.Errorf("tier required for config command")
	}

	tier := args[0]
	if tier == "show" && len(args) > 1 {
		tier = args[1]
	}

	if !config.IsValidTier(tier) {
		fmt.Fprintf(a.Stderr, "Error: Unknown tier %q. Expected one of: center, hub, node\n", tier)
		return fmt.Errorf("invalid tier: %s", tier)
	}

	content, err := config.ReadEmbeddedConfig(a.ConfigFS, tier)
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error reading embedded config: %v\n", err)
		return err
	}

	a.Stdout.Write(content)
	return nil
}

func (a *App) handleVersion(args []string) error {
	version := a.BuildInfo.Version
	if version == "" {
		version = "dev"
	}
	commit := a.BuildInfo.Commit
	if commit == "" {
		commit = "none"
	}
	date := a.BuildInfo.Date
	if date == "" {
		date = "unknown"
	}

	fmt.Fprintf(a.Stdout, "ctmux version %s (commit: %s, built: %s, %s/%s)\n",
		version, commit, date, runtime.GOOS, runtime.GOARCH)
	return nil
}

// PrintUsage prints formatted CLI usage instructions.
func (a *App) PrintUsage(w io.Writer) {
	fmt.Fprintln(w, "ctmux - Layered tmux session manager")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ctmux <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Session Commands:")
	fmt.Fprintln(w, "  center, c       Attach or launch Center tmux session (prefix: F7)")
	fmt.Fprintln(w, "  hub, h          Attach or launch Hub tmux session (prefix: F8)")
	fmt.Fprintln(w, "  node, n         Attach or launch Node tmux session (prefix: C-x)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Management Commands:")
	fmt.Fprintln(w, "  status, ls      Show active ctmux sessions and socket states")
	fmt.Fprintln(w, "  kill <tier>     Kill a running tier session ('center', 'hub', 'node', or 'all')")
	fmt.Fprintln(w, "  config <tier>   Dump embedded tmux configuration to stdout")
	fmt.Fprintln(w, "  reset           Remove cached config files and sockets")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "General Commands:")
	fmt.Fprintln(w, "  version, -v     Display ctmux version information")
	fmt.Fprintln(w, "  help, -h        Show this help message")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Session Flags:")
	fmt.Fprintln(w, "  -d, -detach     Start session in background without attaching")
	fmt.Fprintln(w, "  -dry-run        Print tmux command without executing")
	fmt.Fprintln(w, "  -s, -session    Custom session name (default: zero)")
	fmt.Fprintln(w, "  -cache-dir      Custom directory for sockets and configs (env: CTMUX_CACHE_DIR)")
}
