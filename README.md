# DSM Telegram Bot — LIFELINE (Native DSM)

Emergency control plane for Synology DSM homelab via Telegram (long polling, exit only). Implements PRD/technical specification/threat model in `docs/`. Designed to run **natively** on DSM hosts (no containers required). Docker assets remain optional for testing only.

## Features
- Allowlist of chat IDs (single admin), silent deletion if not allowed.
- Modes: read-only (default), emergency, lockdown.
+- Rate limit of 5 requests/minute/user, audit logs can only be appended.
- DSM API client (health/utilization, list/download/upload File Station).
- Monitoring: health, status (Cloudflared Docker containers, native Tailscale, Docker daemon), resources, network/diagnostic time, public IP.
- File sandbox `/emergency-files` with inbox/upload, 50MB size limit.
- ZIP snapshots (health/status/log) with automatic cleanup.
- Controlled actions with confirmation tokens (TTL 60 seconds), double confirmation for reboot.
- Sensitive messages self-destruct after 1 hour.
- No public IP or inbound port dependencies; only HTTPS outbound to Telegram.

## Quick Start (Native DSM)
1) Copy `.env.example` to `.env` and fill in `TELEGRAM_BOT_TOKEN`, `ALLOWED_USER_IDS`, `DSM_*`.
2) Adjust `configs/config.yaml` if necessary (sandbox root, TTL, mode).
3) Ensure the sandbox directory exists on the host: `/emergency-files/{inbox,snapshots}` with write permissions for the runtime user.
4) Building the binary: `make build`
5) Running locally: `make run` (use `configs/config.yaml` by default) or `CONFIG_PATH=/path/custom make run`
6) Installing to the system: `sudo make install` (binary to `/usr/local/bin`, default configuration to `/etc/lifeline/config.yaml` if it doesn't already exist)
7) Installing systemd: `sudo make install-service` (use `configs/lifeline.service`, enable & start)

## Command Guide (UX)
- Reading/Monitoring: `/health`, `/status`, `/resources`, `/ip`, `/diag net|time`, `/logs <cloudflared|tailscale|docker>`
- Files: `/ls [path]`, `/get <path>`, send any documents for upload to `inbox/`, `/snapshot`
- Actions (emergency mode + confirmation): `/restart <cloudflared|tailscale|docker>`, `/cleanup`, `/apply <filename>`, `/reboot` (double confirmation)
- Security & Mode: `/lockdown`, `/unlock`, `/disable-emergency`, `/mode`, `/help`, `/confirm <token>`

## Security Notes
- No inbound ports; Telegram long polling only.
- Allowed commands: restart/logs is restricted to `cloudflared` (docker container), `tailscale` (native service), and `docker` daemon.
- Path restrictions enforce the sandbox root; deny absolute /`..`.
- Audit logs in `<sandbox>/audit.log` (best effort append only).
- The health check server only binds to `127.0.0.1:8080` (for local monitoring).

## Testing
Run `make test` (rate limiting, token confirmation, path limiting). Expand before production.

## Deployment Checklist
- Tokens and chat IDs are provided via env, never committed.
- Sandbox exists and is writable.
- DSM API token rotation interval is configured (default 24 hours).
- Disable emergency mode after use: `/disable-emergency`.
