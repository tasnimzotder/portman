# Internals

This document describes the internal architecture and implementation details of portman.

## Package Structure

```
internal/
├── cli/          # Command definitions (Cobra)
├── scanner/      # Port scanning and process detection
├── model/        # Data structures
├── output/       # Formatters (table, JSON)
├── ui/           # Terminal UI (watch mode)
└── kill/         # Process termination
```

## Scanner

The scanner package handles port detection and process information gathering.

### Interface

```go
type Scanner interface {
    ListListeners() ([]model.Listener, error)
    GetPort(port int) (*model.Listener, error)
    FindByPattern(pattern string) ([]model.Listener, error)
}
```

### macOS Implementation

**File:** `internal/scanner/darwin.go`

Uses native macOS commands:

| Command | Purpose |
|---------|---------|
| `lsof -i -n -P` | List network connections |
| `ps -o rss=,%cpu= -p <pid>` | Get memory and CPU |
| `lsof -p <pid>` | Count file descriptors |
| `ps -M -p <pid>` | Count threads |

**Options:**

```go
type Options struct {
    IncludeTCP   bool  // Include TCP ports (default: true)
    IncludeUDP   bool  // Include UDP ports (default: true)
    IncludeIPv6  bool  // Include IPv6 (default: true)
    ResolveNames bool  // Resolve hostnames (default: false)
    FetchStats   bool  // Fetch process stats (default: false)
}
```

## Data Models

**File:** `internal/model/types.go`

### Listener

Represents a listening port:

```go
type Listener struct {
    Port            int
    Protocol        string        // "tcp" or "udp"
    Address         string        // Binding address
    PID             int
    Process         *Process
    Connections     []Connection
    ConnectionCount int
    Stats           *ProcessStats
}
```

### Process

```go
type Process struct {
    PID           int
    Name          string
    Command       string
    Cmdline       []string
    User          string
    UID           int
    StartTime     time.Time
    UptimeSeconds int64
}
```

### ProcessStats

```go
type ProcessStats struct {
    MemoryRSS   int64   // Memory in bytes
    CPUPercent  float64
    FDCount     int
    ThreadCount int
}
```

### Connection

```go
type Connection struct {
    LocalAddr       string
    LocalPort       int
    RemoteAddr      string
    RemotePort      int
    State           string
    DurationSeconds int64
}
```

## Output Formatters

### Table Formatter

**File:** `internal/output/table.go`

Formats output as aligned columns:

```
PORT     PROTO    PID      USER       COMMAND      CONNS   UPTIME
3000     tcp      1234     tasnim     node         2       12h34m
```

Features:
- Column alignment with fixed widths
- Truncation with ellipsis for long values
- Optional header row (`--no-header`)

### JSON Formatter

**File:** `internal/output/json.go`

Outputs structured JSON with metadata:

```json
{
  "listeners": [...],
  "scanTime": "2024-01-20T12:34:56Z",
  "platform": "darwin",
  "hostname": "machine.local"
}
```

### Sorting

**File:** `internal/output/helpers.go`

```go
func SortListeners(listeners []model.Listener, by string)
```

Sort options: `port`, `pid`, `user`, `conns`, `uptime`

## UI / Watch Mode

**Files:** `internal/ui/watch.go`, `internal/ui/render.go`, `internal/ui/ansi.go`

### ANSI Escape Codes

```go
const (
    ClearScreen = "\033[2J"
    MoveCursor  = "\033[H"
    ClearLine   = "\033[2K"
    HideCursor  = "\033[?25l"
    ShowCursor  = "\033[?25h"

    Green  = "\033[32m"  // New ports
    Red    = "\033[31m"  // Removed ports
    Yellow = "\033[33m"  // Changed values
    Cyan   = "\033[36m"  // Headers
    Bold   = "\033[1m"
    Dim    = "\033[2m"
    Reset  = "\033[0m"
)
```

### Flicker-Free Rendering

To avoid screen flicker on refresh:

1. **First render**: Clear screen with `ClearAndReset()`
2. **Subsequent renders**: Move cursor to top with `MoveToTop()` (no clear)
3. **Each line**: Clear with `PrintLine()` before writing

```go
if isFirstRender {
    ClearAndReset()
    isFirstRender = false
} else {
    MoveToTop()
}
```

### Change Detection

**PortSnapshot** tracks previous values:

```go
type PortSnapshot struct {
    PID             int
    ConnectionCount int
    MemoryRSS       int64
    CPUPercent      float64
    FDCount         int
    ThreadCount     int
}
```

Compare current vs previous to highlight changes in yellow.

### Terminal Raw Mode

For single-key input (quit with 'q'):

```go
// macOS: stty -f /dev/tty cbreak -echo
cmd := exec.Command("stty", "-f", "/dev/tty", "cbreak", "-echo")
```

Read from `/dev/tty` for key input:

```go
tty, _ := os.Open("/dev/tty")
buf := make([]byte, 1)
tty.Read(buf)
```

## Kill / Signal Handling

**File:** `internal/kill/kill.go`

### Supported Signals

```go
var SignalMap = map[string]syscall.Signal{
    "HUP":     syscall.SIGHUP,
    "SIGHUP":  syscall.SIGHUP,
    "INT":     syscall.SIGINT,
    "SIGINT":  syscall.SIGINT,
    "TERM":    syscall.SIGTERM,
    "SIGTERM": syscall.SIGTERM,
    "KILL":    syscall.SIGKILL,
    "SIGKILL": syscall.SIGKILL,
}
```

### Graceful Shutdown

```go
func KillWithTimeout(pid int, timeout time.Duration) error
```

1. Send SIGTERM
2. Wait up to `timeout` for process to exit
3. If still running, send SIGKILL

### Helper Functions

```go
func Kill(pid int, signal syscall.Signal) error
func WaitForExit(pid int, timeout time.Duration) error
func IsRunning(pid int) bool
func ParseSignal(name string) (syscall.Signal, error)
```

## CLI Structure

**File:** `internal/cli/root.go`

Uses [Cobra](https://github.com/spf13/cobra) for command handling:

```go
var RootCmd = &cobra.Command{
    Use:   "portman [port]",
    Short: "See what's using your ports",
    RunE:  runRoot,
}
```

### Subcommands

| Command | File | Description |
|---------|------|-------------|
| `find` | `find.go` | Search by pattern |
| `port` | `port.go` | Port details |
| `pid` | `pid.go` | PID lookup |
| `kill` | `kill.go` | Kill process |
| `wait` | `wait.go` | Wait for port |

### Flag Inheritance

Global flags are defined with `PersistentFlags()` and inherited by all subcommands:

```go
RootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
RootCmd.PersistentFlags().BoolVarP(&watchMode, "watch", "w", false, "Live display")
```

## Error Handling

### Kill Errors

```go
var (
    ErrPermissionDenied = errors.New("permission denied")
    ErrProcessNotFound  = errors.New("process not found")
    ErrProcessRunning   = errors.New("process still running")
)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error / timeout |
| 2 | Permission denied |
| 3 | Process won't terminate |

## Adding New Features

### New Command

1. Create `internal/cli/newcmd.go`
2. Define `var newCmd = &cobra.Command{...}`
3. Add to root: `RootCmd.AddCommand(newCmd)` in `init()`

### New Platform

1. Create `internal/scanner/linux.go` with build tag
2. Implement `Scanner` interface
3. Use platform-specific commands (e.g., `ss`, `netstat`)

### New Output Format

1. Create `internal/output/format.go`
2. Implement formatter with `Format([]model.Listener) string`
3. Add flag and routing in CLI
