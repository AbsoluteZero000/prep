package interview

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absolutezero000/prep/internal/models"
	"github.com/absolutezero000/prep/internal/openrouter"
	"github.com/absolutezero000/prep/internal/storage"
	"github.com/absolutezero000/prep/internal/tokens"
)

// Engine orchestrates the interview session lifecycle.
type Engine struct {
	client  *openrouter.Client
	budget  *tokens.Budget
	store   *storage.Store
	session *models.Session
}

// NewEngine creates an engine bound to a specific session.
// resumeTokens is the estimated token count of the resume.
func NewEngine(client *openrouter.Client, store *storage.Store, session *models.Session, resumeTokens int) *Engine {
	modelLimit := tokens.GetModelLimit(client.Model())
	budget := tokens.NewBudget(modelLimit, 400, resumeTokens)
	return &Engine{
		client:  client,
		store:   store,
		session: session,
		budget:  budget,
	}
}

// GenerateQuestions uses the LLM to generate interview questions based on the resume.
func (e *Engine) GenerateQuestions(ctx context.Context) error {
	e.session.Status = models.StatusActive
	e.session.UpdatedAt = time.Now()

	resumeText := ""
	if e.session.ResumeHash != "" {
		if cached, ok := e.store.LoadCachedResume(e.session.ResumeHash); ok {
			resumeText = cached
		}
	}

	seen := make(map[string]bool)
	var unique []string
	for attempt := 0; attempt < 2 && len(unique) < e.session.NumQuestions; attempt++ {
		prompt, err := RenderTemplate("question_gen", QuestionGenData{
			NumQuestions: e.session.NumQuestions,
			Difficulty:   string(e.session.Difficulty),
			Role:         e.session.Role,
			Resume:       resumeText,
		})
		if err != nil {
			return fmt.Errorf("rendering question gen template: %w", err)
		}

		req := openrouter.ChatRequest{
			Model: e.client.Model(),
			Messages: []models.Message{
				{Role: "system", Content: prompt},
			},
			MaxTokens:      2000,
			ResponseFormat: &openrouter.ResponseFormat{Type: "json_object"},
		}

		resp, err := e.client.Chat(ctx, req)
		if err != nil {
			return fmt.Errorf("generating questions: %w", err)
		}

		e.session.TokensUsed += resp.Usage.TotalTokens

		var result struct {
			Questions []string `json:"questions"`
		}
		if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
			return fmt.Errorf("parsing questions JSON: %w", err)
		}

		if len(result.Questions) == 0 {
			return fmt.Errorf("API returned empty questions list")
		}

		for _, q := range result.Questions {
			normalized := strings.ToLower(strings.TrimSpace(q))
			if !seen[normalized] {
				seen[normalized] = true
				unique = append(unique, q)
			}
		}
	}

	e.session.Questions = unique
	e.session.NumQuestions = len(unique)

	if err := e.store.SaveSession(e.session); err != nil {
		return fmt.Errorf("saving session after question generation: %w", err)
	}
	return nil
}

// ensureSystemPrompt fills the system message placeholder if not yet set.
func (e *Engine) ensureSystemPrompt() error {
	if len(e.session.History) > 0 && e.session.History[0].Content != "" {
		return nil // already set
	}

	resumeText := ""
	if e.session.ResumeHash != "" {
		if cached, ok := e.store.LoadCachedResume(e.session.ResumeHash); ok {
			resumeText = cached
		}
	}

	systemPrompt, err := RenderTemplate("system", SystemData{
		Resume:       resumeText,
		Role:         e.session.Role,
		Mode:         string(e.session.Mode),
		Difficulty:   string(e.session.Difficulty),
		NumQuestions: e.session.NumQuestions,
	})
	if err != nil {
		return fmt.Errorf("rendering system prompt: %w", err)
	}

	if len(e.session.History) > 0 {
		e.session.History[0].Content = systemPrompt
	} else {
		e.session.History = []models.Message{{Role: "system", Content: systemPrompt}}
	}
	return nil
}

