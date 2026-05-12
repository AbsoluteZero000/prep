// Package tokens manages token budgets and estimation for LLM context windows.
package tokens

import (
	"errors"
	"math"
	"strings"

	"github.com/absolutezero000/prep/internal/models"
)

// ErrContextWindowExhausted is returned when history cannot be trimmed to fit within the budget.
var ErrContextWindowExhausted = errors.New("context window exhausted — cannot trim history to fit model limit")

// ModelContextLimits maps model IDs to their max context token limits.
var ModelContextLimits = map[string]int{
	"mistralai/mistral-7b-instruct":     8192,
	"mistralai/mixtral-8x7b-instruct":   32768,
	"anthropic/claude-3-haiku":          200000,
	"openai/gpt-4o":                     128000,
	"openai/gpt-3.5-turbo":              16385,
	"meta-llama/llama-3-8b-instruct":    8192,
	"meta-llama/llama-3-70b-instruct":   8192,
	"meta-llama/llama-3.1-8b-instruct":  131072,
	"meta-llama/llama-3.1-70b-instruct": 131072,
	"google/gemini-1.5-pro":             1048576,
	"google/gemini-1.5-flash":           1048576,
}

// GetModelLimit returns the context limit for a model, defaulting to 8192 if unknown.
func GetModelLimit(model string) int {
	if limit, ok := ModelContextLimits[model]; ok {
		return limit
	}
	return 8192
}

// Budget tracks token allocation across prompt components.
type Budget struct {
	ModelLimit      int
	SystemPrompt    int
	ResumeTokens    int
	ResponseReserve int
}

// NewBudget creates a Budget with the given constraints.
func NewBudget(modelLimit, systemPrompt, resumeTokens int) *Budget {
	return &Budget{
		ModelLimit:      modelLimit,
		SystemPrompt:    systemPrompt,
		ResumeTokens:    resumeTokens,
		ResponseReserve: 1024,
	}
}

// Available returns the number of tokens remaining for conversation history.
func (b *Budget) Available() int {
	return b.ModelLimit - b.SystemPrompt - b.ResumeTokens - b.ResponseReserve
}

// CanFit returns true if the given token count fits within the available budget.
func (b *Budget) CanFit(tokens int) bool {
	return tokens <= b.Available()
}

// TrimHistory removes oldest non-essential messages until the history fits the budget.
// It never drops the system message (index 0) or the first user message.
// If even the trimmed minimum does not fit, it returns ErrContextWindowExhausted.
func (b *Budget) TrimHistory(msgs []models.Message) []models.Message {
	if len(msgs) == 0 {
		return msgs
	}

	for {
		if b.CanFit(estimateMessagesTokens(msgs)) {
			return msgs
		}

		// Try dropping the oldest non-essential message
		dropped := false
		for i := 1; i < len(msgs); i++ {
			if msgs[i].Role == "user" && i == firstUserIndex(msgs) {
				continue // keep first user message
			}
			msgs = append(msgs[:i], msgs[i+1:]...)
			dropped = true
			break
		}
		if !dropped {
			break
		}
	}

	// After trimming, if still doesn't fit, return error (caller checks)
	return msgs
}

// TrimHistoryToFit is like TrimHistory but returns an error if the budget cannot be satisfied.
func (b *Budget) TrimHistoryToFit(msgs []models.Message) ([]models.Message, error) {
	result := b.TrimHistory(msgs)
	if !b.CanFit(estimateMessagesTokens(result)) {
		return result, ErrContextWindowExhausted
	}
	return result, nil
}

// EstimateTokens estimates the token count of a text string (words * 1.35, rounded up).
func EstimateTokens(text string) int {
	words := len(strings.Fields(text))
	return int(math.Ceil(float64(words) * 1.35))
}

func estimateMessagesTokens(msgs []models.Message) int {
	var total int
	for _, m := range msgs {
		total += EstimateTokens(m.Content)
	}
	return total
}

func firstUserIndex(msgs []models.Message) int {
	for i, m := range msgs {
		if m.Role == "user" {
			return i
		}
	}
	return -1
}
