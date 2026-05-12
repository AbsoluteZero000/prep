package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/absolutezero000/prep/internal/models"
)

// PrintQuestion displays an interview question with numbering.
func PrintQuestion(n, total int, question string) {
	prefix := Colorize(fmt.Sprintf("[Q%d/%d]", n+1, total), ColorCyan+ColorBold)
	text := Colorize(question, ColorCyan)
	fmt.Printf("\n%s %s\n\n", prefix, text)
}

// PrintScore displays an evaluation score.
func PrintScore(score models.Score) {
	color := ColorGreen
	if score.Score <= 4 {
		color = ColorRed
	} else if score.Score <= 7 {
		color = ColorYellow
	}

	label := Colorize(fmt.Sprintf("Score: %d/10", score.Score), color+ColorBold)
	fmt.Printf("\n%s\n", label)

	if len(score.Strengths) > 0 {
		fmt.Printf("  %s %s\n", Colorize("Strengths:", ColorGreen), strings.Join(score.Strengths, ", "))
	}
	if len(score.Gaps) > 0 {
		fmt.Printf("  %s %s\n", Colorize("Gaps:", ColorYellow), strings.Join(score.Gaps, ", "))
	}
	fmt.Println()
}

// PrintSummary displays the end-of-session summary.
func PrintSummary(summary models.Summary) {
	scoreColor := ColorGreen
	if summary.OverallScore <= 4 {
		scoreColor = ColorRed
	} else if summary.OverallScore <= 7 {
		scoreColor = ColorYellow
	}

	fmt.Println(Colorize("\n═══ Session Summary ═══", ColorBold))
	fmt.Printf("%s %s\n",
		Colorize("Overall Score:", ColorBold),
		Colorize(fmt.Sprintf("%d/10", summary.OverallScore), scoreColor+ColorBold))

	if summary.OverallAssessment != "" {
		fmt.Printf("\n%s\n", summary.OverallAssessment)
	}

	if len(summary.TopStrengths) > 0 {
		fmt.Printf("\n%s\n", Colorize("Top Strengths:", ColorGreen+ColorBold))
		for _, s := range summary.TopStrengths {
			fmt.Printf("  ✓ %s\n", s)
		}
	}
	if len(summary.CriticalGaps) > 0 {
		fmt.Printf("\n%s\n", Colorize("Critical Gaps:", ColorRed+ColorBold))
		for _, s := range summary.CriticalGaps {
			fmt.Printf("  ✗ %s\n", s)
		}
	}
	if len(summary.RecommendedStudyTopics) > 0 {
		fmt.Printf("\n%s\n", Colorize("Recommended Study Topics:", ColorCyan+ColorBold))
		for _, s := range summary.RecommendedStudyTopics {
			fmt.Printf("  → %s\n", s)
		}
	}
	fmt.Printf("\n%s %s\n",
		Colorize("Recommendation:", ColorBold),
		Colorize(summary.HiringRecommendation, colorForRec(summary.HiringRecommendation)))
	fmt.Println(Colorize("══════════════════════\n", ColorBold))
}

func colorForRec(rec string) string {
	switch rec {
	case "strong_yes", "yes":
		return ColorGreen
	case "maybe":
		return ColorYellow
	default:
		return ColorRed
	}
}

// PrintSessionMeta displays a single session in a listing.
func PrintSessionMeta(meta models.SessionMeta) {
	score := "—"
	if meta.OverallScore != nil {
		score = fmt.Sprintf("%d/10", *meta.OverallScore)
	}
	statusColor := ColorGreen
	if meta.Status == models.StatusAborted {
		statusColor = ColorRed
	} else if meta.Status == models.StatusActive {
		statusColor = ColorYellow
	}

	fmt.Printf("  %s  %s  %-20s  %-10s  %-8s  %s  %s\n",
		meta.ID,
		meta.CreatedAt.Format("2006-01-02"),
		meta.Role,
		meta.Mode,
		meta.Difficulty,
		score,
		Colorize(string(meta.Status), statusColor))
}

// PrintSessionHeader prints the table header for session listing.
func PrintSessionHeader() {
	fmt.Println(Colorize(fmt.Sprintf("%-12s %-10s %-22s %-12s %-10s %-6s %s",
		"ID", "Date", "Role", "Mode", "Difficulty", "Score", "Status"), ColorBold))
	fmt.Println(strings.Repeat("─", 80))
}

// PromptConfirm asks a yes/no question and returns the answer.
func PromptConfirm(question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(strings.ToLower(text))
	return text == "y" || text == "yes"
}

// PromptInput reads a single line of input.
func PromptInput(label string) string {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

// ReadMultilineInput reads multiline input terminated by an empty line.
// In TTY mode it uses a raw-mode editor with cursor keys, word deletion, etc.
func ReadMultilineInput(prompt string) string {
	fullPrompt := prompt + Colorize(" (3000 chars max, press Enter twice to submit)", ColorGray)

	if IsTTY() {
		result, err := runEditor(fullPrompt)
		if err != nil {
			return ""
		}
		return result
	}

	// Non-TTY fallback: line-by-line with bufio
	fmt.Println(fullPrompt)
	reader := bufio.NewReader(os.Stdin)
	var lines []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lines = append(lines, line)
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