// RunTurn processes a single interview turn: validate input, get response, evaluate.
//
//nolint:gocyclo
func (e *Engine) RunTurn(ctx context.Context, answer string, callbacks UICallbacks) (*models.Turn, error) {
	// Ensure system prompt is set before the first turn
	if err := e.ensureSystemPrompt(); err != nil {
		return nil, err
	}

	action, text := ParseAnswer(answer)

	switch action {
	case "empty":
		return nil, ErrEmptyAnswer
	case "skip":
		turn := &models.Turn{
			Index:     e.session.CurrentTurnIndex(),
			StartedAt: time.Now(),
			Skipped:   true,
			Duration:  0,
		}
		e.session.Turns = append(e.session.Turns, *turn)
		e.session.UpdatedAt = time.Now()
		if err := e.store.SaveSession(e.session); err != nil {
			return nil, fmt.Errorf("saving session after skip: %w", err)
		}
		return turn, nil
	case "quit":
		e.session.MarkAborted()
		if err := e.store.SaveSession(e.session); err != nil {
			return nil, fmt.Errorf("saving session after quit: %w", err)
		}
		return nil, ErrUserQuit
	case "hint":
		hint, err := e.generateHint(ctx)
		if err != nil {
			return nil, err
		}
		if callbacks.OnStreaming != nil {
			callbacks.OnStreaming(hint)
		}
		return nil, nil // same turn, don't advance
	}

	// Validate and sanitize answer
	sanitized, err := ValidateAnswer(text)
	if err != nil {
		// Truncation warning is non-fatal
		text = sanitized
	}

	// Get current question
	if e.session.CurrentTurnIndex() >= len(e.session.Questions) {
		return nil, fmt.Errorf("all questions exhausted")
	}
	question := e.session.Questions[e.session.CurrentTurnIndex()]

	turn := &models.Turn{
		Index:     e.session.CurrentTurnIndex(),
		Question:  question,
		Answer:    text,
		StartedAt: time.Now(),
	}

	// Append user message to history
	e.session.History = append(e.session.History, models.Message{Role: "user", Content: text})

	// Trim history to fit budget
	e.session.History, _ = e.budget.TrimHistoryToFit(e.session.History)

	// Get LLM response (streaming)
	if callbacks.OnProcessing != nil {
		callbacks.OnProcessing()
	}
	if callbacks.OnQuestion != nil {
		callbacks.OnQuestion(question)
	}

	req := openrouter.ChatRequest{
		Model:     e.client.Model(),
		Messages:  e.session.History,
		MaxTokens: 1024,
		Stream:    true,
	}

	stream, err := e.client.ChatStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("starting chat stream: %w", err)
	}

	var fullResponse strings.Builder
	for chunk := range stream {
		fullResponse.WriteString(chunk)
		if callbacks.OnStreaming != nil {
			callbacks.OnStreaming(chunk)
		}
	}

	responseText := strings.TrimSpace(fullResponse.String())
	e.session.History = append(e.session.History, models.Message{Role: "assistant", Content: responseText})

	// Evaluate the answer
	score, err := e.Evaluate(ctx, turn.Question, turn.Answer, 0, "")
	if err != nil {
		return nil, fmt.Errorf("evaluating answer: %w", err)
	}
	turn.Score = score
	turn.Duration = TurnDuration(turn.StartedAt)

	e.session.Turns = append(e.session.Turns, *turn)

	// Check if all questions complete
	if e.session.CurrentTurnIndex() >= e.session.NumQuestions {
		e.session.Status = models.StatusCompleted
	}

	e.session.UpdatedAt = time.Now()
	if err := e.store.SaveSession(e.session); err != nil {
		return nil, fmt.Errorf("saving session after turn: %w", err)
	}

	return turn, nil
}

// Evaluate scores a single answer against the expected answer.
// context provides previous Q&A history for follow-up evaluations (empty for initial answer).
func (e *Engine) Evaluate(ctx context.Context, question, answer string, furDepth int, context string) (*models.Score, error) {
	prompt, err := RenderTemplate("evaluate", EvaluateData{
		Question:      question,
		Answer:        answer,
		Role:          e.session.Role,
		Context:       context,
		FollowUpDepth: furDepth,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering evaluate template: %w", err)
	}

	req := openrouter.ChatRequest{
		Model: e.client.Model(),
		Messages: []models.Message{
			{Role: "system", Content: prompt},
		},
		MaxTokens:      1000,
		ResponseFormat: &openrouter.ResponseFormat{Type: "json_object"},
	}

	resp, err := e.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("evaluation API call: %w", err)
	}

	e.session.TokensUsed += resp.Usage.TotalTokens

	var score models.Score
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &score); err != nil {
		// Retry once with explicit JSON instruction
		retryReq := openrouter.ChatRequest{
			Model: e.client.Model(),
			Messages: []models.Message{
				{Role: "system", Content: prompt + "\n\nReturn ONLY valid JSON. No markdown, no explanation."},
			},
			MaxTokens:      1000,
			ResponseFormat: &openrouter.ResponseFormat{Type: "json_object"},
		}
		retryResp, retryErr := e.client.Chat(ctx, retryReq)
		if retryErr != nil {
			return nil, ErrMalformedScore
		}
		if retryErr := json.Unmarshal([]byte(retryResp.Choices[0].Message.Content), &score); retryErr != nil {
			return nil, ErrMalformedScore
		}
	}

	// Validate score range
	if score.Score < 1 {
		score.Score = 1
	}
	if score.Score > 10 {
		score.Score = 10
	}

	return &score, nil
}

