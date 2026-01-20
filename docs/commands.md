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

Shows process info, connections, and stats:

- **Process**: PID, command, user, uptime
- **Listening**: Address and protocol
- **Connections**: Remote addresses and states
- **Stats**: Memory (RSS), CPU %, file descriptors, threads

## Watch Mode

Monitor ports in real-time with live updates and change highlighting.

### Watch All Ports

```bash
portman --watch
portman -w
```

Shows a live-updating table of all listening ports:
- **Green** highlighting for newly added ports
- **Red** notification for removed ports
- Footer shows total count with +new/-removed

### Watch Single Port

```bash
portman port <port> --watch
portman port <port> -w
```

**Example:**
```bash
portman port 3000 --watch
```

Shows detailed live view with change highlighting:
- **Yellow** highlighting for changed values (connections, memory, CPU, FDs, threads)
- **Green** "Port became active" when port starts listening
- **Red** "Process exited" when port closes

**Customizing refresh interval:**
```bash
portman --watch --interval 500ms
portman port 3000 -w --interval 2s
```

Press `q` to quit watch mode.

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
| `--force` | `-f` | Use SIGKILL instead of SIGTERM |
| `--yes` | `-y` | Skip confirmation prompt |
| `--signal` | `-s` | Signal to send: HUP, INT, TERM, KILL (default: TERM) |
| `--timeout` | | Wait time before SIGKILL (default: 5s) |
| `--quiet` | `-q` | Suppress output |

**Examples:**
```bash
portman kill 3000           # Kill with confirmation
portman kill 3000 -y        # Kill without confirmation
portman kill 3000 -s KILL   # Force kill (SIGKILL)
portman kill 3000 --timeout 10s  # Wait 10s before force kill
```

## Wait

Wait for a port to become available (free) or occupied.

```bash
portman wait <port>
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--timeout` | | Maximum wait time (default: 30s) |
| `--interval` | `-i` | Check interval (default: 100ms) |
| `--exec` | `-e` | Command to run when port is ready |
| `--quiet` | `-q` | Suppress output, just exit code |
| `--invert` | | Wait for port to be OCCUPIED instead of free |

**Examples:**
```bash
portman wait 5432                      # Wait for port to be free
portman wait 3000 --exec "npm start"   # Run command when port is free
portman wait 8080 --invert             # Wait for service to start
portman wait 3000 --timeout 10s        # Wait max 10 seconds
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

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | `-j` | false | Output in JSON format |
| `--no-header` | | false | Omit header row in table output |
| `--tcp` | `-t` | false | Show only TCP ports |
| `--udp` | `-u` | false | Show only UDP ports |
| `--sort` | | port | Sort by: port, pid, user, conns, uptime |
| `--watch` | `-w` | false | Live updating display |
| `--interval` | | 1s | Watch mode refresh interval |
