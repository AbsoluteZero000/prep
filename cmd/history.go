package cmd

import (
	"fmt"

	"github.com/absolutezero000/prep/internal/models"
	"github.com/absolutezero000/prep/internal/storage"
	"github.com/absolutezero000/prep/internal/ui"
	"github.com/spf13/cobra"
)

var historyFlags struct {
	limit  int
	status string
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "List past interview sessions",
	RunE:  runHistory,
}

func init() {
	historyCmd.Flags().IntVar(&historyFlags.limit, "limit", 10, "Number of sessions to show")
	historyCmd.Flags().StringVar(&historyFlags.status, "status", "", "Filter: active|completed|aborted")
	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	store, err := storage.NewStore(storeBaseDir())
	if err != nil {
		return fmt.Errorf("initializing store: %w", err)
	}

	metas, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	if len(metas) == 0 {
		fmt.Println("No sessions found. Start one with 'prep start'.")
		return nil
	}

	// Filter by status if requested
	if historyFlags.status != "" {
		var filtered []models.SessionMeta
		for _, m := range metas {
			if string(m.Status) == historyFlags.status {
				filtered = append(filtered, m)
			}
		}
		metas = filtered
	}

	// Apply limit
	if historyFlags.limit > 0 && historyFlags.limit < len(metas) {
		metas = metas[:historyFlags.limit]
	}

	ui.PrintSessionHeader()
	for _, meta := range metas {
		ui.PrintSessionMeta(meta)
	}

	return nil
}
