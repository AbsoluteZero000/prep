package interview

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/absolutezero000/prep/internal/models"
	"github.com/absolutezero000/prep/internal/resume"
)

var (
	ErrEmptyAnswer    = errors.New("answer cannot be empty")
	ErrUserQuit       = errors.New("user requested to quit session")
	ErrMalformedScore = errors.New("malformed evaluation score from API")
)

// NewSession creates a new interview session.
func NewSession(role string, mode models.Mode, diff models.Difficulty, parseResult resume.ParseResult, numQ int) *models.Session {
	if numQ <= 0 {
		numQ = 5
	}
	if numQ > 15 {
		numQ = 15
	}

	return &models.Session{
		ID:           newSessionID(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ResumeHash:   parseResult.Hash,
		ResumePath:   "",
		Role:         role,
		Mode:         mode,
		Difficulty:   diff,
		NumQuestions: numQ,
		Questions:    nil,
		Turns:        nil,
		History: []models.Message{
			{Role: "system", Content: ""}, // placeholder, filled by engine
		},
		Status:     models.StatusReady,
		TokensUsed: 0,
	}
}

// UICallbacks provides hooks for the UI layer during engine operations.
type UICallbacks struct {
	OnQuestion   func(question string)
	OnStreaming  func(chunk string)
	OnScore      func(score models.Score)
	OnProcessing func() // called before LLM request starts
}

// ParseAnswer classifies a user's text response into an action.
func ParseAnswer(answer string) (action string, text string) {
	trimmed := strings.TrimSpace(strings.ToLower(answer))
	switch trimmed {
	case "skip":
		return "skip", ""
	case "hint":
		return "hint", ""
	case "quit":
		return "quit", ""
	case "exit":
		return "quit", ""
	case "":
		return "empty", ""
	default:
		return "answer", strings.TrimSpace(answer)
	}
}

// TurnDuration returns the duration between start and now.
func TurnDuration(startedAt time.Time) time.Duration {
	return time.Since(startedAt)
}

// newSessionID generates a short unique session identifier.
func newSessionID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ValidateAnswer checks answer length and returns a truncated version if too long.
func ValidateAnswer(answer string) (string, error) {
	if len(strings.TrimSpace(answer)) == 0 {
		return "", ErrEmptyAnswer
	}
	if len(answer) > 3000 {
		return answer[:3000], fmt.Errorf("answer truncated to 3000 characters")
	}
	return answer, nil
}
