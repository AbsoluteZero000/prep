package interview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/absolutezero000/prep/internal/models"
	"github.com/absolutezero000/prep/internal/openrouter"
	"github.com/absolutezero000/prep/internal/resume"
	"github.com/absolutezero000/prep/internal/storage"
	"github.com/absolutezero000/prep/internal/tokens"
)

func TestNewSession(t *testing.T) {
	r := resume.ParseResult{
		RawText:       "test resume",
		WordCount:     2,
		TokenEstimate: 3,
		Hash:          "abc123",
	}

	sess := NewSession("Backend Engineer", models.ModeTechnical, models.DiffSenior, r, 5)
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	if sess.Role != "Backend Engineer" {
		t.Fatalf("expected role 'Backend Engineer', got '%s'", sess.Role)
	}
	if sess.Mode != models.ModeTechnical {
		t.Fatalf("expected ModeTechnical, got %s", sess.Mode)
	}
	if sess.Difficulty != models.DiffSenior {
		t.Fatalf("expected DiffSenior, got %s", sess.Difficulty)
	}
	if sess.NumQuestions != 5 {
		t.Fatalf("expected NumQuestions 5, got %d", sess.NumQuestions)
	}
	if sess.Status != models.StatusReady {
		t.Fatalf("expected StatusReady, got %s", sess.Status)
	}
	if sess.ID == "" {
		t.Fatal("expected non-empty session ID")
	}
}

func TestNewSession_ClampsQuestionCount(t *testing.T) {
	r := resume.ParseResult{Hash: "abc"}
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, r, 0)
	if sess.NumQuestions != 5 {
		t.Fatalf("expected default 5 questions, got %d", sess.NumQuestions)
	}

	sess = NewSession("Engineer", models.ModeMixed, models.DiffMid, r, 20)
	if sess.NumQuestions != 15 {
		t.Fatalf("expected max 15 questions, got %d", sess.NumQuestions)
	}
}

func TestParseAnswer(t *testing.T) {
	tests := []struct {
		input  string
		action string
	}{
		{"skip", "skip"},
		{"SKIP", "skip"},
		{"hint", "hint"},
		{"quit", "quit"},
		{"exit", "quit"},
		{"", "empty"},
		{"  ", "empty"},
		{"My answer", "answer"},
	}
	for _, tt := range tests {
		action, text := ParseAnswer(tt.input)
		if action != tt.action {
			t.Errorf("ParseAnswer(%q) action = %q, want %q", tt.input, action, tt.action)
		}
		if action == "answer" && text == "" {
			t.Errorf("ParseAnswer(%q) returned empty text for answer action", tt.input)
		}
	}
}

func TestValidateAnswer(t *testing.T) {
	// valid answer
	valid, err := ValidateAnswer("test answer")
	if err != nil {
		t.Fatal(err)
	}
	if valid != "test answer" {
		t.Fatalf("expected 'test answer', got '%s'", valid)
	}

	// empty answer
	_, err = ValidateAnswer("")
	if err != ErrEmptyAnswer {
		t.Fatalf("expected ErrEmptyAnswer, got %v", err)
	}

	// answer too long
	long := strings.Repeat("a", 4000)
	truncated, err := ValidateAnswer(long)
	if err != nil {
		t.Logf("truncation warning: %v", err)
	}
	if len(truncated) > 3000 {
		t.Fatalf("expected max 3000 chars, got %d", len(truncated))
	}
}

func TestTurnDuration(t *testing.T) {
	started := time.Now().Add(-time.Minute)
	d := TurnDuration(started)
	if d < 30*time.Second || d > 90*time.Second {
		t.Fatalf("expected ~1 minute duration, got %v", d)
	}
}

func TestNewSessionID(t *testing.T) {
	id1 := newSessionID()
	id2 := newSessionID()
	if id1 == "" || id2 == "" {
		t.Fatal("expected non-empty IDs")
	}
	if id1 == id2 {
		t.Fatalf("expected unique IDs, got same: %s", id1)
	}
	if len(id1) != 12 {
		t.Fatalf("expected 12-char ID, got %d: %s", len(id1), id1)
	}
}

func TestRenderTemplate(t *testing.T) {
	result, err := RenderTemplate("hint", map[string]string{"Question": "What is Go?"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "What is Go?") {
		t.Fatal("expected hint template to include the question")
	}
}

func TestRenderTemplate_System(t *testing.T) {
	result, err := RenderTemplate("system", SystemData{
		Resume:     "Senior Engineer",
		Role:       "Engineer",
		Mode:       "technical",
		Difficulty: "senior",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "Engineer") {
		t.Fatal("expected system template to include role")
	}
	if !strings.Contains(result, "technical") {
		t.Fatal("expected system template to include mode")
	}
}

func TestRenderTemplate_Unknown(t *testing.T) {
	_, err := RenderTemplate("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
}

func TestNewEngine(t *testing.T) {
	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model: "openai/gpt-4o",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 3)

	engine := NewEngine(client, store, sess, 800)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.Session() != sess {
		t.Fatal("engine session mismatch")
	}
}

func TestGenerateQuestions_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "gen-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "openai/gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": `{"questions": ["Q1", "Q2", "Q3"]}`,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     20,
				"completion_tokens": 10,
				"total_tokens":      30,
			},
		})
	}))
	defer srv.Close()

	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model:   "openai/gpt-4o",
		BaseURL: srv.URL + "/api/v1",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{
		RawText: "Experienced Go developer",
		Hash:    "abc",
	}, 3)

	engine := NewEngine(client, store, sess, 800)
	err := engine.GenerateQuestions(context.Background())
	if err != nil {
		t.Fatalf("GenerateQuestions failed: %v", err)
	}
	if len(sess.Questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(sess.Questions))
	}
}

