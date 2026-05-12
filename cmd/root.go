package cmd

import (
	"fmt"
	"os"

	"github.com/absolutezero000/prep/internal/config"
	"github.com/spf13/cobra"
)

var cfgFile string
var noColor bool

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "prep",
	Short: "AI-powered interview preparation CLI",
	Long: `prep - AI-powered interview preparation tool.

Uses your resume to generate personalized interview questions,
evaluates your answers, and provides feedback to help you improve.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config check for setup wizard
		if cmd.Name() == "setup" || cmd.Name() == "help" {
			return nil
		}
		return initConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands and sets the flags.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(func() {})
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.prep/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
}

func initConfig() error {
	if cfgFile != "" {
		config.SetConfigPath(cfgFile)
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.APIKey == "" && os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Println("No API key configured. Run 'prep config setup' to get started.")
		fmt.Println("Or set the OPENROUTER_API_KEY environment variable.")
		return nil
	}

	return nil
}
