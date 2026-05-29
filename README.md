# Sockt CLI

Command-line interface for managing Sockt cloud sandboxes. Built in Go with the Cobra framework.

## Installation

### From Source

```bash
cd cli
go build -o sockt .
```

Move the binary to your PATH:

```bash
mv sockt /usr/local/bin/
```

### Requirements

- Go 1.26.2+

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SOCKT_API_KEY` | API key for credits-based billing (`sockt_live_...`) | — |
| `SOCKT_SANDBOX_TOKEN` | Per-sandbox token for Lightning billing (`sbx_...`) | — |
| `SOCKT_API_URL` | API base URL | `https://api.sockt.dev` |

### Global Flags

| Flag | Description |
|------|-------------|
| `--token` | Override authentication token (takes priority over env vars) |

### Token Priority

1. `--token` flag (highest)
2. `SOCKT_SANDBOX_TOKEN` environment variable
3. `SOCKT_API_KEY` environment variable

---

## Quick Start

```bash
# Set your API key
export SOCKT_API_KEY="sockt_live_your_key_here"

# Create a sandbox and wait for it to be ready
sockt sandbox create --tier nano --wait

# Execute a command (use the sandbox ID from above)
sockt exec <sandbox-id> echo "Hello from Sockt"

# Open an interactive shell
sockt shell <sandbox-id>

# Terminate when done
sockt sandbox terminate <sandbox-id>
```

### Sandbox paths

The sandbox home directory is `/home/sandbox`. Use absolute paths for file uploads (e.g. `/home/sandbox/main.py`). For `files read`, a path relative to the home directory also works (e.g. `main.py`). Default `exec --workdir` is the sandbox home.

When using exec flags (`--timeout`, `--workdir`), put them **before** the sandbox ID:

```bash
sockt exec --timeout 60000 <sandbox-id> npm install
sockt exec --workdir /home/sandbox <sandbox-id> python3 -u main.py
```

---

## Commands Reference

### sockt tiers

List available sandbox tiers with pricing information.

```bash
sockt tiers
```

**Example Output:**

```
TIER          SATS/SEC    USD/HOUR    SATS/HOUR    SATS/DAY
nano          0.50        0.01        1800         43200
micro         1.00        0.02        3600         86400
small         2.50        0.05        9000         216000
medium        5.00        0.10        18000        432000
large         10.00       0.20        36000        864000
gpu-small     25.00       0.50        90000        2160000
```

---

### sockt sandbox create

Create a new sandbox.

```bash
sockt sandbox create [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--tier` | `-t` | `nano` | Sandbox tier (see `sockt tiers`) |
| `--billing` | `-b` | `credits` | Billing method: `credits` or `lightning` |
| `--prepaid-sats` | — | `0` | Prepaid satoshis (lightning billing only) |
| `--initial-credits` | — | `0` | Initial credits in cents (credits billing) |
| `--label` | `-l` | — | Human-readable label for the sandbox |
| `--wait` | `-w` | `false` | Wait until status is "running" and files/exec are reachable |

**Examples:**

```bash
# Create with defaults (nano tier, credits billing)
sockt sandbox create

# Create a medium-tier sandbox and wait for it to start
sockt sandbox create --tier medium --wait

# Create with Lightning billing
sockt sandbox create --tier nano --billing lightning --prepaid-sats 5000

# Create with a label
sockt sandbox create --tier small --label "my-dev-env" --wait
```

**Output:**

The sandbox ID is printed to stdout. When using `--wait`, the CLI polls every 2 seconds until the sandbox is running (up to ~2 minutes).

---

### sockt sandbox status

Get the current status of a sandbox.

```bash
sockt sandbox status <sandbox-id>
```

**Example:**

```bash
sockt sandbox status abc123-def456
```

**Output includes:** sandbox ID, status, tier, billing method, consumed balance, and (for Lightning) remaining seconds and pending invoices.

---

### sockt sandbox pause

Pause a running sandbox. Stops compute billing.

```bash
sockt sandbox pause <sandbox-id>
```

---

### sockt sandbox resume

Resume a paused sandbox.

```bash
sockt sandbox resume <sandbox-id>
```

---

### sockt sandbox terminate

Terminate a sandbox permanently.

```bash
sockt sandbox terminate <sandbox-id>
```

For Lightning-billed sandboxes, any remaining prepaid balance can be refunded if a lightning address was configured.

---

### sockt exec

Execute a command inside a sandbox. The CLI polls for output and streams it as it arrives.