func TestGenerateQuestions_Deduplicates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "gen-123",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": `{"questions": ["Q1", "Q1", "Q2"]}`,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"total_tokens": 10,
			},
		})
	}))
	defer srv.Close()

	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model:   "openai/gpt-4o",
		BaseURL: srv.URL + "/api/v1",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 5)

	engine := NewEngine(client, store, sess, 800)
	engine.GenerateQuestions(context.Background())
	if len(sess.Questions) != 2 {
		t.Fatalf("expected 2 unique questions, got %d: %v", len(sess.Questions), sess.Questions)
	}
}

func TestEvaluate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "eval-123",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": `{"score": 8, "strengths": ["Clear"], "gaps": ["Missing detail"], "ideal_answer_summary": "Good", "follow_up_warranted": false, "follow_up_question": ""}`,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"total_tokens": 15,
			},
		})
	}))
	defer srv.Close()

	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model:   "openai/gpt-4o",
		BaseURL: srv.URL + "/api/v1",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 3)
	sess.Questions = []string{"What is Go?"}
	sess.History = []models.Message{
		{Role: "system", Content: "system prompt"},
	}

	engine := NewEngine(client, store, sess, 800)

	score, err := engine.Evaluate(context.Background(), "What is Go?", "A language.", 0, "")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if score.Score != 8 {
		t.Fatalf("expected score 8, got %d", score.Score)
	}
	if len(score.Strengths) != 1 || score.Strengths[0] != "Clear" {
		t.Fatalf("expected strength 'Clear', got %v", score.Strengths)
	}
}

func TestEvaluate_ClampsScore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "eval-123",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": `{"score": 0, "strengths": [], "gaps": [], "ideal_answer_summary": "", "follow_up_warranted": false, "follow_up_question": ""}`,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{},
		})
	}))
	defer srv.Close()

	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model:   "openai/gpt-4o",
		BaseURL: srv.URL + "/api/v1",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 3)
	sess.Questions = []string{"Q1"}

	engine := NewEngine(client, store, sess, 800)
	score, _ := engine.Evaluate(context.Background(), "Q1", "A1", 0, "")
	if score.Score < 1 {
		t.Fatalf("expected score clamped to min 1, got %d", score.Score)
	}
}

func TestRunTurn_Skip(t *testing.T) {
	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model: "openai/gpt-4o",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 3)
	sess.Questions = []string{"Q1", "Q2", "Q3"}

	engine := NewEngine(client, store, sess, 800)
	budget := tokens.NewBudget(8192, 500, 100)
	engine.budget = budget

	turn, err := engine.RunTurn(context.Background(), "skip", UICallbacks{})
	if err != nil {
		t.Fatalf("RunTurn(skip) failed: %v", err)
	}
	if turn == nil {
		t.Fatal("expected non-nil turn")
	}
	if !turn.Skipped {
		t.Fatal("expected turn to be skipped")
	}
}

func TestRunTurn_Quit(t *testing.T) {
	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model: "openai/gpt-4o",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 3)
	sess.Questions = []string{"Q1"}

	engine := NewEngine(client, store, sess, 800)
	budget := tokens.NewBudget(8192, 500, 100)
	engine.budget = budget

	_, err := engine.RunTurn(context.Background(), "quit", UICallbacks{})
	if err != ErrUserQuit {
		t.Fatalf("expected ErrUserQuit, got %v", err)
	}
	if sess.Status != models.StatusAborted {
		t.Fatalf("expected session to be aborted, got %s", sess.Status)
	}
}

func TestRunTurn_Empty(t *testing.T) {
	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model: "openai/gpt-4o",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 3)
	sess.Questions = []string{"Q1"}

	engine := NewEngine(client, store, sess, 800)
	budget := tokens.NewBudget(8192, 500, 100)
	engine.budget = budget

	_, err := engine.RunTurn(context.Background(), "", UICallbacks{})
	if err != ErrEmptyAnswer {
		t.Fatalf("expected ErrEmptyAnswer, got %v", err)
	}
}

func TestRunTurn_Hint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "hint-123",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Think about concurrency",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{"total_tokens": 5},
		})
	}))
	defer srv.Close()

	client := openrouter.NewClient("sk-or-test123456789012345678901234567890", openrouter.ClientConfig{
		Model:   "openai/gpt-4o",
		BaseURL: srv.URL + "/api/v1",
	})
	store, _ := storage.NewStore(t.TempDir())
	sess := NewSession("Engineer", models.ModeMixed, models.DiffMid, resume.ParseResult{Hash: "abc"}, 3)
	sess.Questions = []string{"Q1"}

	engine := NewEngine(client, store, sess, 800)
	budget := tokens.NewBudget(8192, 500, 100)
	engine.budget = budget

	streamed := false
	callbacks := UICallbacks{
		OnStreaming: func(chunk string) {
			if chunk == "Think about concurrency" {
				streamed = true
			}
		},
	}

	turn, err := engine.RunTurn(context.Background(), "hint", callbacks)
	if err != nil {
		t.Fatalf("RunTurn(hint) failed: %v", err)
	}
	if turn != nil {
		t.Fatal("expected nil turn for hint (no advancement)")
	}
	if !streamed {
		t.Fatal("expected streaming callback for hint")
	}
}
