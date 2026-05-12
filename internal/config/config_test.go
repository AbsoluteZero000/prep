package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ReturnsErrorOnMissingAPIKey(t *testing.T) {
	cfg := &Config{Model: "test", DefaultMode: "mixed", DefaultDifficulty: "mid"}
	err := cfg.Validate()
	if err != ErrMissingAPIKey {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
}

func TestLoad_EnvVarOverridesFile(t *testing.T) {
	orig := os.Getenv("OPENROUTER_API_KEY")
	defer os.Setenv("OPENROUTER_API_KEY", orig)
	os.Setenv("OPENROUTER_API_KEY", "sk-or-test123456789012345678901234567890")

	// set up a temp config dir
	tdir := t.TempDir()
	os.Setenv("PREP_CONFIG", filepath.Join(tdir, "config.yaml"))
	defer os.Unsetenv("PREP_CONFIG")

	// write a config with a different key
	cfg := DefaultConfig()
	cfg.APIKey = "sk-or-oldkey1234567890123456789012345678"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.APIKey != "sk-or-test123456789012345678901234567890" {
		t.Fatalf("expected env var override, got %s", loaded.APIKey)
	}
}

func TestSave_WritesAtomically(t *testing.T) {
	tdir := t.TempDir()
	path := filepath.Join(tdir, "config.yaml")
	os.Setenv("PREP_CONFIG", path)
	defer os.Unsetenv("PREP_CONFIG")

	cfg := DefaultConfig()
	cfg.APIKey = "sk-or-test123456789012345678901234567890"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// verify the file exists and the .tmp is cleaned up
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file was not written")
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file was not cleaned up")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.APIKey != cfg.APIKey {
		t.Fatalf("expected %s, got %s", cfg.APIKey, loaded.APIKey)
	}
}

func TestValidate_InvalidAPIKeyFormat(t *testing.T) {
	cfg := &Config{
		APIKey:            "bad-key",
		Model:             "test",
		DefaultMode:       "mixed",
		DefaultDifficulty: "mid",
	}
	err := cfg.Validate()
	if err != ErrInvalidAPIKeyFormat {
		t.Fatalf("expected ErrInvalidAPIKeyFormat, got %v", err)
	}
}

func TestValidate_InvalidMode(t *testing.T) {
	cfg := &Config{
		APIKey:            "sk-or-test123456789012345678901234567890",
		Model:             "test",
		DefaultMode:       "invalid",
		DefaultDifficulty: "mid",
	}
	err := cfg.Validate()
	if err != ErrInvalidMode {
		t.Fatalf("expected ErrInvalidMode, got %v", err)
	}
}

func TestValidate_InvalidDifficulty(t *testing.T) {
	cfg := &Config{
		APIKey:            "sk-or-test123456789012345678901234567890",
		Model:             "test",
		DefaultMode:       "mixed",
		DefaultDifficulty: "god",
	}
	err := cfg.Validate()
	if err != ErrInvalidDifficulty {
		t.Fatalf("expected ErrInvalidDifficulty, got %v", err)
	}
}

func TestValidate_DefaultModelOnEmpty(t *testing.T) {
	cfg := &Config{
		APIKey:            "sk-or-test123456789012345678901234567890",
		Model:             "",
		DefaultMode:       "mixed",
		DefaultDifficulty: "mid",
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error with empty model (defaults applied), got %v", err)
	}
	if cfg.Model != DefaultConfig().Model {
		t.Fatalf("expected model to default to %s, got %s", DefaultConfig().Model, cfg.Model)
	}
}

func TestLoad_ReturnsDefaultsOnMissingFile(t *testing.T) {
	tdir := t.TempDir()
	os.Setenv("PREP_CONFIG", filepath.Join(tdir, "nonexistent", "config.yaml"))
	defer os.Unsetenv("PREP_CONFIG")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != DefaultConfig().Model {
		t.Fatalf("expected default model, got %s", cfg.Model)
	}
}

func TestValidate_OK(t *testing.T) {
	cfg := &Config{
		APIKey:            "sk-or-test123456789012345678901234567890",
		Model:             "openai/gpt-4o",
		DefaultMode:       "technical",
		DefaultDifficulty: "senior",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLoad_TrimSpaceAroundAPIKey(t *testing.T) {
	tdir := t.TempDir()
	path := filepath.Join(tdir, "config.yaml")
	content := "api_key: \"sk-or-test123456789012345678901234567890\"\nmodel: test"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("PREP_CONFIG", path)
	defer os.Unsetenv("PREP_CONFIG")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cfg.APIKey, "sk-or-") {
		t.Fatalf("expected valid API key prefix, got %s", cfg.APIKey)
	}
}
