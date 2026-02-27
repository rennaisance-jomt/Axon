# Axon — Data Schemas
## JSON Schema Definitions

**Version:** 1.0  
**Date:** February 2026

---

## Session Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Session",
  "type": "object",
  "properties": {
    "session_id": {
      "type": "string",
      "description": "Unique session identifier",
      "pattern": "^[a-zA-Z0-9_-]{1,64}$"
    },
    "status": {
      "type": "string",
      "enum": ["created", "active", "idle", "closed"],
      "description": "Current session state"
    },
    "profile": {
      "type": "string",
      "description": "Path to profile file (cookies/auth)"
    },
    "created_at": {
      "type": "string",
      "format": "date-time",
      "description": "Session creation timestamp"
    },
    "last_action": {
      "type": "string",
      "format": "date-time",
      "description": "Last action timestamp"
    },
    "url": {
      "type": "string",
      "format": "uri",
      "description": "Current page URL"
    },
    "title": {
      "type": "string",
      "description": "Current page title"
    },
    "auth_state": {
      "type": "string",
      "enum": ["unknown", "logged_out", "logged_in", "error"],
      "description": "Authentication state"
    },
    "page_state": {
      "type": "string",
      "enum": ["unknown", "loading", "ready", "error", "captcha", "rate_limited"],
      "description": "Page load state"
    },
    "browser_context": {
      "type": "string",
      "description": "Internal browser context ID"
    }
  },
  "required": ["session_id", "status", "created_at"]
}
```

---

## Element Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "PageElement",
  "type": "object",
  "properties": {
    "ref": {
      "type": "string",
      "description": "Unique element reference (e.g., 'e1', 'a1')"
    },
    "type": {
      "type": "string",
      "enum": ["button", "textbox", "link", "checkbox", "radio", "select", "heading", "paragraph", "image", "list", "form", "container"],
      "description": "Element type"
    },
    "label": {
      "type": "string",
      "description": "Visible label or text content"
    },
    "placeholder": {
      "type": "string",
      "description": "Placeholder text"
    },
    "intent": {
      "type": "string",
      "description": "Classified intent (e.g., 'auth.login', 'social.publish')",
      "pattern": "^[a-z_]+\\.[a-z_]+$"
    },
    "role": {
      "type": "string",
      "description": "ARIA role"
    },
    "selectors": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Stable CSS selectors"
    },
    "reversible": {
      "type": "string",
      "enum": ["read", "write_reversible", "write_irreversible", "sensitive_write"],
      "description": "Action reversibility classification"
    },
    "visible": {
      "type": "boolean",
      "description": "Whether element is visible"
    },
    "enabled": {
      "type": "boolean",
      "description": "Whether element is enabled"
    }
  },
  "required": ["ref", "type", "label"]
}
```

---

## Snapshot Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Snapshot",
  "type": "object",
  "properties": {
    "session_id": {
      "type": "string"
    },
    "url": {
      "type": "string",
      "format": "uri"
    },
    "title": {
      "type": "string"
    },
    "state": {
      "type": "string",
      "enum": ["unknown", "loading", "ready", "error", "captcha", "rate_limited", "logged_in", "logged_out"]
    },
    "depth": {
      "type": "string",
      "enum": ["compact", "standard", "full"]
    },
    "content": {
      "type": "string",
      "description": "Formatted text representation"
    },
    "elements": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/PageElement"
      }
    },
    "warnings": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/Warning"
      }
    },
    "timestamp": {
      "type": "string",
      "format": "date-time"
    },
    "token_count": {
      "type": "integer",
      "description": "Estimated token count for LLM"
    }
  },
  "required": ["session_id", "url", "state", "content"]
}
```

---

## Action Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Action",
  "type": "object",
  "properties": {
    "session_id": {
      "type": "string"
    },
    "ref": {
      "type": "string",
      "description": "Element reference"
    },
    "action": {
      "type": "string",
      "enum": ["click", "fill", "press", "select", "hover", "scroll", "double_click", "right_click"]
    },
    "value": {
      "type": "string",
      "description": "Value for fill/press/select"
    },
    "confirm": {
      "type": "boolean",
      "default": false,
      "description": "Required for irreversible actions"
    },
    "intent": {
      "type": "string",
      "description": "Classified intent of action"
    },
    "reversible": {
      "type": "string",
      "enum": ["read", "write_reversible", "write_irreversible", "sensitive_write"]
    }
  },
  "required": ["session_id", "ref", "action"]
}
```

---

## Action Result Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ActionResult",
  "type": "object",
  "properties": {
    "success": {
      "type": "boolean"
    },
    "result": {
      "type": "string",
      "description": "Human-readable result message"
    },
    "requires_confirm": {
      "type": "boolean",
      "description": "Whether irreversible action needs confirmation"
    },
    "message": {
      "type": "string",
      "description": "Message to display to agent"
    },
    "error_type": {
      "type": "string",
      "enum": [
        "element_not_found",
        "navigation_failed",
        "timeout",
        "captcha",
        "rate_limited",
        "auth_required",
        "ssrf_blocked",
        "irreversible_unconfirmed",
        "injection_warning",
        "session_not_found"
      ]
    },
    "suggestion": {
      "type": "string",
      "description": "Suggested action to recover"
    },
    "recoverable": {
      "type": "boolean"
    }
  },
  "required": ["success"]
}
```

---

## Warning Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Warning",
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "enum": [
        "prompt_injection_suspected",
        "ssrf_attempt",
        "untrusted_content",
        "irreversible_action",
        "rate_limit_warning",
        "captcha_detected",
        "auth_expired"
      ]
    },
    "severity": {
      "type": "string",
      "enum": ["low", "medium", "high", "critical"]
    },
    "message": {
      "type": "string"
    },
    "location": {
      "type": "string",
      "description": "Where the warning was detected"
    },
    "raw": {
      "type": "string",
      "description": "Raw content that triggered warning (for debugging)"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time"
    }
  },
  "required": ["type", "severity", "message"]
}
```

