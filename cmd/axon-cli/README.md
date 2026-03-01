# Axon CLI Tool

A simple command-line interface to interact with the Axon local server and test browser tasks.

## Build and Run

```bash
go build -o axon-cli.exe ./cmd/axon-cli

# Start server in a separate terminal:
# go run ./cmd/axon

# 1. Create a session
./axon-cli.exe -cmd create -session test1

# 2. Navigate
./axon-cli.exe -cmd navigate -session test1 -url https://github.com

# 3. Take a snapshot
./axon-cli.exe -cmd snapshot -session test1

# 4. Act (e.g., click a specific ref)
./axon-cli.exe -cmd act -session test1 -action click -ref e1
```
