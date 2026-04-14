package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Demo       bool             `yaml:"demo"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Auth       AuthConfig       `yaml:"auth"`
	Encryption EncryptionConfig `yaml:"encryption"`
	Logging    LoggingConfig    `yaml:"logging"`
	Backup     BackupConfig     `yaml:"backup"`
	UniFi      UniFiConfig      `yaml:"unifi"`
	SMTP       SMTPConfig       `yaml:"smtp"`
}

type ServerConfig struct {
	Listen              string    `yaml:"listen"`
	ExternalURL         string    `yaml:"external_url"`
	AutoStartInterfaces *bool     `yaml:"auto_start_interfaces"`
	TLS                 TLSConfig `yaml:"tls"`
}

func boolPtr(b bool) *bool { return &b }

// ShouldAutoStart returns whether interfaces should be synced on startup (default: true).
func (c *ServerConfig) ShouldAutoStart() bool {
	if c.AutoStartInterfaces == nil {
		return true
	}
	return *c.AutoStartInterfaces
}

type TLSConfig struct {
	Mode      string `yaml:"mode"`
	ACMEEmail string `yaml:"acme_email"`
	CertFile  string `yaml:"cert_file"`
	KeyFile   string `yaml:"key_file"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	SessionTimeout string       `yaml:"session_timeout"`
	MaxSessionAge  string       `yaml:"max_session_age"`
	Local          LocalAuth    `yaml:"local"`
	OIDC           []OIDCConfig `yaml:"oidc"`
	MFA            MFAConfig    `yaml:"mfa"`
}

type LocalAuth struct {
	Enabled           bool `yaml:"enabled"`
	MinPasswordLength int  `yaml:"min_password_length"`
}

type OIDCConfig struct {
	Name             string   `yaml:"name"`
	Issuer           string   `yaml:"issuer"`
	ClientID         string   `yaml:"client_id"`
	ClientSecretFile string   `yaml:"client_secret_file"`
	Scopes           []string `yaml:"scopes"`
	AdminClaim       string   `yaml:"admin_claim"`
	AdminValue       string   `yaml:"admin_value"`
}

type MFAConfig struct {
	Required bool `yaml:"required"`
}

type EncryptionConfig struct {
	MasterKeyFile string `yaml:"master_key_file"`
}

type LoggingConfig struct {
	ConnectionPollInterval string `yaml:"connection_poll_interval"`
	ConnectionTimeout      string `yaml:"connection_timeout"`
	RetentionDays          int    `yaml:"retention_days"`
	AuditRetentionDays     int    `yaml:"audit_retention_days"`
}

type BackupConfig struct {
	Path           string `yaml:"path"`
	Schedule       string `yaml:"schedule"`
	RetentionCount int    `yaml:"retention_count"`
}

type UniFiConfig struct {
	Enabled       bool   `yaml:"enabled"`
	ControllerURL string `yaml:"controller_url"`
	Username      string `yaml:"username"`
	PasswordFile  string `yaml:"password_file"`
	Site          string `yaml:"site"`
}

type SMTPConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	PasswordFile string `yaml:"password_file"`
	From         string `yaml:"from"`
	TLS          string `yaml:"tls"` // "none", "starttls", "tls"
}

// Enabled returns true if SMTP is configured.
func (c *SMTPConfig) Enabled() bool {
	return c.Host != ""
}

// Password returns the SMTP password from env var or file.
func (c *SMTPConfig) Password() (string, error) {
	if pw := os.Getenv("WGRIFT_SMTP_PASSWORD"); pw != "" {
		return pw, nil
	}
	if c.PasswordFile == "" {
		return "", nil
	}
	data, err := os.ReadFile(c.PasswordFile)
	if err != nil {
		return "", fmt.Errorf("reading SMTP password file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func Defaults() Config {
	return Config{
		Demo: os.Getenv("WGRIFT_DEMO_MODE") == "true",
		Server: ServerConfig{
			Listen:              "0.0.0.0:8080",
			AutoStartInterfaces: boolPtr(true),
			TLS:                 TLSConfig{Mode: "none"},
		},
		Database: DatabaseConfig{
			Path: "./wgrift.db",
		},
		Auth: AuthConfig{
			SessionTimeout: "30m",
			MaxSessionAge:  "24h",
			Local: LocalAuth{
				Enabled:           true,
				MinPasswordLength: 16,
			},
		},
		Encryption: EncryptionConfig{
			MasterKeyFile: "/etc/wgrift/master.key",
		},
		Logging: LoggingConfig{
			ConnectionPollInterval: "30s",
			ConnectionTimeout:      "180s",
			RetentionDays:          90,
			AuditRetentionDays:     365,
		},
		Backup: BackupConfig{
			Path:           "/var/lib/wgrift/backups",
			Schedule:       "0 2 * * *",
			RetentionCount: 30,
		},
		UniFi: UniFiConfig{
			Site: "default",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	// Allow env var override for master key
	if key := os.Getenv("WGRIFT_MASTER_KEY"); key != "" {
		cfg.Encryption.MasterKeyFile = ""
	}

	if os.Getenv("WGRIFT_DEMO_MODE") == "true" {
		cfg.Demo = true
	}

	if v := os.Getenv("WGRIFT_AUTO_START_INTERFACES"); v != "" {
		val := v == "true" || v == "1"
		cfg.Server.AutoStartInterfaces = &val
	}

	return cfg, nil
}

// MasterKey returns the master encryption key from env var or file.
func (c *Config) MasterKey() (string, error) {
	if key := os.Getenv("WGRIFT_MASTER_KEY"); key != "" {
		return key, nil
	}

	if c.Encryption.MasterKeyFile == "" {
		return "", fmt.Errorf("no master key configured: set WGRIFT_MASTER_KEY or encryption.master_key_file")
	}

	data, err := os.ReadFile(c.Encryption.MasterKeyFile)
	if err != nil {
		return "", fmt.Errorf("reading master key file: %w", err)
	}

	return string(data), nil
}
