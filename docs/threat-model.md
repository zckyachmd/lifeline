

# Threat Model — Project LIFELINE
**Telegram-based Emergency Control Plane**

> Dokumen ini mendefinisikan **threat model resmi** untuk Project **LIFELINE**.
> Dokumen ini **WAJIB** menjadi referensi sebelum implementasi dan setiap perubahan Tech Spec.

---

## 1. Scope & Assumptions

### 1.1 Scope
Threat model ini mencakup:
- Telegram Bot (LIFELINE service)
- Host OS (Linux / NAS / DSM)
- Docker runtime & service adapter
- File system sandbox
- Network outbound dependency (Telegram)

### 1.2 Assumptions
- Single administrator (owner-operated)
- Telegram dianggap **trusted transport**
- Host OS dianggap **trusted baseline**
- Tidak ada zero-trust model di dalam host
- Bot memiliki privilege tinggi (blast radius besar)

---

## 2. Assets to Protect

| Asset | Description |
|-----|------------|
| Host OS | Integritas & availability sistem |
| Docker & Services | Cloudflared, Tailscale, container lain |
| Network Access | Akses keluar & kontrol tunnel |
| File System | Konfigurasi, log, snapshot |
| Credentials | Token bot, tunnel credentials |
| Availability | Kemampuan recovery saat emergency |

---

## 3. Trust Boundaries

```
Telegram Cloud (trusted transport)
        ↓
LIFELINE Bot Process (high privilege)
        ↓
Host OS / Docker / File System
```

**Catatan penting:**
Jika bot dikompromikan, **host sepenuhnya terkompromikan**.

---

## 4. Primary Threat Table

| ID | Threat | Attack Vector | Impact | Likelihood | Mitigation (MANDATORY) |
|----|--------|--------------|--------|------------|------------------------|
| T1 | Bot token leak | Token bocor via repo/log | Full remote takeover | Medium | Token via env only, never logged, rotatable |
| T2 | Unauthorized user | Telegram ID spoof / forward | Full control | Low | Hard allowlist + silent drop |
| T3 | Path traversal | `../` pada file command | Arbitrary file read | Medium | Path jail + clean & root verify |
| T4 | Arbitrary command exec | Shell passthrough | RCE | High | No shell, fixed system wrappers |
| T5 | Accidental reboot | Human error | Downtime | High | Confirm token + TTL |
| T6 | Replay attack | Reuse confirm token | Repeat destructive action | Medium | Single-use token + TTL |
| T7 | Log poisoning | Crafted user input | Audit confusion | Low | Controlled output, no raw echo |
| T8 | Resource exhaustion | Snapshot spam / flood | Bot crash | Medium | Rate limit + size cap |
| T9 | Privilege escalation | Bot runs as root | Host compromise | Medium | Minimal permission + wrapper |
| T10 | Telegram outage | API unavailable | No recovery | Low | Accepted risk |

---

## 5. Abuse Scenarios (Human Error)

| ID | Scenario | Risk | Mitigation |
|----|----------|------|------------|
| A1 | Reboot saat disk penuh | Data corruption | Snapshot sebelum reboot |
| A2 | Restart service salah | Downtime | Service allowlist |
| A3 | Upload config salah | Service failure | Manual `/apply` + confirm |
| A4 | Panic command spam | Resource exhaustion | Rate limit |
| A5 | Lupa disable emergency | Persistent high privilege | `/disable-emergency` |

---

## 6. File System Threats

| Threat | Description | Mitigation |
|------|------------|------------|
| Arbitrary read | Akses `/etc/passwd` | Sandbox root only |
| Arbitrary write | Overwrite system file | Inbox-only + no exec |
| Binary upload | Malware upload | Reject executable bit |
| Zip bomb | Snapshot abuse | Size & time limit |

---

## 7. Command Risk Classification

| Command | Risk Level | Required Safeguard |
|------|-----------|--------------------|
| `/health` | Low | Read-only |
| `/status` | Low | Read-only |
| `/logs` | Medium | Line limit |
| `/snapshot` | Medium | Rate limit |
| `/restart` | High | Confirm token |
| `/cleanup` | High | Confirm token |
| `/apply` | High | Confirm token |
| `/reboot` | Critical | Double confirm + snapshot |

---

## 8. System Adapter Threats

| Adapter | Threat | Mitigation |
|-------|--------|------------|
| systemctl | Exec abuse | Hardcoded args only |
| docker | Destructive prune | Scope-limited wrapper |
| tailscale | Auth reset misuse | No auto-auth |
| cloudflared | Cert misuse | Restart-only |

---

## 9. Design Decisions Enforced by Threat Model

| Decision | Rationale |
|--------|-----------|
| No webhook | Eliminate inbound attack surface |
| Long polling | Simpler trust boundary |
| No shell access | Remove RCE class entirely |
| Read-only default | Minimize human error |
| Single admin | Reduce auth complexity |
| No auto-remediation | Avoid cascading failures |

---

## 10. Residual Risks (Accepted)

| Risk | Reason |
|----|------|
| Telegram outage | No alternative OOB channel |
| Physical access attack | Out of scope |
| Insider threat (owner) | Accepted |
| Host root compromise | Out of scope |

---

## 11. Security Review Checklist (Release Gate)

- [ ] Bot token not hardcoded
- [ ] Allowlist enforced
- [ ] Path jail tested
- [ ] Confirm token TTL enforced
- [ ] Replay blocked
- [ ] Rate limit effective
- [ ] Audit log append-only
- [ ] Read-only default active

---

## 12. Final Statement

> **LIFELINE bukan zero-trust system.**
> **Ini adalah controlled blast-radius system.**

Jika threat model ini dilanggar:
- LIFELINE berubah menjadi **backdoor**
- PRD dan Tech Spec dianggap **tidak valid**

Dokumen ini mengikat seluruh implementasi dan perubahan desain berikutnya.
