# Internals

Portman is a Go CLI built with Cobra. Interactive views use Bubble Tea v2, Bubbles, and Lip Gloss.

## Packages

| Package | Responsibility |
|---------|----------------|
| `internal/cli` | Commands, flag validation, output routing, exit results |
| `internal/scanner` | Socket discovery and optional process statistics |
| `internal/model` | Processes, bindings, connections, binding identity |
| `internal/output` | Tables, process trees, JSON, CSV, TSV |
| `internal/tui` | Interactive list, detail, find, kill, wait, and watch views |
| `internal/kill` | Signals, exit checks, graceful shutdown and escalation |
| `internal/wait` | Deadline-aware polling and cancellation |

## Scanner

The `Scanner` interface exposes `ListListeners`, `GetPort`, `FindByPattern`, and `ListByPID`. macOS is the supported runtime platform.

Discovery uses `lsof -n -P -F0pcuLfPtnT -i`, with transport and port selectors when requested. NUL-delimited fields preserve process names containing whitespace. TCP LISTEN sockets and bound UDP sockets, including connected UDP sockets, are included. Unbound UDP sockets and outgoing TCP connections are excluded from listener rows.

Each binding is identified by protocol, address, port, and PID. Repeated file descriptors for the same binding are deduplicated; different addresses and owners remain separate. IPv6 wildcard sockets use `::`. Established connections are attributed by PID, protocol, family, local port, and binding address.

An empty result is valid. `lsof` exit status 1 with no output or diagnostics means no matches. Other failures, diagnostics, and partial failed scans return an error. A failed scan never proves that a port is free.

Single-port detail and kill commands reject multiple owners or protocols. Use a port range such as `34567-34567`, PID lookup, or `--tcp`/`--udp` to inspect or disambiguate bindings. Multiple addresses owned by the same PID and protocol are permitted; single-port details show the first binding in deterministic address order.

Listings batch process uptime collection in one `ps` call. Detailed CLI views enable `FetchStats` and fetch RSS, CPU, numeric file descriptors, and threads. These statistics are best effort. Wait polling uses `GetPortStatusContext` to check occupancy without fetching process statistics or selecting an owner for an action. Cancellation and timeout stop its `lsof` subprocess.

## Command results

Inspection commands share filtering, sorting, JSON, and delimited formatting. Collection JSON always contains a `listeners` array, including `[]` for empty results. Single-port JSON remains one object, or `{}` for no listener. CSV and TSV use `encoding/csv` for escaping.

Interactive wait exits when polling finishes and returns an explicit result. Only a successful wait may run `--exec`; cancellation, timeout, and persistent scanner errors fail the command. Transient scan errors are retried until the deadline. `--quiet` wait failures emit no output. A child command's exit status is preserved.

Kill commands recheck port ownership after confirmation. By default they send SIGTERM and wait for `--timeout`. With `--force`, they escalate to SIGKILL if needed. Explicit `--signal KILL` sends SIGKILL immediately. SIGHUP reports successful delivery without requiring the process to exit. Text and interactive kill share the same signal implementation, and interactive failures reach the CLI exit status.

## Watch

Watch schedules the next scan after the current scan finishes, so slow scans do not overlap. Errors clear after a successful refresh. Binding keys detect owner replacement and distinguish addresses. Previous connection counts and process snapshots are stored during model updates; rendering does not mutate them.

## Exit codes and checks

Exit 0 means success; exit 1 means a command error, cancellation, timeout, or failed termination. `wait --exec` preserves the child command's nonzero exit code. These results apply to both text and interactive modes.

Run `go test -race ./...`, `go vet ./...`, and `go build ./...`. Regression coverage includes machine-readable scanner fixtures, actual TCP/UDP sockets, CLI output and exit codes, disposable process signaling, and TUI model outcomes. CI and releases read the Go version from `go.mod`.
