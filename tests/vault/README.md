# Axon Vault Security Evaluation

Axon is fundamentally designed to execute agentic workflows securely. The BadgerVault architecture prevents raw AI agents from exfiltrating or misusing credentials. 

To cryptographically and physically prove our security boundaries, we constructed this comprehensive suite of unit and end-to-end (E2E) integration tests. These tests are standalone, open, and reproduceable.

## Directory Structure

```text
tests/vault/
├── vault_security_test.py    # Automated Python Evaluation Script
└── README.md                 # This documentation
```

## E2E Integration Test Objectives (`vault_security_test.py`)

This test operates an actual headless Chromium browser via the Axon Go engine, performing interactions seamlessly from the `axon` Python SDK. It validates the three primary security guards of the BadgerVault system.

### Test 1: Intelligent Auto-Login Detection
**Goal:** Verify that the Axon engine can semantically "understand" the accessibility tree of a generic form to map unstructured inputs to structured vault credentials.
**Method:** 
Instead of hardcoded CSS locators, the test provides zero hints. Axon's Snapshot Extractor parses the DOM, ignoring layout structures (like `input_group` wrappers), and determines pure fillable node assignments. It correctly yields `@vault:corp-admin:username` precisely attached to the actual `<input type="email">` node, simply because its placeholder matched an email pattern (e.g. `example@org.com`).

### Test 2: Physical DOM Masking (Session Replay Protection)
**Goal:** Prove that plaintext credentials are mathematically never rendered to visual pixels or replayable logs in non-password fields (such as email/username fields). 
**Method:** 
Axon dynamically intercepts the frontend rendering context. Before the engine writes the credential into the `<input type="email">` DOM node, it synchronously executes Chrome DevTools Protocol (CDP) Javascript injected directly onto the `element.Object`. It physically mutates the element's property: `type="password"`, and marks it `data-axon-masked="true"`. Screen recorders like LogRocket or simple interceptors will strictly receive `******`. 

### Test 3: Anti-Phishing Guard (Cross-Origin Protection)
**Goal:** Prove that an AI visually hallucinating—or being socially engineered into interacting with—a malicious clone site will be definitively halted by the Axon engine.
**Method:** 
The SDK navigates to a visually identical decoy login page where the form's target (`action="..."`) points to a cross-origin untrusted domain (`evil-attacker.com`). When the agent commands Axon to inject the credential via the `fill` action, the Axon Go backend statically performs domain equivalence checks utilizing eTLD+1 standards. The extraction immediately aborts, returning an HTTP `422 Unprocessable Entity` to the caller, permanently thwarting the prompt injection or hallucination.

## Go Unit Tests (Mathematical Proofs)

Alongside the Integration tests, the mathematical logic that powers these E2E defenses are mathematically validated in Go core:
- **`internal/security/vault_test.go`**: Validates `GetBaseDomain()` logic down to `publicsuffix.EffectiveTLDPlusOne` truncation, ensuring 0.00% bypass capability for domain spoofing (e.g. preventing `example.com.evil.com` from accessing `example.com` secrets).
- **`internal/browser/snapshot_test.go`**: Validates that heuristic AI prompt hints strictly map to actionable `textbox`, `email`, or `password` inputs, rather than composite token-saving structures like `input_group`. This ensures Vault Suggestions (`@vault:xxx...`) never float pointlessly in a wrapper node, forcing injection failures.

## Running the E2E Evaluation

1. Compile and start the Axon engine natively, or ensure the binary is within PATH.
2. Execute the self-contained E2E script:
   ```powershell
   $env:PYTHONPATH="python"
   python tests/vault/vault_security_test.py
   ```

*The automated tests require zero spoofing or artificial triggers.*
