package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig holds all configuration loaded from env or YAML.
type AppConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
	DSM      DSMConfig      `yaml:"dsm"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
	Sandbox  SandboxConfig  `yaml:"sandbox"`
}

// TelegramConfig describes Telegram bot settings.
type TelegramConfig struct {
	Token        string  `yaml:"token"`
	AdminChatIDs []int64 `yaml:"admin_chat_ids"`
	PollTimeout  int     `yaml:"poll_timeout"`
}

// DSMConfig contains Synology DSM API settings.
type DSMConfig struct {
	BaseURL           string `yaml:"base_url"`
	APIToken          string `yaml:"api_token"`
	TokenRefreshHours int    `yaml:"token_refresh_hours"`
}

// SecurityConfig stores rate limit and command control settings.
type SecurityConfig struct {
	RateLimitPerMin   int      `yaml:"rate_limit"`
	CmdWhitelist      []string `yaml:"cmd_whitelist"`
	ConfirmTTLSeconds int      `yaml:"confirm_ttl_seconds"`
	DefaultMode       string   `yaml:"default_mode"`
	MaxRequestPerMin  int      `yaml:"max_request_per_min"` // alias, fallback if provided
}

// LoggingConfig controls log level/output.
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// SandboxConfig defines sandbox and file settings.
type SandboxConfig struct {
	Root      string `yaml:"root"`
	MaxFileMB int    `yaml:"max_file_mb"`
}

// Load reads YAML config (if present) and overrides with env vars.
func Load(path string) (*AppConfig, error) {
	cfg := defaultConfig()

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read config: %w", err)
			}
			if err := yaml.Unmarshal(b, cfg); err != nil {
				return nil, fmt.Errorf("parse yaml: %w", err)
			}
		}
	}

	overrideFromEnv(cfg)

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func defaultConfig() *AppConfig {
	return &AppConfig{
		Telegram: TelegramConfig{
			PollTimeout: 30,
		},
		DSM: DSMConfig{
			TokenRefreshHours: 24,
		},
		Security: SecurityConfig{
			RateLimitPerMin:   5,
			ConfirmTTLSeconds: 60,
			DefaultMode:       "readonly",
		},
		Logging: LoggingConfig{Level: "info"},
		Sandbox: SandboxConfig{
			Root:      "/emergency-files",
			MaxFileMB: 50,
		},
	}
}

func overrideFromEnv(cfg *AppConfig) {
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		cfg.Telegram.Token = v
	}
	if v := os.Getenv("ALLOWED_USER_IDS"); v != "" {
		ids := strings.Split(v, ",")
		cfg.Telegram.AdminChatIDs = make([]int64, 0, len(ids))
		for _, id := range ids {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
					cfg.Telegram.AdminChatIDs = append(cfg.Telegram.AdminChatIDs, parsed)
				}
			}
		}
	}
	if v := os.Getenv("LIFELINE_MODE"); v != "" {
		cfg.Security.DefaultMode = strings.ToLower(v)
	}
	if v := os.Getenv("RATE_LIMIT_PER_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.RateLimitPerMin = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Security.RateLimitPerMin = 60 / n
		}
	}
	if v := os.Getenv("SANDBOX_ROOT"); v != "" {
		cfg.Sandbox.Root = v
	}
	if v := os.Getenv("MAX_FILE_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Sandbox.MaxFileMB = n
		}
	}
	if v := os.Getenv("CONFIRM_TOKEN_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Security.ConfirmTTLSeconds = n
		}
	}
	if v := os.Getenv("DSM_BASE_URL"); v != "" {
		cfg.DSM.BaseURL = v
	}
	if v := os.Getenv("DSM_API_TOKEN"); v != "" {
		cfg.DSM.APIToken = v
	}
}

func (c *AppConfig) validate() error {
	if c.Telegram.Token == "" {
		return errors.New("telegram token required")
	}
	if len(c.Telegram.AdminChatIDs) == 0 {
		return errors.New("admin chat ids required")
	}
	if c.Sandbox.Root == "" {
		return errors.New("sandbox root required")
	}
	if c.Security.RateLimitPerMin <= 0 {
		return errors.New("rate limit must be >0")
	}
	if c.Security.ConfirmTTLSeconds <= 0 {
		return errors.New("confirm ttl must be >0")
	}
	mode := strings.ToLower(c.Security.DefaultMode)
	switch mode {
	case "readonly", "emergency", "lockdown":
		c.Security.DefaultMode = mode
	default:
		return fmt.Errorf("invalid default mode: %s", c.Security.DefaultMode)
	}
	return nil
}

// ConfirmTTL returns TTL as duration.
func (c *AppConfig) ConfirmTTL() time.Duration {
	return time.Duration(c.Security.ConfirmTTLSeconds) * time.Second
}

// TokenRefreshInterval returns DSM token rotation interval.
func (c *AppConfig) TokenRefreshInterval() time.Duration {
	return time.Duration(c.DSM.TokenRefreshHours) * time.Hour
}