---

## Audit Log Entry Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "AuditLogEntry",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Unique entry ID"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time"
    },
    "session_id": {
      "type": "string"
    },
    "agent_id": {
      "type": "string",
      "description": "ID of agent that performed action"
    },
    "action": {
      "type": "string",
      "description": "Action type (navigate, snapshot, act, etc.)"
    },
    "target_ref": {
      "type": "string",
      "description": "Element reference if applicable"
    },
    "target_intent": {
      "type": "string",
      "description": "Classified intent of target element"
    },
    "domain": {
      "type": "string",
      "description": "Domain action was performed on"
    },
    "reversibility": {
      "type": "string",
      "enum": ["read", "write_reversible", "write_irreversible", "sensitive_write"]
    },
    "confirmed_by": {
      "type": "string",
      "description": "What confirmed irreversible action (agent/user)"
    },
    "parameters": {
      "type": "object",
      "description": "Action parameters (secrets redacted)"
    },
    "result": {
      "type": "string",
      "enum": ["success", "failed", "blocked", "requires_confirm"]
    },
    "warnings": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/Warning"
      }
    },
    "prev_hash": {
      "type": "string",
      "description": "Previous log entry hash (for chain)"
    },
    "this_hash": {
      "type": "string",
      "description": "Hash of this entry"
    }
  },
  "required": ["id", "timestamp", "session_id", "action", "result"]
}
```

---

## Element Memory Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ElementMemory",
  "type": "object",
  "properties": {
    "domain": {
      "type": "string",
      "description": "Website domain"
    },
    "path": {
      "type": "string",
      "description": "URL path pattern"
    },
    "element": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string",
          "description": "Friendly name (e.g., 'compose_box')"
        },
        "intent": {
          "type": "string"
        },
        "selectors": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "required": ["name", "intent", "selectors"]
    },
    "visit_count": {
      "type": "integer"
    },
    "last_visited": {
      "type": "string",
      "format": "date-time"
    },
    "last_matched": {
      "type": "string",
      "format": "date-time"
    },
    "match_success_rate": {
      "type": "number",
      "minimum": 0,
      "maximum": 1
    }
  },
  "required": ["domain", "element"]
}
```

---

## Configuration Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "AxonConfig",
  "type": "object",
  "properties": {
    "server": {
      "type": "object",
      "properties": {
        "host": {
          "type": "string",
          "default": "0.0.0.0"
        },
        "port": {
          "type": "integer",
          "default": 8020
        },
        "read_timeout": {
          "type": "string"
        },
        "write_timeout": {
          "type": "string"
        }
      }
    },
    "browser": {
      "type": "object",
      "properties": {
        "headless": {
          "type": "boolean",
          "default": true
        },
        "binary_path": {
          "type": "string"
        },
        "pool_size": {
          "type": "integer",
          "default": 5
        },
        "launch_options": {
          "type": "object"
        }
      }
    },
    "security": {
      "type": "object",
      "properties": {
        "ssrf": {
          "type": "object",
          "properties": {
            "enabled": {
              "type": "boolean",
              "default": true
            },
            "allow_private_network": {
              "type": "boolean",
              "default": false
            },
            "domain_allowlist": {
              "type": "array",
              "items": {
                "type": "string"
              }
            },
            "domain_denylist": {
              "type": "array",
              "items": {
                "type": "string"
              }
            }
          }
        },
        "prompt_injection": {
          "type": "object",
          "properties": {
            "enabled": {
              "type": "boolean",
              "default": true
            },
            "mode": {
              "type": "string",
              "enum": ["warn", "strip", "block"],
              "default": "warn"
            },
            "sensitivity": {
              "type": "string",
              "enum": ["low", "medium", "high"],
              "default": "medium"
            }
          }
        }
      }
    },
    "storage": {
      "type": "object",
      "properties": {
        "path": {
          "type": "string"
        },
        "session_ttl": {
          "type": "string"
        },
        "audit_retention": {
          "type": "string"
        }
      }
    }
  }
}
```

---

## Cookie Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Cookie",
  "type": "object",
  "properties": {
    "name": {
      "type": "string"
    },
    "value": {
      "type": "string"
    },
    "domain": {
      "type": "string"
    },
    "path": {
      "type": "string",
      "default": "/"
    },
    "expires": {
      "type": "string",
      "format": "date-time"
    },
    "http_only": {
      "type": "boolean"
    },
    "secure": {
      "type": "boolean"
    },
    "same_site": {
      "type": "string",
      "enum": ["Strict", "Lax", "None"]
    }
  },
  "required": ["name", "value", "domain"]
}
```

---

## Profile Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SessionProfile",
  "type": "object",
  "properties": {
    "name": {
      "type": "string"
    },
    "domain": {
      "type": "string"
    },
    "cookies": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/Cookie"
      }
    },
    "local_storage": {
      "type": "object"
    },
    "created_at": {
      "type": "string",
      "format": "date-time"
    },
    "last_used": {
      "type": "string",
      "format": "date-time"
    }
  },
  "required": ["name", "domain"]
}
```

---

*Axon Data Schemas v1.0 | February 2026*
