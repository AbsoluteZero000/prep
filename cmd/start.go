package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/absolutezero000/prep/internal/config"
	"github.com/absolutezero000/prep/internal/interview"
	"github.com/absolutezero000/prep/internal/models"
	"github.com/absolutezero000/prep/internal/openrouter"
	"github.com/absolutezero000/prep/internal/resume"
	"github.com/absolutezero000/prep/internal/storage"
	"github.com/absolutezero000/prep/internal/ui"
	"github.com/spf13/cobra"
)

var startFlags struct {
	resumePath string
	role       string
	mode       string
	difficulty string
	numQ       int
	model      string
	stream     bool
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new interview session",
	Long:  `Start an interactive AI-powered interview session based on your resume.`,
	RunE:  runStart,
}

func init() {
	startCmd.Flags().StringVarP(&startFlags.resumePath, "resume", "r", "", "Path to resume file (PDF, DOCX, TXT)")
	startCmd.Flags().StringVar(&startFlags.role, "role", "", "Target job role (default: inferred from resume)")
	startCmd.Flags().StringVar(&startFlags.mode, "mode", "", "interview mode: behavioral|technical|mixed|sysdesign")
	startCmd.Flags().StringVar(&startFlags.difficulty, "difficulty", "", "difficulty: junior|mid|senior|staff")
	startCmd.Flags().IntVarP(&startFlags.numQ, "questions", "q", 5, "Number of questions (max 15)")
	startCmd.Flags().StringVar(&startFlags.model, "model", "", "Override the AI model for this session")
	startCmd.Flags().BoolVar(&startFlags.stream, "stream", true, "Enable streaming output")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Resolve resume path
	resumePath := startFlags.resumePath
	if resumePath == "" {
		if cfg.RememberResume && cfg.LastResumePath != "" {
			fmt.Printf("Use cached resume from %s? [Y/n]: ", cfg.LastResumePath)
			var resp string
			fmt.Scanln(&resp)
			if resp != "n" && resp != "no" {
				resumePath = cfg.LastResumePath
			}
		}
		if resumePath == "" {
			return fmt.Errorf("provide a resume with --resume")
		}
	}

	// Parse resume
	var parseResult resume.ParseResult
	spinner := ui.NewSpinner("Parsing resume")
	spinner.Start()

	store, err := storage.NewStore(storeBaseDir())
	if err != nil {
		spinner.Fail(err)
		return err
	}

	parseResult, err = resume.Parse(resumePath)
	if err != nil {
		spinner.Fail(err)
		return fmt.Errorf("parsing resume: %w", err)
	}

	// Check cache
	if cached, ok := store.LoadCachedResume(parseResult.Hash); ok {
		parseResult.RawText = cached
		spinner.Stop()
		fmt.Println("Using cached resume")
	} else {
		store.CacheResume(parseResult.Hash, parseResult.RawText)
		spinner.Stop()
	}

	// Print warnings
	for _, w := range parseResult.Warnings {
		ui.PrintWarning(w)
	}

	// Determine role
	role := startFlags.role
	if role == "" {
		role = "Software Engineer" // default fallback
	}

	// Determine mode
	mode := models.ModeMixed
	if startFlags.mode != "" {
		mode = models.Mode(startFlags.mode)
	} else if cfg.DefaultMode != "" {
		mode = models.Mode(cfg.DefaultMode)
	}

	// Determine difficulty
	difficulty := models.DiffMid
	if startFlags.difficulty != "" {
		difficulty = models.Difficulty(startFlags.difficulty)
	} else if cfg.DefaultDifficulty != "" {
		difficulty = models.Difficulty(cfg.DefaultDifficulty)
	}

	// Create OpenRouter client
	apiKey := cfg.APIKey
	if envKey := os.Getenv("OPENROUTER_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	model := startFlags.model
	if model == "" {
		model = cfg.Model
	}

	orc := openrouter.NewClient(apiKey, openrouter.ClientConfig{
		Model:          model,
		FallbackModels: cfg.FallbackModels,
	})

	// Create session
	session := interview.NewSession(role, mode, difficulty, parseResult, startFlags.numQ)
	session.ResumePath = resumePath

	engine := interview.NewEngine(orc, store, session, parseResult.TokenEstimate)

	// Set system prompt with resume context
	sysPrompt, err := interview.RenderTemplate("system", interview.SystemData{
		Resume:       parseResult.RawText,
		Role:         role,
		Mode:         string(mode),
		Difficulty:   string(difficulty),
		NumQuestions: session.NumQuestions,
	})
	if err != nil {
		return fmt.Errorf("rendering system prompt: %w", err)
	}
	if len(session.History) > 0 {
		session.History[0].Content = sysPrompt
	}

	// Generate questions
	spinner2 := ui.NewSpinner("Generating interview questions")
	spinner2.Start()
	if err := engine.GenerateQuestions(context.Background()); err != nil {
		spinner2.Fail(err)
		return err
	}
	spinner2.Stop()
	fmt.Printf("Generated %d questions\n", len(session.Questions))

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n\nSession interrupted. Saving...")
		session.MarkAborted()
		store.SaveSession(session)
		fmt.Printf("Session saved. Resume with: prep review %s\n", session.ID)
		os.Exit(0)
	}()

	// Interview loop
	for i := 0; i < len(session.Questions); i++ {
		if session.Status == models.StatusAborted {
			break
		}

		question := session.Questions[i]
		ui.PrintQuestion(i, len(session.Questions), question)

		answer := ui.ReadMultilineInput("Your answer")

		action, _ := interview.ParseAnswer(answer)
		if action == "quit" {
			session.MarkAborted()
			store.SaveSession(session)
			fmt.Printf("Session saved. Resume with: prep review %s\n", session.ID)
			return nil
		}

		var fbSpinner *ui.Spinner
		var streamStarted bool
		callbacks := interview.UICallbacks{
			OnProcessing: func() {
				fbSpinner = ui.NewSpinner("Evaluating your answer")
				fbSpinner.Start()
			},
			OnStreaming: func(chunk string) {
				if !streamStarted {
					streamStarted = true
					if fbSpinner != nil {
						fbSpinner.Stop()
					}
					fmt.Print(ui.Colorize("─── Feedback ───\n", ui.ColorBold))
				}
				fmt.Print(ui.Colorize(chunk, ui.ColorMagenta))
			},
		}

		turn, err := engine.RunTurn(context.Background(), answer, callbacks)
		if fbSpinner != nil && !streamStarted {
			fbSpinner.Stop()
		}
		if err != nil {
			if err == interview.ErrEmptyAnswer {
				fmt.Println("Please provide an answer. Type 'skip' to skip.")
				i-- // retry same question
				continue
			}
			return fmt.Errorf("processing answer: %w", err)
		}
		if turn == nil && err == nil {
			i-- // hint was shown, retry same question
			continue
		}

		// Follow-up loop (max depth 3)
		furDepth := 0
		for furDepth < 3 && turn != nil && turn.Score != nil && turn.Score.FollowUpWarranted && turn.Score.FollowUpQuestion != "" {
			furDepth++
			fuq := turn.Score.FollowUpQuestion
			fuPrompt := fmt.Sprintf("Follow-up %d (max 3): %s", furDepth, fuq)
			fua := ui.ReadMultilineInput(fuPrompt)

			action, _ := interview.ParseAnswer(fua)
			if action == "quit" {
				session.MarkAborted()
				store.SaveSession(session)
				fmt.Printf("Session saved. Resume with: prep review %s\n", session.ID)
				return nil
			}

			streamStarted = false
			score, err := engine.RunFollowUp(context.Background(), fuq, fua, furDepth, callbacks)
			if fbSpinner != nil && !streamStarted {
				fbSpinner.Stop()
			}
			if err != nil {
				return fmt.Errorf("processing follow-up: %w", err)
			}
			turn.Score = score
		}

		if turn != nil && turn.Score != nil {
			ui.PrintScore(*turn.Score)
		}
	}

	// Generate summary
	if session.Status != models.StatusAborted {
		spinner3 := ui.NewSpinner("Generating session summary")
		spinner3.Start()
		summary, err := engine.Summarize(context.Background())
		if err != nil {
			spinner3.Fail(err)
			return err
		}
		spinner3.Stop()
		ui.PrintSummary(*summary)

		path, err := store.ExportMarkdown(session)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not export markdown: %v", err))
		} else {
			fmt.Printf("Session summary exported to: %s\n", path)
		}
	}

	return nil
}

// storeBaseDir returns the base directory for prep data.
func storeBaseDir() string {
	if d := os.Getenv("PREP_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".prep"
	}
	return home + "/.prep"
}
