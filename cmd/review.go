package cmd

import (
	"fmt"
	"strings"

	"github.com/absolutezero000/prep/internal/storage"
	"github.com/absolutezero000/prep/internal/ui"
	"github.com/spf13/cobra"
)

var reviewFlags struct {
	export bool
}

var reviewCmd = &cobra.Command{
	Use:   "review [session-id]",
	Short: "Review a past interview session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReview,
}

func init() {
	reviewCmd.Flags().BoolVar(&reviewFlags.export, "export", false, "Export session to markdown")
	rootCmd.AddCommand(reviewCmd)
}

//nolint:gocyclo
func runReview(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("session ID is required")
	}
	sessionID := args[0]

	store, err := storage.NewStore(storeBaseDir())
	if err != nil {
		return fmt.Errorf("initializing store: %w", err)
	}

	// Try exact match first
	sess, err := store.LoadSession(sessionID)
	if err != nil {
		// Try partial ID match
		metas, listErr := store.ListSessions()
		if listErr != nil {
			return fmt.Errorf("session not found and cannot list sessions: %w", err)
		}

		var matches []string
		for _, m := range metas {
			if strings.HasPrefix(m.ID, sessionID) {
				matches = append(matches, m.ID)
			}
		}

		if len(matches) == 0 {
			return fmt.Errorf("no session found matching '%s'", sessionID)
		}
		if len(matches) > 1 {
			return fmt.Errorf("multiple sessions match '%s': %v", sessionID, matches)
		}

		sess, err = store.LoadSession(matches[0])
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}
	}

	// Print session details
	fmt.Printf("\n%s\n", ui.Colorize(fmt.Sprintf("Session: %s", sess.ID), ui.ColorBold))
	fmt.Printf("Date: %s | Role: %s | Mode: %s | Difficulty: %s\n",
		sess.CreatedAt.Format("2006-01-02 15:04"),
		sess.Role, sess.Mode, sess.Difficulty)

	if sess.Status != "" {
		statusColor := ui.ColorGreen
		if sess.Status == "aborted" {
			statusColor = ui.ColorRed
		}
		fmt.Printf("Status: %s\n\n", ui.Colorize(string(sess.Status), statusColor))
	}

	for i, turn := range sess.Turns {
		ui.PrintQuestion(i, len(sess.Questions), turn.Question)
		fmt.Printf("Answer: %s\n\n", turn.Answer)

		if turn.Score != nil {
			ui.PrintScore(*turn.Score)
		}

		if turn.Skipped {
			ui.PrintWarning("(skipped)")
		}

		for j, fu := range turn.FollowUps {
			fmt.Printf("Follow-up %d: %s\n", j+1, fu.Question)
		}
		fmt.Println(strings.Repeat("─", 60))
	}

	if sess.Summary != nil {
		ui.PrintSummary(*sess.Summary)
	}

	if reviewFlags.export {
		path, err := store.ExportMarkdown(sess)
		if err != nil {
			return fmt.Errorf("exporting session: %w", err)
		}
		fmt.Printf("Exported to: %s\n", path)
	}

	return nil
}
