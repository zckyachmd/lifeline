# Project LIFELINE
**Emergency Control Plane for Homelab**

> **Single Source of Truth**
> Dokumen ini adalah PRD final untuk Project **LIFELINE**, konsisten dengan *decision tree* dan *command specification*.
> Dokumen ini HARUS dibaca sebelum Technical Specification dibuat.

---

## Metadata

- **Project Name:** LIFELINE
- **Category:** Emergency Control Plane
- **Primary Interface:** Telegram Bot
- **Network Model:** Outbound-only (no inbound access)
- **Target Platform:** Linux / NAS (DSM-compatible)
- **Status:** PRD Final – Approved
- **Audience:** Homelab Owner / Infra Engineer (single operator)

---

## 1. Overview

LIFELINE adalah sistem **emergency access** berbasis Telegram yang berfungsi sebagai **jalur kontrol terakhir** ketika seluruh mekanisme remote access utama (Cloudflare Zero Trust, Tailscale, SSH, DSM UI) tidak dapat diakses, sementara host masih online secara fisik.

LIFELINE **bukan alat kenyamanan**, melainkan **lifeline operasional** untuk monitoring, diagnosis, dan recovery terbatas.

---

## 2. Background & Problem Statement

### 2.1 Existing Setup
Homelab bergantung pada:
- Cloudflare Zero Trust (containerized / Docker)
- Tailscale (native DSM)

### 2.2 Core Problem
Dalam kondisi tertentu:
- Kedua tunnel dapat **down bersamaan**
- Server tetap **online (wired, reachable dari LAN)**
- Tidak ada jalur inbound atau remote access
- Recovery membutuhkan **akses onsite**

Hal ini menciptakan **single point of failure pada control plane**, bukan pada data plane.

### 2.3 Impact
- Tidak bisa melakukan restart service
- Tidak bisa mengambil log
- Tidak bisa mengetahui root cause
- Downtime memanjang hanya karena tidak ada akses

---

## 3. Product Goals

LIFELINE bertujuan untuk:

1. Menyediakan **emergency control plane** yang independen dari tunnel
2. Memberikan **visibility cepat** dari perangkat mobile
3. Memungkinkan **recovery terbatas & terkontrol**
4. Menyediakan **akses file darurat (sandboxed)**
5. Tetap **aman, eksplisit, dan minimal**

---

## 4. Non-Goals (Explicit)

LIFELINE **TIDAK** bertujuan untuk:

- Menggantikan SSH, DSM UI, atau Web UI lain
- Menjadi file manager atau file server
- Menyediakan interactive shell
- Menjadi primary atau daily access channel
- Mengelola multi-user atau role kompleks

---

## 5. Target User

- Single homelab operator
- Advanced infra-aware user
- Mengakses dari mobile (HP)
- Digunakan dalam kondisi **panic / emergency**

---

## 6. Success Metrics

| Metric | Target |
|------|--------|
| Remote tunnel recovery success | ≥ 80% |
| Mean Time To Recovery (remote) | < 5 menit |
| Unauthorized action | 0 |
| Data loss | 0 |

---

## 7. Architecture Overview

```
Telegram Client (Mobile)
        ↓
Telegram Bot API (HTTPS)
        ↓  (Outbound-only)
LIFELINE Service
        ↓
System / Docker / Sandbox Filesystem
```

### Architectural Principles
- Tidak ada inbound traffic
- Tidak membuka port
- Tidak bergantung pada tunnel
- Stateless per command
- Explicit allowlist di semua layer

---

## 8. Functional Requirements

### 8.1 Authentication & Authorization

- Bot hanya merespon Telegram User ID yang di-allowlist
- Semua request lain di-*silent drop*
- Tidak ada konsep multi-user (v1)
- Tidak ada dynamic permission

---

### 8.2 Monitoring & Visibility

Bot HARUS menyediakan informasi berikut:
- Status internet
- Status Cloudflared
- Status Tailscale
- Status Docker
- Resource usage (disk, RAM, load)
- Public IP

**Contoh command:**
- `/health`
- `/status`
- `/resources`
- `/ip`

---

### 8.3 Diagnostics

Bot HARUS mampu:
- Validasi DNS resolver
- Validasi time / NTP / clock drift
- Mengambil log terbatas per service

**Contoh command:**
- `/diag net`
- `/diag time`
- `/logs <service>`

---

### 8.4 Recovery Actions (Controlled)

Bot BOLEH melakukan:
- Restart service tertentu
- Cleanup resource (docker prune, log cleanup)
- Reboot host (last resort)

Semua recovery action:
- Wajib confirmation token
- Token single-use
- Token time-bound (TTL)

**Contoh command:**
- `/restart <service>`
- `/cleanup`
- `/reboot`

---

### 8.5 File Access (Sandboxed)

#### 8.5.1 Sandbox Root
```
/emergency-files/
├─ logs/
├─ snapshots/
├─ exports/
└─ inbox/
```

#### Rules
- Tidak boleh absolute path
- Tidak boleh path traversal (`../`)
- Tidak boleh execute file
- Ukuran file dibatasi

**Command:**
- `/ls`
- `/get <path>`
- `/apply <filename>`

---

### 8.6 Snapshot & Forensics

Bot HARUS bisa:
- Mengumpulkan kondisi sistem saat ini
- Menggabungkan ke satu archive (ZIP)
- Mengirimkan snapshot via Telegram

**Command:**
- `/snapshot`

---

### 8.7 Safety Modes

LIFELINE memiliki beberapa mode operasi:

| Mode | Behaviour |
|----|----------|
| Emergency | Full feature |
| Read-only | Monitoring only |
| Lockdown | Semua aksi destruktif disabled |

**Command:**
- `/disable-emergency`
- `/lockdown`
- `/unlock`

---

## 9. Non-Functional Requirements

### 9.1 Security
- No arbitrary command execution
- No shell passthrough
- Allowlist di semua command
- Audit log wajib untuk semua aksi

---

### 9.2 Reliability
- Bot harus survive restart
- Graceful failure
- Stateless per request

---

### 9.3 Performance
- Response command < 2 detik
- Snapshot < 30 detik
- File transfer ≤ 50 MB

---

## 10. UX Principles

- Mobile-first
- Output ringkas & terbaca
- Tidak spam
- Error jelas
- Aksi destruktif selalu double-confirm

---

## 11. Deployment Requirements

- Prefer single binary (Golang)
- Bisa dijalankan sebagai:
  - systemd service
  - Docker container (host network)
- Dependency minimal

---

## 12. Risks & Mitigations

| Risk | Mitigation |
|----|-----------|
| Bot jadi backdoor | Sandbox + confirm |
| Human error | Read-only default |
| Telegram outage | Accepted risk |
| ISP block Telegram | No workaround |

---

## 13. Open Decisions (Before Tech Spec)

- DSM CLI integration (yes/no)
- IPv6 handling strategy
- Snapshot manual vs scheduled
- Final upload size limit

---

## 14. Project Status

- ✅ Decision Tree Recovery
- ✅ Command Specification
- ✅ PRD Final
- ⏭️ Threat Model
- ⏭️ Technical Specification

---

## 15. Final Statement

> **LIFELINE bukan convenience tool.**
> **Ini jalur hidup terakhir saat semua akses lain mati.**

Dokumen ini menjadi dasar untuk seluruh desain teknis dan implementasi berikutnya.

---
# Last Updated: 2026-01-28 15:00 WIB
