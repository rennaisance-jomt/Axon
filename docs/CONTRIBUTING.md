# Axon — Contributing Guide
## How to Contribute

**Version:** 1.0  
**Date:** February 2026

---

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. By participating, you agree to:
- Be respectful and inclusive
- Use welcoming and inclusive language
- Be collaborative and constructive
- Accept constructive criticism gracefully

---

## Getting Started

### Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.22+ |
| Git | 2.0+ |
| Docker | Latest (optional) |

### Development Setup

```bash
# 1. Fork the repository
# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/axon.git
cd axon

# 3. Add upstream remote
git remote add upstream https://github.com/superclaw/axon.git

# 4. Create a branch
git checkout -b feature/your-feature-name

# 5. Install dependencies
go mod download

# 6. Run tests
go test ./...

# 7. Start development server
go run ./cmd/axon
```

---

## Development Workflow

### 1. Pick an Issue

- Check [Issues](https://github.com/superclaw/axon/issues) for open issues
- Look for `good first issue` labels for beginners
- Comment on issues to claim them

### 2. Create a Branch

```bash
# Feature branch
git checkout -b feature/my-awesome-feature

# Bug fix branch
git checkout -b fix/bug-description

# Documentation branch
git checkout -b docs/improve-something
```

### 3. Make Changes

- Write clean, readable code
- Follow Go conventions (see below)
- Add tests for new features
- Update documentation if needed

### 4. Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat` — New feature
- `fix` — Bug fix
- `docs` — Documentation
- `style` — Formatting
- `refactor` — Code restructuring
- `test` — Tests
- `chore` — Maintenance

**Examples:**
```
feat(snapshot): add depth parameter for compact snapshots
fix(ssrf): block navigation to private IP ranges
docs(api): update endpoint documentation
```

### 5. Push and Create PR

```bash
# Push your branch
git push origin feature/my-feature

# Create PR via GitHub UI
# Fill in the PR template
```

---

## Coding Standards

### Go Conventions

1. **Formatting** — Use `gofmt`:
   ```bash
   gofmt -w .
   ```

2. **Linting** — Use `golangci-lint`:
   ```bash
   golangci-lint run
   ```

3. **Import Order** (GoLand/VSCode will do this):
   - Standard library
   - Third-party packages
   - Internal packages

4. **Naming**:
   - `camelCase` for variables and functions
   - `PascalCase` for exported types and functions
   - `snake_case` for database fields

5. **Error Handling**:
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to do thing: %w", err)
   }
   
   // Bad
   if err != nil {
       fmt.Println(err)
       return nil
   }
   ```

### Project Structure

```
internal/          # Private application code
  ├── server/      # HTTP handlers
  ├── browser/     # Browser automation
  ├── security/    # Security middleware
  └── storage/     # Database operations

pkg/               # Reusable packages
cmd/               # Application entrypoints
api/               # API definitions (OpenAPI)
test/              # Test utilities
```

---

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/server/...

# Run with verbose output
go test -v ./...
```

### Writing Tests

```go
package browser_test

import (
    "testing"
    "github.com/superclaw/axon/internal/browser"
)

func TestSessionCreate(t *testing.T) {
    // Arrange
    manager := browser.NewPool()
    
    // Act
    session, err := manager.Create("test-session")
    
    // Assert
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if session.ID != "test-session" {
        t.Errorf("expected session ID 'test-session', got '%s'", session.ID)
    }
}
```

### Test Naming

- `Test<Function>` — Unit tests
- `TestIntegration<Feature>` — Integration tests
- `TestE2E<Workflow>` — End-to-end tests

---

## Documentation

### Updating Docs

1. **API Changes** — Update `API_SPEC.md` and `DATA_SCHEMAS.md`
2. **Config Changes** — Update `DEPLOYMENT.md`
3. **New Features** — Update `FEATURES.md`
4. **Code Changes** — Add godoc comments

### Doc Format

Use clear, concise language. Include:
- What it does
- Parameters
- Return values
- Examples

```go
// Navigate opens a URL in the session's browser.
// Returns error if navigation fails or SSRF check fails.
func (s *Session) Navigate(url string) error
```

---

## Pull Request Guidelines

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] Added tests for new functionality

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
```

### Review Process

1. **Automated Checks** must pass:
   - Go build
   - Tests
   - Linting (golangci-lint)
   - Security scan (gosec)

2. **Code Review** requires:
   - 1 approval from maintainer
   - All comments addressed

3. **CI/CD** will:
   - Run full test suite
   - Build binaries for all platforms
   - Run security scans

---

## Security Guidelines

### Reporting Security Issues

**DO NOT** create public issues for security vulnerabilities.

Email: security@superclaw.io

We will:
- Acknowledge within 24 hours
- Provide timeline for fix
- Credit reporters (with permission)

### Secure Coding

- Never log sensitive data (passwords, tokens)
- Validate all inputs
- Use parameterized queries
- Follow OWASP guidelines

---

## Community

### Getting Help

- [GitHub Discussions](https://github.com/superclaw/axon/discussions) — Q&A
- [Discord](https://discord.gg/superclaw) — Chat

### Recognizing Contributors

We recognize contributions in:
- README.md contributors section
- Release notes
- Discord role: "Axon Contributor"

---

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

## Quick Reference

```bash
# Development
go run ./cmd/axon                           # Run
gofmt -w .                                 # Format
golangci-lint run                          # Lint
go test -v -race ./...                     # Test with race detector

# Build
go build -o axon ./cmd/axon                # Binary
CGO_ENABLED=0 go build -o axon ./cmd/axon  # Cross-compile

# Docker
docker build -t axon .
docker run -p 8020:8020 axon
```

---

*Axon Contributing Guide v1.0 | February 2026*
