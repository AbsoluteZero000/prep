package tokens

import (
	"testing"

	"github.com/absolutezero000/prep/internal/models"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello world", 3}, // 2 * 1.35 = 2.7 → 3
		{"", 0},            // 0
		{"a b c d e", 7},   // 5 * 1.35 = 6.75 → 7
		{"one two three four five six seven eight nine ten", 14}, // 10 * 1.35 = 13.5 → 14
	}
	for _, tt := range tests {
		got := EstimateTokens(tt.input)
		if got != tt.expected {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestNewBudget(t *testing.T) {
	b := NewBudget(8192, 500, 1000)
	if b.ModelLimit != 8192 {
		t.Fatalf("expected ModelLimit 8192, got %d", b.ModelLimit)
	}
	if b.ResponseReserve != 1024 {
		t.Fatalf("expected ResponseReserve 1024, got %d", b.ResponseReserve)
	}
}

func TestAvailable(t *testing.T) {
	b := NewBudget(8192, 500, 1000)
	avail := b.Available()
	// 8192 - 500 - 1000 - 1024 = 5668
	if avail != 5668 {
		t.Fatalf("expected 5668 available, got %d", avail)
	}
}

func TestCanFit(t *testing.T) {
	b := NewBudget(8192, 500, 1000)
	if !b.CanFit(5000) {
		t.Error("expected 5000 to fit")
	}
	if b.CanFit(6000) {
		t.Error("expected 6000 not to fit")
	}
}

func TestGetModelLimit(t *testing.T) {
	if limit := GetModelLimit("openai/gpt-4o"); limit != 128000 {
		t.Fatalf("expected 128000, got %d", limit)
	}
	if limit := GetModelLimit("unknown/model"); limit != 8192 {
		t.Fatalf("expected 8192 default, got %d", limit)
	}
}

func TestTrimHistory_Basic(t *testing.T) {
	b := NewBudget(500, 50, 50) // 500 - 50 - 50 - 1024 = -624 available → very tight
	msgs := []models.Message{
		{Role: "system", Content: "You are an assistant."}, // 4 words ≈ 6 tokens ← kept
		{Role: "user", Content: "Hello"},                   // 1 word ≈ 2 tokens ← first user, kept
		{Role: "assistant", Content: "Hi there!"},          // 2 words ≈ 3 tokens ← may be dropped
		{Role: "user", Content: "What is Go?"},             // 3 words ≈ 5 tokens ← may be dropped
		{Role: "assistant", Content: "Go is a language."},  // 4 words ≈ 6 tokens ← may be dropped
	}

	result := b.TrimHistory(msgs)

	// should keep index 0 (system) and first user (index 1)
	if len(result) < 2 {
		t.Fatal("expected at least system and first user message")
	}
	if result[0].Role != "system" {
		t.Fatal("expected system message to be preserved")
	}
	if result[1].Role != "user" || result[1].Content != "Hello" {
		t.Fatal("expected first user message to be preserved")
	}
}

func TestTrimHistoryToFit_Exhausted(t *testing.T) {
	b := NewBudget(100, 50, 50) // 100 - 50 - 50 - 1024 = -1024 → impossible
	msgs := []models.Message{
		{Role: "system", Content: "x"},
		{Role: "user", Content: "y"},
	}

	_, err := b.TrimHistoryToFit(msgs)
	if err != ErrContextWindowExhausted {
		t.Fatalf("expected ErrContextWindowExhausted, got %v", err)
	}
}

func TestTrimHistory_Empty(t *testing.T) {
	b := NewBudget(8192, 0, 0)
	result := b.TrimHistory(nil)
	if result != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestTrimHistory_KeepsFirstUser(t *testing.T) {
	b := NewBudget(200, 50, 50) // 200 - 50 - 50 - 1024 = -924
	msgs := []models.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "assistant", Content: "Welcome! I'm ready."},
		{Role: "user", Content: "I am the candidate."}, // first user message
		{Role: "assistant", Content: "Great, let's start."},
		{Role: "user", Content: "My name is John."},
	}

	result := b.TrimHistory(msgs)

	// first user at index 2 must be kept
	found := false
	for _, m := range result {
		if m.Content == "I am the candidate." {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected first user message to be preserved")
	}
}
