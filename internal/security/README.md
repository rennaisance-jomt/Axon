# Axon Internal: Security Module

The `internal/security` package implements the multi-tier defense architecture for Axon, assuming the browser is a high-risk environment.

## 🛡️ Multi-Tier Defense Layers

### 1. **SSRF Guard** (`ssrf.go`)
Protects against Server-Side Request Forgery.
*   **Validation**: Every `Navigate` request is screened against a denylist of private/internal networks (e.g., `169.254.169.254`, `10.0.0.0/8`).
*   **Scheme Allowlist**: Enforces `https` and `http` by default.
*   **Domain Controls**: User-configurable allowlists and denylists.

### 2. **Prompt Injection Guard** (`prompt_injection.go`)
Defends against malicious web content manipulating the agent's instructions.
*   **Detection**: Scans extracted page content for known injection patterns (e.g., "ignore all previous instructions").
*   **Modes**: Configured via `config.yaml` as `warn`, `strip`, or `block`.

### 3. **Action Reversibility Classifier** (`reversibility.go`)
Native protection against high-risk agent behavior.
*   **Classification**: Every action (`Click`, `Fill`, `Press`) is classified into three categories:
    1.  **Read**: Purely informational/navigation.
    2.  **Write Reversible**: Data entry that can be undone (e.g., typing in a text field).
    3.  **Write Irreversible**: Actions with permanent consequences (e.g., "Submit Order," "Delete Account").
*   **Human-in-the-loop**: High-risk actions automatically trigger a `RequiresConfirm` response unless the API call includes `confirm=true`.

### 4. **Audit Logger** (`audit.go`)
Cryptographically chained, non-repudiable audit logs.
*   **Chain of Trust**: Every action is SHA-256 hashed with the hash of the previous event, creating a tamper-proof chain.
*   **Persistence**: Handled via `internal/storage` (BadgerDB).
*   **Verifiable**: A `VerifyChain()` method allows programmatic proof that the logs have not been altered.

## 🚀 Security Configuration

```yaml
security:
  ssrf:
    enabled: true
    allow_private_network: false
  prompt_injection:
    enabled: true
    mode: "warn"
  reversibility:
    require_confirm: true
```

---
*Axon Security Module v1.0 | February 2026*
