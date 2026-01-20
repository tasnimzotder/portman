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
portman                # List all listening ports
portman 3000           # Details for port 3000
portman find node      # Find by process name
portman kill 3000      # Kill process on port
portman wait 5432      # Wait for port to be available
```

## Commands

| Command                  | Description                |
| ------------------------ | -------------------------- |
| `portman`                | List all listening ports   |
| `portman <port>`         | Show port details          |
| `portman find <pattern>` | Find by name/user/pid      |
| `portman kill <port>`    | Kill process on port       |
| `portman wait <port>`    | Wait for port availability |
| `portman pid <pid>`      | Find ports by PID          |

## Flags

| Flag           | Description                             |
| -------------- | --------------------------------------- |
| `--json`, `-j` | JSON output                             |
| `--tcp`, `-t`  | Show only TCP                           |
| `--udp`, `-u`  | Show only UDP                           |
| `--sort`       | Sort by: port, pid, user, conns, uptime |

## License

MIT
