package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for Axon
type Config struct {
	Server   ServerConfig
	Browser  BrowserConfig
	Security SecurityConfig
	Storage  StorageConfig
	Logging  LoggingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// BrowserConfig holds browser configuration
type BrowserConfig struct {
	Headless       bool
	BinaryPath     string
	PoolSize       int
	LaunchOptions map[string]interface{}
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	SSRF           SSRFConfig
	PromptInjection PromptInjectionConfig
	Reversibility  ReversibilityConfig
}

// SSRFConfig holds SSRF protection configuration
type SSRFConfig struct {
	Enabled             bool
	AllowPrivateNetwork bool
	DomainAllowlist    []string
	DomainDenylist     []string
	SchemeAllowlist    []string
}

// PromptInjectionConfig holds prompt injection detection configuration
type PromptInjectionConfig struct {
	Enabled     bool
	Mode        string // warn, strip, block
	Sensitivity string // low, medium, high
}

// ReversibilityConfig holds action reversibility configuration
type ReversibilityConfig struct {
	RequireConfirm           bool
	ActionBudgetPerHour     int
	EscalateOnBudgetExceeded bool
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Path          string
	SessionTTL    time.Duration
	AuditRetention time.Duration
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output string // stdout, file
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8020,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Browser: BrowserConfig{
			Headless:   true,
			PoolSize:   5,
			LaunchOptions: map[string]interface{}{},
		},
		Security: SecurityConfig{
			SSRF: SSRFConfig{
				Enabled:             true,
				AllowPrivateNetwork: false,
				DomainAllowlist:    []string{},
				DomainDenylist:     []string{},
				SchemeAllowlist:    []string{"https", "http"},
			},
			PromptInjection: PromptInjectionConfig{
				Enabled:     true,
				Mode:        "warn",
				Sensitivity: "medium",
			},
			Reversibility: ReversibilityConfig{
				RequireConfirm:           true,
				ActionBudgetPerHour:     10,
				EscalateOnBudgetExceeded: true,
			},
		},
		Storage: StorageConfig{
			Path:          "./data/axon.db",
			SessionTTL:    24 * time.Hour,
			AuditRetention: 90 * 24 * time.Hour,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

// Load loads configuration from file, environment, and flags
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Set up viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.axon")
	viper.AddConfigPath("/etc/axon")

	// Environment variables
	viper.SetEnvPrefix("AXON")
	viper.AutomaticEnv()

	// Override with environment variables
	viper.BindEnv("SERVER_HOST", "host")
	viper.BindEnv("SERVER_PORT", "port")
	viper.BindEnv("BROWSER_HEADLESS", "headless")
	viper.BindEnv("BROWSER_POOL", "pool_size")
	viper.BindEnv("LOG_LEVEL", "log_level")
	viper.BindEnv("DATA_DIR", "data_dir")

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Unmarshal to config
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
