# ctmux

A modern, lightweight layered [tmux](https://github.com/tmux/tmux) session manager written in Go with **zero external dependencies**.

`ctmux` manages tiered, nested tmux workflows (Center, Hub, Node), allowing you to run multiple nested tmux sessions without key binding collisions.

Originating from [tmux-center](https://github.com/carltonf/iclocal/tree/master/bin/tmux-center).

---

## Architecture: Tiered Sessions

| Tier | Default Prefix | Status Bar Style | Intended Purpose |
| :--- | :--- | :--- | :--- |
| **`center`** | <kbd>F7</kbd> | Steel blue | Root orchestrator managing multiple regional hubs |
| **`hub`** | <kbd>F8</kbd> | White / Bright | Regional gateway managing multiple remote node sessions |
| **`node`** | <kbd>Ctrl</kbd>+<kbd>x</kbd> | Cyan / Dark | Leaf workstation or remote server session |

Each tier runs with its own dedicated Unix domain socket and configuration, preventing nested tmux sessions from intercepting each other's prefix keys.

---

## Features

- **Self-Contained Single Binary:** Embedded tmux configurations (`center.conf`, `hub.conf`, `node.conf`) using Go's `embed` package.
- **Automatic Config Synchronization:** Automatically detects changes in the embedded configurations and synchronizes them to the cache directory.
- **XDG & OS Standard Compliance:** Automatically places configs and sockets in standard directories (`$XDG_CACHE_HOME/ctmux` or `~/.cache/ctmux`), or custom paths via `CTMUX_CACHE_DIR` or `--cache-dir`.
- **Zero External Dependencies:** Built entirely with the Go standard library.
- **Cross-Platform:** Supports Linux, macOS, and Windows (via Cygwin/MSYS).
- **Session Management:** Built-in commands to inspect running sessions (`status`), kill sessions (`kill`), inspect configs (`config`), and clean up caches (`reset`).

---

## Installation

### From Source (Go 1.22+)

```bash
git clone https://github.com/carltonf/ctmux.git
cd ctmux
go install .
```

Or build locally:

```bash
go build -o ctmux .
```

---

## Usage

```text
ctmux <command> [flags]
```

### Commands

#### Starting Sessions
- `ctmux center` (or `ctmux c`): Attach or launch Center tmux session (prefix: <kbd>F7</kbd>)
- `ctmux hub` (or `ctmux h`): Attach or launch Hub tmux session (prefix: <kbd>F8</kbd>)
- `ctmux node` (or `ctmux n`): Attach or launch Node tmux session (prefix: <kbd>Ctrl</kbd>+<kbd>x</kbd>)

#### Managing Sessions & Configuration
- `ctmux status` (or `ctmux ls`): View running status, socket paths, and active windows for all tiers.
- `ctmux kill <tier>`: Terminate a running tier (`center`, `hub`, `node`, or `all`).
- `ctmux config <tier>`: Print the embedded tmux configuration for inspection.
- `ctmux reset`: Remove cached configuration and socket files.
- `ctmux version`: Show version, commit, build date, and OS/architecture.
- `ctmux help`: Display usage guide.

---

## Flags & Environment Variables

| Flag / Option | Description |
| :--- | :--- |
| `-d`, `-detach` | Start the session in the background without attaching |
| `-dry-run` | Print the tmux invocation command without executing it |
| `-s`, `-session` | Custom session name (defaults to `zero`) |
| `-cache-dir` | Custom directory for sockets and configs |
| `CTMUX_CACHE_DIR` | Environment variable to override the default cache directory |

### Examples

Launch `center` in background:
```bash
ctmux center -d
```

Preview tmux command without running:
```bash
ctmux hub -dry-run
```

Inspect embedded configuration:
```bash
ctmux config node
```

Check active sessions:
```bash
ctmux status
```

---

## Development & Testing

Run unit tests:
```bash
go test -v ./...
```

Run code analysis:
```bash
go vet ./...
```

Build for multiple platforms:
```bash
# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -o out/ctmux-linux .

# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o out/ctmux.exe .
```
