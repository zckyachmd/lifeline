package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"

	"zckyachmd/lifeline/internal/auth"
	"zckyachmd/lifeline/internal/mode"
	"zckyachmd/lifeline/internal/security/audit"
	"zckyachmd/lifeline/internal/security/confirm"
	rl "zckyachmd/lifeline/internal/security/ratelimit"
	"zckyachmd/lifeline/internal/services"
)

// Bot wires Telegram updates with services.
type Bot struct {
	api        *tgbotapi.BotAPI
	auth       *auth.Authorizer
	limiter    *rl.Limiter
	confirm    *confirm.Manager
	modes      *mode.Manager
	audit      *audit.Logger
	monitor    *services.MonitoringService
	files      *services.FileService
	system     *services.SystemService
	snapshot   *services.SnapshotService
	logger     zerolog.Logger
	sandbox    string
	confirmTTL time.Duration
	pollWait   int
}

// New constructs bot handler.
func New(api *tgbotapi.BotAPI, authz *auth.Authorizer, limiter *rl.Limiter, confirmMgr *confirm.Manager, modes *mode.Manager, auditLog *audit.Logger, monitor *services.MonitoringService, files *services.FileService, sys *services.SystemService, snap *services.SnapshotService, logger zerolog.Logger, sandbox string, confirmTTL time.Duration, pollWait int) *Bot {
	return &Bot{
		api:        api,
		auth:       authz,
		limiter:    limiter,
		confirm:    confirmMgr,
		modes:      modes,
		audit:      auditLog,
		monitor:    monitor,
		files:      files,
		system:     sys,
		snapshot:   snap,
		logger:     logger,
		sandbox:    sandbox,
		confirmTTL: confirmTTL,
		pollWait:   pollWait,
	}
}

// Start begins polling loop.
func (b *Bot) Start(ctx context.Context) error {
	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = b.pollWait
	updates := b.api.GetUpdatesChan(ucfg)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if update.Message != nil {
				b.handleMessageSafe(ctx, update.Message)
			}
		}
	}
}

// handleMessageSafe ensures panics are recovered and logged.
func (b *Bot) handleMessageSafe(ctx context.Context, m *tgbotapi.Message) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error().Interface("panic", r).Msg("panic in handler")
			if m != nil && m.From != nil {
				b.audit.Write(m.From.ID, "panic", "error", map[string]string{"text": m.Text})
			}
		}
	}()
	b.handleMessage(ctx, m)
}

func (b *Bot) handleMessage(ctx context.Context, m *tgbotapi.Message) {
	if m == nil || m.From == nil {
		return
	}
	userID := m.From.ID
	if !b.auth.IsAllowed(userID) {
		return // silent drop
	}
	if !b.limiter.Allow(userID) {
		b.reply(m.Chat.ID, "Rate limit exceeded. Slow down (5/min).", 0)
		b.audit.Write(userID, "ratelimit", "deny", map[string]string{"text": m.Text})
		return
	}

	if m.Document != nil {
		b.handleUpload(ctx, m)
		return
	}

	if m.IsCommand() {
		b.routeCommand(ctx, m)
	}
}

