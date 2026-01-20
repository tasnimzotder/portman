# Commands

## List Ports

List all listening ports on your system.

```bash
portman
```

**Output columns:**

| Column | Description |
|--------|-------------|
| PORT | Port number |
| PROTO | Protocol (TCP/UDP) |
| PID | Process ID |
| USER | Process owner |
| CONNS | Active connections |
| UPTIME | Process uptime |
| PROCESS | Process name |

## Port Details

Get detailed information about a specific port.

```bash
portman <port>
# or
portman port <port>
```

**Example:**
```bash
portman 3000
```

Shows process info plus active connections with remote addresses.

## Find

Search for ports by pattern (matches process name, command, user, port, or PID).

```bash
portman find <pattern>
```

**Examples:**
```bash
portman find node      # Find Node.js processes
portman find postgres  # Find PostgreSQL
portman find 8080      # Find port 8080
```

## Kill

Kill a process using a specific port.

```bash
portman kill <port>
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip confirmation |
| `--yes` | `-y` | Auto-confirm |
| `--signal` | `-s` | Signal to send (default: TERM) |
| `--timeout` | | Timeout for graceful shutdown |
| `--quiet` | `-q` | Suppress output |

**Examples:**
```bash
portman kill 3000           # Kill with confirmation
portman kill 3000 -y        # Kill without confirmation
portman kill 3000 -s KILL   # Force kill (SIGKILL)
```

## Wait

Wait for a port to become available (free) or occupied.

```bash
portman wait <port>
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--timeout` | Maximum wait time (default: 30s) |
| `--interval` | Check interval (default: 500ms) |
| `--exec` | Command to run when port is ready |
| `--quiet` | Suppress output |
| `--invert` | Wait for port to be occupied |

**Examples:**
```bash
portman wait 5432                      # Wait for PostgreSQL to free up
portman wait 3000 --exec "npm start"   # Run command when port is free
portman wait 8080 --invert             # Wait for service to start
```

## PID Lookup

Find all ports used by a specific process ID.

```bash
portman pid <pid>
```

**Example:**
```bash
portman pid 1234
```

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | Output in JSON format |
| `--tcp` | `-t` | Show only TCP ports |
| `--udp` | `-u` | Show only UDP ports |
| `--sort` | | Sort by: port, pid, user, conns, uptime |
