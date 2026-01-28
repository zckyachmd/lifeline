# LIFELINE Command UX

Semua perintah via Telegram chat (admin-only, silent drop user lain). Bot berjalan native di DSM, outbound-only.

## Monitoring & Diagnostics
- `/health` — ringkas health DSM + resources.
- `/status` — status cloudflared (container), tailscale (native), docker daemon.
- `/resources` — CPU/mem/disk ringkas.
- `/ip` — public IP lookup (outbound).
- `/diag net` — ping 1.1.1.1 (latency cepat).
- `/diag time` — `timedatectl` untuk sinkron waktu.
- `/logs <cloudflared|tailscale|docker>` — tail log layanan (cloudflared via `docker logs`).

## Files (sandbox `/emergency-files`)
- `/ls [path]` — list isi direktori relatif sandbox.
- `/get <path>` — kirim file (<=50MB).
- Upload dokumen — otomatis disimpan ke `inbox/` (dibatasi 50MB).
- `/snapshot` — kumpulkan health/status/logs ke ZIP dan kirim, auto-clean.

## Recovery Actions (emergency mode + token)
- `/restart <cloudflared|tailscale|docker>` — restart layanan (cloudflared lewat docker restart, tailscale & docker via systemctl).
- `/cleanup` — `docker system prune -f` (confirm token).
- `/apply <filename>` — pindahkan file dari `inbox/` ke root sandbox (confirm token).
- `/reboot` — reboot host (double confirm).

## Safety & Modes
- `/mode` — tampilkan mode aktif.
- `/lockdown` — disable aksi destruktif, hanya /unlock.
- `/unlock` — kembali ke readonly.
- `/disable-emergency` — set readonly.
- `/confirm <token>` — eksekusi aksi yang menunggu konfirmasi.
- `/help` — ringkasan singkat perintah.

## UX Catatan
- Respons sensitif (log, reboot) auto-delete setelah 1 jam.
- Rate limit 5 req/menit per user; kalau kena limit balas singkat.
- Semua perintah mengaudit ke `<sandbox>/audit.log` (append-only best effort).
- Tidak ada dependency IP publik; hanya outbound HTTPS ke Telegram + API yang dibutuhkan.