// RunFollowUp processes a follow-up answer: streams LLM feedback, re-evaluates,
// and returns the updated score (which may include another follow-up).
func (e *Engine) RunFollowUp(ctx context.Context, question, answer string, furDepth int, callbacks UICallbacks) (*models.Score, error) {
	sanitized, err := ValidateAnswer(answer)
	if err != nil {
		return nil, err
	}

	if len(e.session.Turns) == 0 {
		return nil, fmt.Errorf("no active turn for follow-up")
	}
	turn := &e.session.Turns[len(e.session.Turns)-1]

	e.session.History = append(e.session.History, models.Message{Role: "user", Content: sanitized})
	e.session.History, _ = e.budget.TrimHistoryToFit(e.session.History)

	if callbacks.OnProcessing != nil {
		callbacks.OnProcessing()
	}

	req := openrouter.ChatRequest{
		Model:     e.client.Model(),
		Messages:  e.session.History,
		MaxTokens: 1024,
		Stream:    true,
	}

	stream, err := e.client.ChatStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("starting follow-up stream: %w", err)
	}

	var fullResponse strings.Builder
	for chunk := range stream {
		fullResponse.WriteString(chunk)
		if callbacks.OnStreaming != nil {
			callbacks.OnStreaming(chunk)
		}
	}

	responseText := strings.TrimSpace(fullResponse.String())
	e.session.History = append(e.session.History, models.Message{Role: "assistant", Content: responseText})

	// Build context from previous Q&A and all prior follow-ups
	context := fmt.Sprintf("ORIGINAL QUESTION: %s\nORIGINAL ANSWER: %s", turn.Question, turn.Answer)
	for i, fu := range turn.FollowUps {
		context += fmt.Sprintf("\nFOLLOW-UP %d: %s\nFOLLOW-UP ANSWER %d: %s", i+1, fu.Question, i+1, fu.Answer)
	}

	score, err := e.Evaluate(ctx, question, sanitized, furDepth, context)
	if err != nil {
		return nil, fmt.Errorf("evaluating follow-up: %w", err)
	}

	turn.FollowUps = append(turn.FollowUps, models.FollowUp{
		Question: question,
		Answer:   sanitized,
	})
	turn.Score = score

	e.session.UpdatedAt = time.Now()
	if err := e.store.SaveSession(e.session); err != nil {
		return nil, fmt.Errorf("saving session after follow-up: %w", err)
	}

	return score, nil
}

// Summarize generates an end-of-session summary.
func (e *Engine) Summarize(ctx context.Context) (*models.Summary, error) {
	if len(e.session.Turns) == 0 {
		return nil, fmt.Errorf("no turns to summarize")
	}

	prompt, err := RenderTemplate("summarize", SummarizeData{
		Turns: e.session.Turns,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering summarize template: %w", err)
	}

	req := openrouter.ChatRequest{
		Model: e.client.Model(),
		Messages: []models.Message{
			{Role: "system", Content: prompt},
		},
		MaxTokens:      1000,
		ResponseFormat: &openrouter.ResponseFormat{Type: "json_object"},
	}

	resp, err := e.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("summarize API call: %w", err)
	}

	e.session.TokensUsed += resp.Usage.TotalTokens

	var summary struct {
		Summary models.Summary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &summary); err != nil {
		return nil, fmt.Errorf("parsing summary JSON: %w", err)
	}

	e.session.MarkCompleted(summary.Summary)
	if err := e.store.SaveSession(e.session); err != nil {
		return nil, fmt.Errorf("saving session after summary: %w", err)
	}

	return &summary.Summary, nil
}

// generateHint asks the LLM for a hint about the current question.
func (e *Engine) generateHint(ctx context.Context) (string, error) {
	if len(e.session.Turns) == 0 && len(e.session.Questions) == 0 {
		return "", fmt.Errorf("no active question to hint about")
	}

	currentQ := ""
	if e.session.CurrentTurnIndex() < len(e.session.Questions) {
		currentQ = e.session.Questions[e.session.CurrentTurnIndex()]
	}

	prompt, err := RenderTemplate("hint", map[string]string{
		"Question": currentQ,
	})
	if err != nil {
		return "", fmt.Errorf("rendering hint template: %w", err)
	}

	req := openrouter.ChatRequest{
		Model: e.client.Model(),
		Messages: []models.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 300,
	}

	resp, err := e.client.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("hint API call: %w", err)
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// Model returns the model name configured on the client.
func (e *Engine) Model() string {
	return e.client.Model()
}

// Session returns the current session.
func (e *Engine) Session() *models.Session {
	return e.session
}
