# portman

**See what's using your ports. Kill rogue processes. Wait for services.**

[![Release](https://github.com/tasnimzotder/portman/actions/workflows/release.yml/badge.svg)](https://github.com/tasnimzotder/portman/actions/workflows/release.yml)
[![CI](https://github.com/tasnimzotder/portman/actions/workflows/ci.yml/badge.svg)](https://github.com/tasnimzotder/portman/actions/workflows/ci.yml)

A fast, simple CLI tool for inspecting and managing network ports on macOS.

## Features

- List all listening ports with process info
- Get detailed information about a specific port
- **Live watch mode** with real-time updates and change highlighting
- Find ports by process name, user, or PID
- Kill processes occupying ports
- Wait for ports to become available
- Process stats: Memory, CPU, file descriptors, threads

## Quick Start

```bash
# Install
brew install tasnimzotder/tap/portman

# List all listening ports
portman

# Check what's using port 3000
portman 3000

# Watch port 3000 in real-time
portman port 3000 --watch

# Watch all ports live
portman --watch

# Kill the process on port 3000
portman kill 3000
```

## Platform Support

Currently supported on **macOS only**. Linux support coming soon.
