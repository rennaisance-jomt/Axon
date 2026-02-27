# Axon — Deployment Guide
## Configuration and Production Setup

**Version:** 1.0  
**Date:** February 2026

---

## Configuration Files

### Default Location

Axon looks for config in this order:
1. `./config.yaml` (current directory)
2. `~/.axon/config.yaml` (home directory)
3. `/etc/axon/config.yaml` (system-wide)

### Example Configuration

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8020
  read_timeout: 30s
  write_timeout: 30s

browser:
  headless: true
  binary_path: ""  # Auto-detect if empty
  pool_size: 5
  launch_options:
    args:
      - "--no-sandbox"
      - "--disable-setuid-sandbox"
      - "--disable-dev-shm-usage"

security:
  ssrf:
    enabled: true
    allow_private_network: false
    domain_allowlist: []
    domain_denylist: []
    scheme_allowlist:
      - "https"
      - "http"
  prompt_injection:
    enabled: true
    mode: "warn"  # warn | strip | block
    sensitivity: "medium"
  reversibility:
    require_confirm: true
    action_budget_per_hour: 10
    escalate_on_budget_exceeded: true

storage:
  path: "./data/axon.db"
  session_ttl: 24h
  audit_retention: 90d

logging:
  level: "info"  # debug | info | warn | error
  format: "json"  # json | text
  output: "stdout"  # stdout | file | syslog

intent:
  enabled: true
  classifier_url: "http://localhost:8021"  # Python gRPC bridge
```

---

## Environment Variables

Override config via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `AXON_HOST` | `0.0.0.0` | Server host |
| `AXON_PORT` | `8020` | Server port |
| `AXON_DATA_DIR` | `./data` | Storage directory |
| `AXON_LOG_LEVEL` | `info` | Log level |
| `AXON_HEADLESS` | `true` | Run browser headless |
| `AXON_BROWSER_POOL` | `5` | Max concurrent browsers |
| `AXON_CONFIG` | - | Path to config file |

---

## Deployment Modes

### Development (Local)

```bash
# Quick start
./axon

# With custom config
./axon --config dev.yaml

# Enable debug logging
AXON_LOG_LEVEL=debug ./axon
```

### Production (Server)

```bash
# Build binary
CGO_ENABLED=0 go build -ldflags="-s -w" -o axon ./cmd/axon

# Run as service (systemd)
sudo systemctl start axon

# Or Docker
docker run -d -p 8020:8020 -v $(pwd)/data:/data axon:latest
```

### Docker Deployment

```bash
# Build image
docker build -t axon:latest .

# Run container
docker run -d \
  --name axon \
  -p 8020:8020 \
  -v ./data:/data \
  -v ./config.yaml:/app/config.yaml \
  axon:latest

# With Docker Compose
docker-compose up -d
```

### Docker Compose Example

```yaml
version: "3.8"

services:
  axon:
    build: .
    ports:
      - "8020:8020"
    volumes:
      - ./data:/data
      - ./config.yaml:/app/config.yaml
    environment:
      - AXON_LOG_LEVEL=info
      - AXON_BROWSER_POOL=3
    restart: unless-stopped

  intent-classifier:
    build: ./intent-service
    ports:
      - "8021:8021"
    restart: unless-stopped
```

---

## Session Profiles (Auth Vault)

### Creating a Profile

```bash
# Navigate to login page manually
# (Axon will detect this automatically)

# Or export from browser:
# 1. Log into website in regular Chrome
# 2. Export cookies: Chrome → Settings → Advanced → Site settings → Cookies → Export
# 3. Save as JSON
```

### Supported Profile Formats

| Format | Extension | Example |
|--------|-----------|---------|
| Playwright | `.json` | `x_session.json` |
| Cookie-Editor | `.json` | `github_cookies.json` |
| Netscape | `.txt` | `gmail_cookies.txt` |

### Using a Profile

```bash
# Create session with profile
curl -X POST http://localhost:8020/api/v1/sessions \
  -d '{"id": "x_main", "profile": "./profiles/x_session.json"}'
```

---

## Security Configuration

### SSRF Protection

```yaml
security:
  ssrf:
    enabled: true
    allow_private_network: false
    domain_allowlist:
      - "example.com"
      - "*.google.com"
    domain_denylist:
      - "evil.com"
      - "*.phishing.net"
