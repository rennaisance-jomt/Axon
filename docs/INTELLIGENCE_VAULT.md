# Secure Credential Management (Vault)

Axon's credential vault is a secure way to manage passwords and logins for your AI agents. It ensures that your agents can log in to websites without ever seeing your actual plain-text passwords.

## The Problem: Password Security
When an AI agent uses a standard browser, it reads the code of the webpage. If you use the agent to log in:
1. The agent sees the password value to type it in.
2. The agent might "leak" the password in logs or chat sessions.
3. The agent could potentially be tricked by a malicious website into entering your credentials on the wrong domain.

## The Solution: Axon's Secure Vault
Axon's vault sits between your agent and the browser to keep credentials safe.

### 1. Hidden Injection
Agents use a reference (a simple name) instead of the real password. For example:
`@vault:github_login:password`

When Axon sees this, it retrieves the real password from its secure internal database and types it directly into the website's password field. The actual password never travels back to the AI model or shows up in logs.

### 2. Domain Binding (Anti-Phishing)
Every login stored in the vault is linked to a specific website domain.
- A password for `github.com` will **only** be typed if the browser is actually on `github.com`.
- If a malicious site tries to trick the agent, Axon will block the injection.

### 3. Masking
When credentials are typed into a field, Axon automatically hides that part of the screen in its visual snapshots and logs. They appear as `******` to anyone watching the process.

### 4. Encryption at Rest
All credentials are stored in an encrypted database on your local machine using standard security patterns (AES-256).

---

## Usage Guide

### 1. Management via CLI
Use the command line to add or remove logins.

```bash
# Add a login for a specific website
axon vault add github https://github.com --user "myname" --pass "mypassword123"

# List your stored login names (metadata only)
axon vault list

# Delete a stored login
axon vault delete github
```

### 2. Filling a Field in the Browser
You can tell Axon to fill a login field from the CLI too:

```bash
# Fill a field by ID using your 'github' login
axon vault fill mysession github --ref e1 --field password

# Fill a field using a search description
axon vault fill mysession github --intent "password login field"
```

### 3. Using the Python SDK
```python
from axon.client import Axon

axon = Axon()

# Add a login
await axon.add_secret(
    name="my_creds",
    value="secret_val",
    url="https://example.com",
    username="admin",
    password="password123"
)

# Use it in a browser session
await axon.vault_fill(
    session_id="session123",
    ref="e5",
    secret_name="my_creds",
    field="password"
)
```

---

## Security Technical Details
- **Storage**: Local encrypted database (AES-256-GCM).
- **Isolation**: Each login is locked to a specific domain.
- **Reporting**: Unauthorized use attempts are logged as security errors.
