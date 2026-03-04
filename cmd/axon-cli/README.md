# Axon CLI Tool

A command-line interface to interact with the Axon browser automation server.

## Installation

```bash
go build -o axon-cli.exe ./cmd/axon-cli
```

## Configuration

The CLI supports configuration files in multiple locations (in order of precedence):
1. `./axon-cli.yaml` (current directory)
2. `$HOME/.axon/axon-cli.yaml`
3. `/etc/axon/axon-cli.yaml`

Example configuration file:

```yaml
api_url: "http://localhost:8020/api/v1"
```

You can also override the API URL using the `AXON_API_URL` environment variable:

```bash
export AXON_API_URL=http://localhost:8020/api/v1
```

Or using the `--api-url` flag with any command.

## Usage

```bash
# Start the Axon server first (in a separate terminal):
# go run ./cmd/axon

# Or use the pre-built binary:
# ./axon.exe
```

### Session Management

```bash
# Create a new session
./axon-cli.exe session create mysession

# List all sessions
./axon-cli.exe session list

# Get session details
./axon-cli.exe session info mysession

# Delete a session
./axon-cli.exe session delete mysession
```

### Navigate

```bash
# Navigate to a URL
./axon-cli.exe navigate mysession https://github.com

# Or with flags
./axon-cli.exe navigate --session mysession --url https://github.com
```

### Snapshot

```bash
# Take a snapshot of the current page
./axon-cli.exe snapshot mysession

# With specific element focus
./axon-cli.exe snapshot mysession --ref e1
```

### Act

```bash
# Click an element
./axon-cli.exe act mysession click --ref e1

# Fill an input field
./axon-cli.exe act mysession fill --ref e2 --value "Hello World"

# Hover over an element
./axon-cli.exe act mysession hover --ref e3

# Select an option
./axon-cli.exe act mysession select --ref e4 --value "option1"
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `session create <id>` | Create a new session |
| `session list` | List all active sessions |
| `session info <id>` | Get session details |
| `session delete <id>` | Delete a session |
| `navigate <session> <url>` | Navigate to a URL |
| `snapshot <session>` | Get page snapshot |
| `act <session> <action>` | Perform an action |

## Options

| Flag | Description |
|------|-------------|
| `--api-url <url>` | Override API URL |
| `-h, --help` | Show help |
| `--version` | Show version |
