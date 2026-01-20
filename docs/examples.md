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
