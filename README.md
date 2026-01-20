# portman

> See what's using your ports. Kill rogue processes. Wait for services.

[![Release](https://github.com/tasnimzotder/portman/actions/workflows/release.yml/badge.svg)](https://github.com/tasnimzotder/portman/actions/workflows/release.yml)
[![CI](https://github.com/tasnimzotder/portman/actions/workflows/ci.yml/badge.svg)](https://github.com/tasnimzotder/portman/actions/workflows/ci.yml)
[![Platform](https://img.shields.io/badge/platform-macOS-lightgrey)](https://github.com/tasnimzotder/portman)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Installation

```bash
brew install tasnimzotder/tap/portman
```

Or download from [GitHub Releases](https://github.com/tasnimzotder/portman/releases).

> **Note:** Currently supported on macOS only. Linux support coming soon.

## Usage

```bash
portman                  # List all listening ports
portman 3000             # Details for port 3000
portman --watch          # Live-updating display
portman port 3000 -w     # Watch a single port
portman find node        # Find by process name
portman kill 3000        # Kill process on port
portman wait 5432        # Wait for port to be available
```

## Commands

| Command                  | Description                   |
| ------------------------ | ----------------------------- |
| `portman`                | List all listening ports      |
| `portman <port>`         | Show port details             |
| `portman port <port>`    | Show detailed info for a port |
| `portman find <pattern>` | Find by name/user/command     |
| `portman kill <port>`    | Kill process on port          |
| `portman wait <port>`    | Wait for port availability    |
| `portman pid <pid>`      | Find ports by PID             |

## Flags

### Global

| Flag            | Description                             |
| --------------- | --------------------------------------- |
| `--json`, `-j`  | JSON output                             |
| `--tcp`, `-t`   | Show only TCP                           |
| `--udp`, `-u`   | Show only UDP                           |
| `--sort`        | Sort by: port, pid, user, conns, uptime |
| `--watch`, `-w` | Live-updating display                   |
| `--interval`    | Watch refresh interval (default: 1s)    |
| `--no-header`   | Omit header row                         |

### kill

| Flag             | Description                             |
| ---------------- | --------------------------------------- |
| `--force`, `-f`  | Use SIGKILL instead of SIGTERM          |
| `--yes`, `-y`    | Skip confirmation                       |
| `--signal`, `-s` | Signal to send (HUP, INT, TERM, KILL)   |
| `--quiet`, `-q`  | No output on success                    |
| `--timeout`      | Wait time before SIGKILL (with --force) |

### wait

| Flag               | Description                      |
| ------------------ | -------------------------------- |
| `--timeout`        | Maximum wait time (default: 30s) |
| `--interval`, `-i` | Check interval (default: 100ms)  |
| `--exec`, `-e`     | Command to run once available    |
| `--invert`         | Wait for port to be FREE instead |
| `--quiet`, `-q`    | No output, just exit code        |

## Watch Mode

Watch mode provides a live-updating display with change highlighting:

```bash
portman --watch              # Watch all ports
portman port 3000 --watch    # Watch single port with stats
```

- Press `q` to quit
- New ports highlighted in green
- Changed values highlighted in yellow
- Removed ports shown briefly in red

## License

MIT
