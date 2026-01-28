

# Technical Specification — Project LIFELINE
**Emergency Control Plane (Telegram Bot)**

> Dokumen ini adalah **Technical Specification resmi** untuk Project **LIFELINE**.
> Diturunkan langsung dari `PRD.md` dan menjadi **panduan implementasi development**.
> Semua keputusan teknis DI SINI mengikat implementasi.

---

## 0. Scope & Ground Rules

### 0.1 Scope
Tech spec ini mencakup:
- Arsitektur kode
- Modul & tanggung jawab
- Alur eksekusi command
- Security boundary
- File system rules
- Deployment model

### 0.2 Out of Scope
- UI/UX Telegram lanjutan
- Multi-user / role
- Metrics / Prometheus
- Auto-remediation

---

## 1. Technology Stack

### 1.1 Language
- **Golang ≥ 1.22**
  - Single binary
  - Predictable runtime
  - Strong stdlib
  - Cocok untuk emergency system

### 1.2 Telegram Library
- `github.com/go-telegram-bot-api/telegram-bot-api/v5`
  - Long polling (no webhook)
  - No inbound dependency

### 1.3 OS Assumptions
- Linux-based
- systemd available
- Docker available (optional)

---

## 2. High-Level Architecture

```
Telegram Update
      ↓
Command Router
      ↓
AuthZ & Rate Limit
      ↓
Mode Gate (Emergency / RO / Lockdown)
      ↓
Command Handler
      ↓
System Adapter (safe wrapper)
```

---

## 3. Process Model

### 3.1 Startup Flow
1. Load config (env / file)
2. Validate allowlisted user IDs
3. Validate sandbox directory
4. Start Telegram long polling
5. Enter READ-ONLY by default

### 3.2 Runtime Model
- Stateless per command
- No background scheduler (v1)
- All actions synchronous

---

## 4. Configuration

### 4.1 Environment Variables
```env
TELEGRAM_BOT_TOKEN=
ALLOWED_USER_IDS=123456789
LIFELINE_MODE=readonly
SANDBOX_ROOT=/emergency-files
MAX_FILE_MB=50
CONFIRM_TOKEN_TTL=60
RATE_LIMIT_SECONDS=3
```

---

## 5. Module Breakdown

### 5.1 `main`
- Bootstrap application
- Wire dependencies
- Start update loop

---

### 5.2 `config`
Responsibilities:
- Parse env
- Validate required fields
- Normalize values

---

### 5.3 `auth`
Responsibilities:
- Validate Telegram user ID
- Silent reject unauthorized user

---

### 5.4 `ratelimit`
Responsibilities:
- In-memory per-user rate limit
- Sliding window (simple)

---

### 5.5 `mode`
Responsibilities:
- Maintain global bot mode
- Enforce mode constraints

Modes:
- emergency
- readonly
- lockdown

---

### 5.6 `router`
Responsibilities:
- Parse command
- Match allowlisted commands
- Dispatch handler

---

### 5.7 `confirm`
Responsibilities:
- Generate confirm token
- TTL enforcement
- Bind token to command & user

---

### 5.8 `handlers`
Each handler:
- Explicit input validation
- No shell passthrough
- Deterministic output

Handlers include:
- health
- status
- resources
- diag
- logs
- restart
- cleanup
- reboot
- snapshot
- file (ls/get/apply)

---

### 5.9 `system`
Safe wrappers around:
- systemctl
- docker
- tailscale
- cloudflared

Rules:
- No raw exec
- Hardcoded binary paths
- Fixed arguments only

---

### 5.10 `filesystem`
Responsibilities:
- Path jail enforcement
- Size validation
- Read/write sandbox only

---

### 5.11 `audit`
Responsibilities:
- Append-only audit log
- Local persistence

---

## 6. Command Execution Flow (Example)

### `/restart tailscale`

1. Router matches command
2. AuthZ check
3. Rate limit check
4. Mode check (must be emergency)
5. Generate confirm token
6. Await `/confirm <token>`
7. Execute system wrapper
8. Log audit
9. Return result

---

## 7. File System Rules (Critical)

### 7.1 Path Validation
- Reject absolute path
- Reject `..`
- Resolve & verify inside sandbox root

### 7.2 File Upload
- Saved to `sandbox/inbox`
- Never executable
- Explicit `/apply` required

---

## 8. Snapshot Implementation

Snapshot MUST collect:
- `/health`
- `/status`
- `/resources`
- `/diag net`
- Last 100 lines logs

Flow:
1. Collect outputs
2. Write temp files
3. Zip archive
4. Send to Telegram
5. Cleanup temp files

---

## 9. Security Boundaries

### 9.1 Forbidden Actions
- Arbitrary shell
- Custom command args
- File execution
- Network listening

### 9.2 Trust Assumptions
- Telegram transport trusted
- Host OS trusted
- No zero-trust inside host

---

## 10. Error Handling

Principles:
- Fail closed
- No panic
- No stacktrace to user

---

## 11. Deployment Models

### 11.1 systemd (Preferred)
- Start on boot
- Restart on failure

### 11.2 Docker
- Host network
- Read-only root FS
- Sandbox mounted RW

---

## 12. Logging

### 12.1 Audit Log (Required)
```
2026-01-28T21:14Z user=123 cmd=/restart tailscale
```

### 12.2 Debug Log (Optional)
- Stdout / journald

---

## 13. Development Checklist

- [ ] Config validation
- [ ] AuthZ silent drop
- [ ] Path jail enforced
- [ ] Confirm token TTL
- [ ] Rate limit works
- [ ] Read-only default
- [ ] Audit log append-only

---

## 14. Open Technical Decisions

- DSM CLI adapter?
- IPv6 disable or detect?
- Persistent mode storage?
- Confirm token storage (memory vs file)?

---

## 15. Final Notes

> **If this spec is violated, LIFELINE becomes a backdoor.**

Implement exactly as specified.
Refactor only with explicit security review.