func (b *Bot) routeCommand(ctx context.Context, m *tgbotapi.Message) {
	cmd := strings.ToLower(m.Command())
	args := strings.Fields(m.CommandArguments())

	if cmd == "confirm" {
		b.handleConfirm(ctx, m, args)
		return
	}

	// mode enforcement for lockdown
	if b.modes.Current() == mode.Lockdown {
		// only unlock allowed
		if cmd != "unlock" {
			b.reply(m.Chat.ID, "Bot is in lockdown. Only /unlock allowed.", 0)
			b.audit.Write(m.From.ID, "/"+cmd, "deny", nil)
			return
		}
	}

	switch cmd {
	case "start", "help":
		b.reply(m.Chat.ID, helpText(), 0)
	case "health":
		out, err := b.monitor.Health(ctx)
		b.respond(m, cmd, out, err, false)
	case "status":
		out, err := b.monitor.Status(ctx)
		b.respond(m, cmd, out, err, false)
	case "resources":
		out, err := b.monitor.Resources(ctx)
		b.respond(m, cmd, out, err, false)
	case "ip":
		out, err := b.monitor.PublicIP(ctx)
		b.respond(m, cmd, out, err, false)
	case "diag":
		if len(args) == 0 {
			b.reply(m.Chat.ID, "Usage: /diag <net|time>", 0)
			return
		}
		switch args[0] {
		case "net":
			out, err := b.monitor.DiagNet(ctx)
			b.respond(m, cmd, out, err, false)
		case "time":
			out, err := b.monitor.DiagTime(ctx)
			b.respond(m, cmd, out, err, false)
		default:
			b.reply(m.Chat.ID, "Unknown diag target", 0)
		}
	case "logs":
		if len(args) == 0 {
			b.reply(m.Chat.ID, "Usage: /logs <service>", 0)
			return
		}
		if !b.system.IsAllowedService(args[0]) {
			b.reply(m.Chat.ID, "Service not allowed", 0)
			return
		}
		out, err := b.system.TailLogs(ctx, args[0], 100)
		b.respond(m, cmd, out, err, true)
	case "ls":
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		list, err := b.files.List(path)
		if err != nil {
			b.respond(m, cmd, "", err, false)
			return
		}
		b.reply(m.Chat.ID, strings.Join(list, "\n"), 0)
		b.audit.Write(m.From.ID, "/ls", "ok", map[string]string{"path": path})
	case "get":
		if len(args) == 0 {
			b.reply(m.Chat.ID, "Usage: /get <path>", 0)
			return
		}
		f, _, err := b.files.Read(args[0])
		if err != nil {
			b.respond(m, cmd, "", err, false)
			return
		}
		defer f.Close()
		doc := tgbotapi.NewDocument(m.Chat.ID, tgbotapi.FileReader{Name: filepath.Base(args[0]), Reader: f})
		if _, err := b.api.Send(doc); err != nil {
			b.respond(m, cmd, "", err, false)
			return
		}
		b.audit.Write(m.From.ID, "/get", "ok", map[string]string{"path": args[0]})
	case "snapshot":
		buf, err := b.snapshot.Build(ctx)
		if err != nil {
			b.respond(m, cmd, "", err, false)
			return
		}
		filePath, err := b.snapshot.Save(buf, filepath.Join(b.sandbox, "snapshots"))
		if err != nil {
			b.respond(m, cmd, "", err, false)
			return
		}
		f, err := os.Open(filePath)
		if err != nil {
			b.respond(m, cmd, "", err, false)
			return
		}
		defer f.Close()
		doc := tgbotapi.NewDocument(m.Chat.ID, tgbotapi.FileReader{Name: filepath.Base(filePath), Reader: f})
		if _, err := b.api.Send(doc); err != nil {
			b.respond(m, cmd, "", err, false)
			return
		}
		b.audit.Write(m.From.ID, "/snapshot", "ok", nil)
		go func() { _ = os.Remove(filePath) }()
	case "restart":
		if !b.requireMode(m, mode.Emergency) {
			return
		}
		if len(args) == 0 {
			b.reply(m.Chat.ID, "Usage: /restart <service>", 0)
			return
		}
		b.issueConfirm(m, cmd, args, false)
	case "cleanup":
		if !b.requireMode(m, mode.Emergency) {
			return
		}
		b.issueConfirm(m, cmd, args, false)
	case "reboot":
		if !b.requireMode(m, mode.Emergency) {
			return
		}
		b.issueConfirm(m, cmd, args, true)
	case "apply":
		if !b.requireMode(m, mode.Emergency) {
			return
		}
		if len(args) == 0 {
			b.reply(m.Chat.ID, "Usage: /apply <filename>", 0)
			return
		}
		b.issueConfirm(m, cmd, args, false)
	case "lockdown":
		b.modes.Set(mode.Lockdown)
		b.reply(m.Chat.ID, "Lockdown enabled. Destructive commands disabled.", 0)
		b.audit.Write(m.From.ID, "/lockdown", "ok", nil)
	case "unlock":
		b.modes.Set(mode.ReadOnly)
		b.reply(m.Chat.ID, "Lockdown lifted. Mode=readonly.", 0)
		b.audit.Write(m.From.ID, "/unlock", "ok", nil)
	case "disable-emergency":
		b.modes.Set(mode.ReadOnly)
		b.reply(m.Chat.ID, "Emergency mode disabled. Mode=readonly.", 0)
		b.audit.Write(m.From.ID, "/disable-emergency", "ok", nil)
	case "mode":
		b.reply(m.Chat.ID, fmt.Sprintf("Current mode: %s", b.modes.Current()), 0)
	default:
		b.reply(m.Chat.ID, "Unknown command. Use /help", 0)
	}
}

func (b *Bot) handleUpload(ctx context.Context, m *tgbotapi.Message) {
	file := m.Document
	if file.FileSize > int(b.files.MaxBytes()) {
		b.reply(m.Chat.ID, "File too large", 0)
		return
	}
	fileCfg := tgbotapi.FileConfig{FileID: file.FileID}
	tgFile, err := b.api.GetFile(fileCfg)
	if err != nil {
		b.reply(m.Chat.ID, "Failed to fetch file", 0)
		return
	}
	url := tgFile.Link(b.api.Token)
	resp, err := http.Get(url)
	if err != nil {
		b.reply(m.Chat.ID, "Download error", 0)
		return
	}
	defer resp.Body.Close()
	savedPath, err := b.files.Save(file.FileName, resp.Body)
	if err != nil {
		b.reply(m.Chat.ID, fmt.Sprintf("Save failed: %v", err), 0)
		return
	}
	b.reply(m.Chat.ID, fmt.Sprintf("File stored in inbox: %s", savedPath), 0)
	b.audit.Write(m.From.ID, "upload", "ok", map[string]string{"file": file.FileName})
}