```bash
sockt exec <sandbox-id> <command...>
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--workdir` | `-d` | `.` | Working directory inside the sandbox |
| `--timeout` | — | — | Execution timeout in milliseconds |
| `--poll` | — | `500` | Poll interval in milliseconds |

**Examples:**

```bash
# Simple command
sockt exec abc123 echo "hello world"

# Run in a specific directory
sockt exec abc123 --workdir /home/sandbox/myproject ls -la

# Long-running command with custom timeout (flags before sandbox-id)
sockt exec --timeout 60000 abc123 npm install

# Multiple words are joined as a single command
sockt exec abc123 cat /etc/os-release
```

**Behavior:**

- stdout is written to your terminal's stdout
- stderr is written to your terminal's stderr
- Exit code mirrors the remote command's exit code
- Polls every 500ms (configurable) until the command completes
- Refuses to run if the sandbox is paused, terminated, or awaiting payment
- Retries briefly when the pod is still starting (503 / host errors)

---

### sockt exec-cancel

Cancel a running execution.

```bash
sockt exec-cancel <execution-id>
```

---

### sockt shell

Open an interactive WebSocket shell to a sandbox. Provides a full PTY experience.

```bash
sockt shell <sandbox-id>
```

**Features:**

- Raw terminal mode (keystrokes sent immediately)
- Terminal resize synchronization (responds to window size changes)
- Signal handling (Ctrl+C sent to remote, Ctrl+D exits)
- Billing notifications displayed inline (low balance, deposit required)

**Example:**

```bash
sockt shell abc123
# You're now in an interactive shell inside the sandbox
# Use it like any terminal. Press Ctrl+D or type 'exit' to disconnect.
```

---

### sockt files write

Write a file to the sandbox filesystem.

```bash
sockt files write <sandbox-id> <remote-path> [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | — | Local file to upload (reads stdin if not set) |
| `--encoding` | — | `utf8` | Content encoding: `utf8` or `base64` |

**Examples:**

```bash
# Upload a local file
sockt files write abc123 /home/sandbox/app.py --file ./app.py

# Pipe content from stdin
echo "Hello World" | sockt files write abc123 /home/sandbox/greeting.txt

