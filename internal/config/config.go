// Package config manages user configuration for the prep CLI.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/absolutezero000/prep/internal/storage"
	"gopkg.in/yaml.v3"
)

var (
	ErrMissingAPIKey       = errors.New("API key is required — set OPENROUTER_API_KEY or run 'prep config setup'")
	ErrInvalidAPIKeyFormat = errors.New("API key must start with 'sk-or-' followed by at least 32 alphanumeric characters")
	ErrInvalidMode         = errors.New("mode must be one of: behavioral, technical, mixed, sysdesign")
	ErrInvalidDifficulty   = errors.New("difficulty must be one of: junior, mid, senior, staff")
	ErrConfigNotFound      = errors.New("config file not found — run 'prep config setup' first")
)

var (
	allowedModes        = map[string]bool{"behavioral": true, "technical": true, "mixed": true, "sysdesign": true}
	allowedDifficulties = map[string]bool{"junior": true, "mid": true, "senior": true, "staff": true}
	apiKeyRegex         = regexp.MustCompile(`^sk-or-[a-zA-Z0-9-]{32,}$`)
)

// configPathOverride allows tests and CLI to override the config location.
var configPathOverride string

// SetConfigPath sets a custom config path (for CLI --config flag).
func SetConfigPath(path string) {
	configPathOverride = path
}

// Config holds all user-configurable settings for the prep CLI.
type Config struct {
	APIKey            string   `yaml:"api_key"`
	Model             string   `yaml:"model"`
	FallbackModels    []string `yaml:"fallback_models"`
	DefaultMode       string   `yaml:"default_mode"`
	DefaultDifficulty string   `yaml:"default_difficulty"`
	RememberResume    bool     `yaml:"remember_resume"`
	LastResumePath    string   `yaml:"last_resume_path"`
}

func configPath() string {
	if configPathOverride != "" {
		return configPathOverride
	}
	if p := os.Getenv("PREP_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".prep", "config.yaml")
}

func configDir() string {
	return filepath.Dir(configPath())
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Model:             "mistralai/mistral-7b-instruct",
		DefaultMode:       "mixed",
		DefaultDifficulty: "mid",
	}
}

// Load reads the config from disk, applying environment variable overrides.
// If the file does not exist it returns a default config without error.
func Load() (*Config, error) {
	cfg := DefaultConfig()
	path := configPath()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // config path is user-controlled by design
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if envKey := os.Getenv("OPENROUTER_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	}

	return cfg, nil
}

// Save writes the config to disk atomically.
func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := configPath()
	if path == "" {
		return errors.New("cannot determine config path")
	}
	if err := storage.WriteFileAtomic(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// Validate checks all config fields and returns the first error encountered.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}
	if !apiKeyRegex.MatchString(c.APIKey) {
		return ErrInvalidAPIKeyFormat
	}
	if c.Model == "" {
		c.Model = DefaultConfig().Model
	}
	if !allowedModes[strings.ToLower(c.DefaultMode)] {
		return ErrInvalidMode
	}
	if !allowedDifficulties[strings.ToLower(c.DefaultDifficulty)] {
		return ErrInvalidDifficulty
	}
	return nil
}

// RunSetupWizard provides an interactive first-run setup (stub for now).
func RunSetupWizard() error {
	path := configPath()
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	fmt.Printf("Config directory created at %s\n", dir)
	fmt.Printf("Run 'prep config set-key YOUR_API_KEY' to set your OpenRouter API key.\n")
	fmt.Printf("Config file will be stored at %s\n", path)
	return nil
}
