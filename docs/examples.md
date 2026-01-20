# Examples

## Development Workflow

### Check if port is in use before starting dev server

```bash
portman 3000 || npm run dev
```

### Kill whatever is using your dev port

```bash
portman kill 3000 -y && npm run dev
```

### Wait for database to be ready

```bash
portman wait 5432 --invert --exec "npm run migrate"
```

## Watch Mode

### Monitor all ports in real-time

```bash
portman --watch
```

### Watch a specific port during development

```bash
# Watch your dev server port
portman port 3000 --watch

# Faster refresh for debugging
portman port 3000 -w --interval 500ms
```

### Monitor database connections

```bash
# Watch PostgreSQL port
portman port 5432 --watch

# Watch MySQL port
portman port 3306 -w
```

### Filter and watch

```bash
# Watch only TCP ports
portman --watch --tcp

# Watch only UDP ports
portman --watch --udp
```

## Debugging

### Find all Node.js processes using ports

```bash
portman find node
```

### Get JSON output for scripting

```bash
portman --json | jq '.listeners[] | select(.port == 3000)'
```

### Sort by connection count to find busy services

```bash
portman --sort conns
```

### Check process stats for a port

```bash
portman 3000
# Shows: Memory, CPU %, FDs, Threads
```

## System Administration

### List all ports sorted by uptime

```bash
portman --sort uptime
```

### Find which user is running a service

```bash
portman find postgres
```

### Kill a stuck process with SIGKILL

```bash
portman kill 8080 -s KILL -y
```

### Monitor system ports continuously

```bash
# Watch all ports with 2-second refresh
portman --watch --interval 2s
```

## Scripting

### Check if a port is free (exit code)

```bash
if portman 3000 > /dev/null 2>&1; then
    echo "Port 3000 is in use"
else
    echo "Port 3000 is free"
fi
```

### Get PID of process on port

```bash
portman 3000 --json | jq -r '.process.pid'
```

### Wait with timeout

```bash
portman wait 3000 --timeout 10s || echo "Timeout waiting for port"
```

### Output without headers for parsing

```bash
portman --no-header | awk '{print $1, $3}'
```

### Get memory usage of a port's process

```bash
portman 3000 --json | jq -r '.stats.memoryRSS'
```