# Upload a binary file as base64
base64 image.png | sockt files write abc123 /home/sandbox/image.png --encoding base64
```

---

### sockt files read

Read a file from the sandbox filesystem.

```bash
sockt files read <sandbox-id> <remote-path> [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--encoding` | — | `utf8` | Content encoding: `utf8` or `base64` |
| `--max-bytes` | — | — | Maximum bytes to read |

**Examples:**

```bash
# Read and display a file
sockt files read abc123 /home/sandbox/output.txt

# Save to local file
sockt files read abc123 /home/sandbox/result.json > result.json

# Read binary file as base64
sockt files read abc123 /home/sandbox/image.png --encoding base64 | base64 -d > image.png

# Read only first 1024 bytes
sockt files read abc123 /home/sandbox/large-file.log --max-bytes 1024
```

---

### sockt files ls

List files in a sandbox directory.

```bash
sockt files ls <sandbox-id> [path] [flags]
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--recursive` | `-r` | `false` | List recursively |
| `--max-depth` | — | — | Maximum directory depth (with `--recursive`) |

**Examples:**

```bash
# List home directory
sockt files ls abc123

# List a specific path
sockt files ls abc123 /home/sandbox/myproject

# Recursive listing
sockt files ls abc123 --recursive

# Recursive with depth limit
sockt files ls abc123 --recursive --max-depth 3
```

**Example Output:**

```
TYPE  SIZE      NAME
d     4096      .config
d     4096      myproject
-     1523      app.py
-     89        requirements.txt
```

---

## Examples

### Complete Credits Workflow

```bash
export SOCKT_API_KEY="sockt_live_abc123"

# Create and wait
SANDBOX_ID=$(sockt sandbox create --tier small --wait)

# Install dependencies and run
sockt exec $SANDBOX_ID pip install requests
sockt exec $SANDBOX_ID python -c "import requests; print(requests.get('https://httpbin.org/ip').text)"

# Check status
sockt sandbox status $SANDBOX_ID

# Clean up
sockt sandbox terminate $SANDBOX_ID
```

### Complete Lightning Workflow

```bash
# Create with lightning billing (no API key needed)
sockt sandbox create --tier nano --billing lightning --prepaid-sats 5000

# Output includes a BOLT11 invoice - pay it with your Lightning wallet
# After payment confirms, the sandbox starts (~30-90s)

# Set the sandbox token for subsequent commands
export SOCKT_SANDBOX_TOKEN="sbx_returned_token"

# Use the sandbox
sockt exec <sandbox-id> whoami

# Terminate (remaining balance refundable)
sockt sandbox terminate <sandbox-id>
```

### Scripting with Cleanup

```bash
#!/bin/bash
set -e

export SOCKT_API_KEY="sockt_live_abc123"
SANDBOX_ID=""

cleanup() {
    if [ -n "$SANDBOX_ID" ]; then
        sockt sandbox terminate "$SANDBOX_ID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

SANDBOX_ID=$(sockt sandbox create --tier small --wait)

# Upload project files
sockt files write "$SANDBOX_ID" /home/sandbox/main.py --file ./main.py
sockt files write "$SANDBOX_ID" /home/sandbox/requirements.txt --file ./requirements.txt

# Run tests
sockt exec "$SANDBOX_ID" pip install -r /home/sandbox/requirements.txt
sockt exec "$SANDBOX_ID" --workdir /home/sandbox python -m pytest

# Download results
sockt files read "$SANDBOX_ID" /home/sandbox/test-results.xml > test-results.xml
```

### Piping Stdin for File Operations

```bash
# Generate content and upload
cat <<EOF | sockt files write abc123 /home/sandbox/config.json
{
  "debug": true,
  "port": 8080
}
EOF

# Process remote file locally
sockt files read abc123 /home/sandbox/data.csv | wc -l
```

---

## Billing

### Credits Mode

- Requires `SOCKT_API_KEY` (format: `sockt_live_...`)
- Usage is automatically deducted from your account balance
- If balance is depleted, the sandbox pauses — top up and resume
- On terminate, unused allocated credits are refunded to your account

### Lightning Mode

- No API key required (anonymous access)
- `sandbox create` returns a BOLT11 invoice — pay it to start the sandbox
- A `sandbox_token` (`sbx_...`) is returned — use it for all subsequent operations
- When balance runs low, new invoices appear in status output — pay to continue
- On terminate, provide a `lightning_address` to receive refund of unused balance

---

## Error Handling

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | CLI error (invalid args, connection failure, API error) |
| `N` | For `sockt exec`: mirrors the remote command's exit code |

### Error Output

All errors are printed to stderr with an `Error:` prefix:

```
Error: API error (401): unauthorized
Error: API error (503): pod starting, retrying...
Error: sandbox not found
```

### Automatic Retries

The CLI automatically retries on HTTP 503 when the error indicates the pod is still starting (up to 3 attempts with 2-second backoff).

---

## Troubleshooting

### "API error (401): unauthorized"

- Verify your API key or sandbox token is set correctly
- Check that the token hasn't been revoked
- Ensure you're using the right token type for the billing method

### "API error (503): no capacity"

- The requested tier has no available capacity
- Try a different tier or wait and retry

### "API error (400): pod unreachable" / host_error 404

- The sandbox hasn't finished starting yet
- Use `--wait` on create (waits for status **and** data plane), or retry after a few seconds
- Confirm the sandbox is `running` with `sockt sandbox status` (not paused or terminated)

### Shell Disconnects Immediately

- Verify the sandbox is in "running" state
- Check that your token has access to the sandbox
- Ensure no firewall is blocking WebSocket connections

### Command Hangs

- The default exec has no timeout — set `--timeout` flag
- Check sandbox status — it may have been paused due to balance depletion

---

## Development

### Project Structure

```
cli/
├── main.go              # Entry point
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── cmd/
│   ├── root.go          # Root command, global flags, client factory
│   ├── sandbox.go       # sandbox create/status/pause/resume/terminate
│   ├── exec.go          # exec and exec-cancel commands
│   ├── shell.go         # Interactive WebSocket shell
│   ├── files.go         # files write/read/ls commands
│   └── tiers.go         # tiers listing command
└── internal/
    └── api/
        ├── client.go    # HTTP client with auth and retry logic
        └── models.go    # Request/response data structures
```

### Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | v1.9.1 | CLI framework |
| `github.com/gorilla/websocket` | v1.5.0 | WebSocket (shell) |
| `golang.org/x/term` | v0.43.0 | Terminal raw mode |

### Building

```bash
cd cli
go build -o sockt .

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o sockt-darwin-arm64 .
GOOS=linux GOARCH=amd64 go build -o sockt-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o sockt.exe .
```