```

### Prompt Injection

```yaml
security:
  prompt_injection:
    enabled: true
    mode: "warn"  # Options: warn | strip | block
    sensitivity: "medium"  # low | medium | high
```

- **warn**: Flag suspicious content, return to agent
- **strip**: Remove suspicious content before returning
- **block**: Reject snapshot if injection detected

### Action Reversibility

```yaml
security:
  reversibility:
    require_confirm: true
    action_budget_per_hour: 10
    escalate_on_budget_exceeded: true
```

---

## Resource Requirements

### Minimum (Development)

| Resource | Value |
|----------|-------|
| CPU | 1 core |
| RAM | 2 GB |
| Disk | 500 MB |
| Browser instances | 1-2 |

### Recommended (Production)

| Resource | Value |
|----------|-------|
| CPU | 2-4 cores |
| RAM | 4-8 GB |
| Disk | 5 GB (for logs/sessions) |
| Browser instances | 5-10 concurrent |

---

## Health Checks

### Endpoint

```bash
curl http://localhost:8020/health
```

Response:
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": "1h23m45s",
  "sessions": {
    "active": 3,
    "max": 10
  },
  "browsers": {
    "active": 2,
    "idle": 3
  },
  "storage": {
    "size_mb": 125
  }
}
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8020
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health
    port: 8020
  initialDelaySeconds: 5
  periodSeconds: 10
```

---

## Logging

### JSON Format (Production)

```json
{
  "level": "info",
  "time": "2026-02-27T10:00:00Z",
  "msg": "session_created",
  "session_id": "abc123",
  "ip": "192.168.1.1"
}
```

### Log Files

| File | Contents |
|------|----------|
| `axon.log` | Application logs |
| `audit.log` | Action audit trail |
| `error.log` | Errors and stack traces |

### Rotate Logs

```bash
# Daily rotation, keep 7 days
./axon 2>&1 | rotate -s 7M -c "gzip"

# Or use logrotate
/etc/logrotate.d/axon
```

---

## Performance Tuning

### Increase Browser Pool

```yaml
browser:
  pool_size: 10  # Increase for more concurrent sessions
```

### Tune HTTP Server

```yaml
server:
  read_timeout: 60s      # Increase for large requests
  write_timeout: 60s       # Increase for large responses
  max_header_bytes: 1048576
  pool_max_idle_conns: 100
```

### Memory Management

```yaml
browser:
  launch_options:
    args:
      - "--disable-gpu"
      - "--single-process"  # Reduce memory per browser
      - "--js-flags=--max-old-space-size=512"
```

---

## Backup and Recovery

### Backup Data

```bash
# Stop Axon
sudo systemctl stop axon

# Backup data directory
tar -czf axon-backup-$(date +%Y%m%d).tar.gz ./data/

# Restart
sudo systemctl start axon
```

### Restore

```bash
# Stop Axon
sudo systemctl stop axon

# Restore
tar -xzf axon-backup-20260227.tar.gz

# Restart
sudo systemctl start axon
```

---

## Monitoring

### Prometheus Metrics

```yaml
# Enable metrics endpoint
server:
  metrics_enabled: true

metrics:
  enabled: true
  endpoint: "/metrics"
```

Metrics available:
- `axon_requests_total` — Total HTTP requests
- `axon_session_active` — Active sessions
- `axon_browser_pool_used` — Browser pool utilization
- `axon_action_duration_seconds` — Action latency histogram
- `axon_snapshots_total` — Snapshots generated

### Grafana Dashboard

Import `grafana/dashboard.json` for pre-built dashboard.

---

## Troubleshooting

### High Memory Usage

1. Reduce browser pool size
2. Enable headless mode
3. Clear old sessions:
   ```bash
   curl -X DELETE http://localhost:8020/api/v1/sessions/cleanup?older_than=24h
   ```

### Slow Response Times

1. Check browser pool utilization
2. Enable debug logging to identify bottlenecks
3. Profile with: `go tool pprof http://localhost:8020/debug/pprof/`

### Browser Crashes

1. Check Chrome/Chromium version
2. Add launch flags:
   ```yaml
   browser:
     launch_options:
       args:
         - "--no-sandbox"
         - "--disable-dev-shm-usage"
   ```

---

*Axon Deployment Guide v1.0 | February 2026*