func (b *Bot) issueConfirm(m *tgbotapi.Message, cmd string, args []string, double bool) {
	token, pa := b.confirm.Issue(m.From.ID, cmd, args, double)
	b.reply(m.Chat.ID, fmt.Sprintf("Confirm with /confirm %s (ttl %s)", token, b.confirmTTL), 0)
	b.audit.Write(m.From.ID, "/"+cmd, "pending", map[string]string{"token": token, "args": strings.Join(args, ",")})
	_ = pa
}

func (b *Bot) handleConfirm(ctx context.Context, m *tgbotapi.Message, args []string) {
	if len(args) == 0 {
		b.reply(m.Chat.ID, "Usage: /confirm <token>", 0)
		return
	}
	pa, err := b.confirm.Consume(m.From.ID, args[0])
	if err != nil {
		b.reply(m.Chat.ID, "Invalid/expired token", 0)
		return
	}
	// double-confirm flow: issue new token instead of executing
	if pa.Double {
		token, _ := b.confirm.Issue(m.From.ID, pa.Command, pa.Args, false)
		b.reply(m.Chat.ID, fmt.Sprintf("Second confirmation required: /confirm %s", token), 0)
		return
	}

	switch pa.Command {
	case "restart":
		out, err := b.system.RestartService(ctx, first(pa.Args))
		b.respond(m, pa.Command, out, err, false)
	case "cleanup":
		out, err := b.system.Cleanup(ctx)
		b.respond(m, pa.Command, out, err, false)
	case "reboot":
		out, err := b.system.Reboot(ctx)
		b.respond(m, pa.Command, out, err, true)
	case "apply":
		if len(pa.Args) == 0 {
			b.reply(m.Chat.ID, "apply requires filename", 0)
			return
		}
		// move file from inbox to root (controlled)
		src := filepath.Join(b.sandbox, "inbox", filepath.Base(pa.Args[0]))
		dst := filepath.Join(b.sandbox, filepath.Base(pa.Args[0]))
		if err := os.Rename(src, dst); err != nil {
			b.respond(m, pa.Command, "", err, false)
			return
		}
		b.reply(m.Chat.ID, fmt.Sprintf("Applied %s to sandbox root", filepath.Base(pa.Args[0])), 0)
		b.audit.Write(m.From.ID, "/apply", "ok", map[string]string{"file": pa.Args[0]})
	default:
		b.reply(m.Chat.ID, "Unknown token command", 0)
	}
}

func (b *Bot) respond(m *tgbotapi.Message, cmd string, out string, err error, sensitive bool) {
	status := "ok"
	if err != nil {
		status = "error"
		out = fmt.Sprintf("%s", err)
	}
	ttl := time.Duration(0)
	if sensitive {
		ttl = time.Hour
	}
	sent := b.reply(m.Chat.ID, out, ttl)
	b.audit.Write(m.From.ID, "/"+cmd, status, nil)
	if sensitive && sent != nil {
		go b.deleteLater(m.Chat.ID, sent.MessageID, time.Hour)
	}
}

func (b *Bot) reply(chatID int64, text string, ttl time.Duration) *tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	sent, err := b.api.Send(msg)
	if err != nil {
		b.logger.Error().Err(err).Msg("send message failed")
		return nil
	}
	if ttl > 0 {
		go b.deleteLater(chatID, sent.MessageID, ttl)
	}
	return &sent
}

func (b *Bot) deleteLater(chatID int64, messageID int, ttl time.Duration) {
	timer := time.NewTimer(ttl)
	<-timer.C
	del := tgbotapi.DeleteMessageConfig{ChatID: chatID, MessageID: messageID}
	_, _ = b.api.Request(del)
}

func (b *Bot) requireMode(m *tgbotapi.Message, required mode.Mode) bool {
	if !b.modes.Allowed(required) {
		b.reply(m.Chat.ID, fmt.Sprintf("Command requires %s mode", required), 0)
		return false
	}
	return true
}

func helpText() string {
	return "LIFELINE commands:\n" +
		"/health /status /resources /ip\n" +
		"/diag net|time /logs <svc>\n" +
		"/ls [path] /get <path> /snapshot\n" +
		"/restart <svc> /cleanup /reboot (confirm)\n" +
		"/lockdown /unlock /mode"
}

func first(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
