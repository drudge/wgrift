package config

import (
	"fmt"
	"os"

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
}

type ServerConfig struct {
	Listen      string    `yaml:"listen"`
	ExternalURL string    `yaml:"external_url"`
	TLS         TLSConfig `yaml:"tls"`
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

func Defaults() Config {
	return Config{
		Demo: os.Getenv("WGRIFT_DEMO_MODE") == "true",
		Server: ServerConfig{
			Listen: "0.0.0.0:8080",
			TLS:    TLSConfig{Mode: "none"},
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
