package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/absolutezero000/prep/internal/config"
	"github.com/absolutezero000/prep/internal/openrouter"
	"github.com/absolutezero000/prep/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key [api-key]",
	Short: "Set OpenRouter API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigSetKey,
}

var configSetModelCmd = &cobra.Command{
	Use:   "set-model [model-id]",
	Short: "Set the AI model",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigSetModel,
}

var configSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run interactive configuration setup",
	RunE:  runConfigSetup,
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete configuration file",
	RunE:  runConfigReset,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetKeyCmd)
	configCmd.AddCommand(configSetModelCmd)
	configCmd.AddCommand(configSetupCmd)
	configCmd.AddCommand(configResetCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println("Current configuration:")
	fmt.Printf("  Model:             %s\n", cfg.Model)
	fmt.Printf("  Default Mode:      %s\n", cfg.DefaultMode)
	fmt.Printf("  Default Difficulty: %s\n", cfg.DefaultDifficulty)
	fmt.Printf("  Remember Resume:   %v\n", cfg.RememberResume)
	if cfg.LastResumePath != "" {
		fmt.Printf("  Last Resume:       %s\n", cfg.LastResumePath)
	}

	// Redact API key - show last 4 chars
	if cfg.APIKey != "" {
		key := cfg.APIKey
		if len(key) > 4 {
			key = strings.Repeat("*", len(key)-4) + key[len(key)-4:]
		}
		fmt.Printf("  API Key:           %s\n", key)
	} else {
		fmt.Println("  API Key:           <not set>")
	}

	if len(cfg.FallbackModels) > 0 {
		fmt.Printf("  Fallback Models:   %s\n", strings.Join(cfg.FallbackModels, ", "))
	}

	return nil
}

func runConfigSetKey(cmd *cobra.Command, args []string) error {
	key := args[0]
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.APIKey = key

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("API key saved successfully.")
	return nil
}

func runConfigSetModel(cmd *cobra.Command, args []string) error {
	model := args[0]
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Validate against OpenRouter (optional, warn on failure)
	apiKey := cfg.APIKey
	if envKey := os.Getenv("OPENROUTER_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	if apiKey != "" {
		orc := openrouter.NewClient(apiKey, openrouter.ClientConfig{
			Model: model,
		})
		if err := orc.ValidateModel(cmd.Context(), model); err != nil {
			ui.PrintWarning(fmt.Sprintf("Model validation warning: %v", err))
		}
	}

	cfg.Model = model
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Model set to %s\n", model)
	return nil
}

func runConfigSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("Interactive setup:")
	fmt.Println()

	key := ui.PromptInput("Enter your OpenRouter API key (sk-or-...)")
	cfg := config.DefaultConfig()
	cfg.APIKey = key

	model := ui.PromptInput(fmt.Sprintf("AI model [%s]", cfg.Model))
	if model != "" {
		cfg.Model = model
	}

	mode := ui.PromptInput(fmt.Sprintf("Default interview mode (behavioral|technical|mixed|sysdesign) [%s]", cfg.DefaultMode))
	if mode != "" {
		cfg.DefaultMode = mode
	}

	diff := ui.PromptInput(fmt.Sprintf("Default difficulty (junior|mid|senior|staff) [%s]", cfg.DefaultDifficulty))
	if diff != "" {
		cfg.DefaultDifficulty = diff
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("\nConfiguration saved!")
	return nil
}

func runConfigReset(cmd *cobra.Command, args []string) error {
	if !ui.PromptConfirm("Are you sure you want to delete the configuration file?") {
		fmt.Println("Canceled.")
		return nil
	}

	// Find and remove config file
	cfgPath := ""
	if p := os.Getenv("PREP_CONFIG"); p != "" {
		cfgPath = p
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			cfgPath = home + "/.prep/config.yaml"
		}
	}

	if cfgPath != "" {
		if err := os.Remove(cfgPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Config file does not exist.")
				return nil
			}
			return fmt.Errorf("removing config: %w", err)
		}
		fmt.Println("Configuration deleted.")
	}

	return nil
}
