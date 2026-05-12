package openrouter

import (
	"encoding/json"
	"testing"

	"github.com/absolutezero000/prep/internal/models"
	openrouterlib "github.com/revrost/go-openrouter"
)

func TestNewClient(t *testing.T) {
	client := NewClient("sk-or-test123456789012345678901234567890", ClientConfig{
		Model: "openai/gpt-4o",
	})
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.cfg.Model != "openai/gpt-4o" {
		t.Fatalf("expected model openai/gpt-4o, got %s", client.cfg.Model)
	}
}

func TestNewClient_WithConfig(t *testing.T) {
	client := NewClient("sk-or-test123456789012345678901234567890", ClientConfig{
		Model:          "mistralai/mistral-7b-instruct",
		FallbackModels: []string{"openai/gpt-4o-mini"},
	})
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if len(client.cfg.FallbackModels) != 1 {
		t.Fatalf("expected 1 fallback model, got %d", len(client.cfg.FallbackModels))
	}
}

func TestToMessageSlice(t *testing.T) {
	msgs := toMessageSlice("system", "You are a bot", "user", "Hello")
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[0].Content != "You are a bot" {
		t.Fatal("first message mismatch")
	}
	if msgs[1].Role != "user" || msgs[1].Content != "Hello" {
		t.Fatal("second message mismatch")
	}
}

func TestToMessageSlice_OddCount(t *testing.T) {
	msgs := toMessageSlice("system", "msg", "user")
	if msgs != nil {
		t.Fatal("expected nil for odd arguments")
	}
}

func TestMessageToLibrary(t *testing.T) {
	msg := models.Message{Role: "user", Content: "test"}
	libMsg := messageToLibrary(msg)
	if libMsg.Role != "user" {
		t.Fatalf("expected role user, got %s", libMsg.Role)
	}
	if libMsg.Content.Text != "test" {
		t.Fatalf("expected content 'test', got '%s'", libMsg.Content.Text)
	}
}

func TestRequestToLibrary(t *testing.T) {
	req := ChatRequest{
		Model: "openai/gpt-4o",
		Messages: []models.Message{
			{Role: "system", Content: "test"},
		},
		MaxTokens: 100,
	}

	libReq := req.toLibraryRequest()
	if libReq.Model != "openai/gpt-4o" {
		t.Fatalf("expected model 'openai/gpt-4o', got %s", libReq.Model)
	}
	if len(libReq.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(libReq.Messages))
	}
	if libReq.MaxTokens != 100 {
		t.Fatalf("expected MaxTokens 100, got %d", libReq.MaxTokens)
	}
	if libReq.Stream {
		t.Fatal("expected Stream to be false for non-streaming request")
	}
}

func TestRequestToLibrary_JSONFormat(t *testing.T) {
	req := ChatRequest{
		Model: "openai/gpt-4o",
		Messages: []models.Message{
			{Role: "user", Content: "Return JSON"},
		},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	libReq := req.toLibraryRequest()
	if libReq.ResponseFormat == nil {
		t.Fatal("expected non-nil ResponseFormat")
	}
	if string(libReq.ResponseFormat.Type) != "json_object" {
		t.Fatalf("expected json_object, got %s", libReq.ResponseFormat.Type)
	}
}

func TestRequestToLibrary_TextFormat(t *testing.T) {
	req := ChatRequest{
		Model: "openai/gpt-4o",
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
		},
		ResponseFormat: &ResponseFormat{Type: "text"},
	}

	libReq := req.toLibraryRequest()
	if libReq.ResponseFormat == nil {
		t.Fatal("expected non-nil ResponseFormat")
	}
	if string(libReq.ResponseFormat.Type) != "text" {
		t.Fatalf("expected text, got %s", libReq.ResponseFormat.Type)
	}
}

func TestRequestToLibrary_NoFormat(t *testing.T) {
	req := ChatRequest{
		Model: "openai/gpt-4o",
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	libReq := req.toLibraryRequest()
	if libReq.ResponseFormat != nil {
		t.Fatal("expected nil ResponseFormat")
	}
}

func TestRequestToLibrary_Streaming(t *testing.T) {
	req := ChatRequest{
		Model:  "openai/gpt-4o",
		Stream: true,
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	libReq := req.toLibraryRequest()
	if !libReq.Stream {
		t.Fatal("expected Stream to be true")
	}
}

func TestResponseFromLibrary(t *testing.T) {
	libResp := openrouterlib.ChatCompletionResponse{
		ID: "test-123",
		Choices: []openrouterlib.ChatCompletionChoice{
			{
				Index: 0,
				Message: openrouterlib.ChatCompletionMessage{
					Role: "assistant",
					Content: openrouterlib.Content{
						Text: "Hello!",
					},
				},
				FinishReason: "stop",
			},
		},
		Usage: &openrouterlib.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	resp := responseFromLibrary(libResp)
	if resp.ID != "test-123" {
		t.Fatalf("expected ID 'test-123', got '%s'", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Fatalf("expected content 'Hello!', got '%s'", resp.Choices[0].Message.Content)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Fatalf("expected PromptTokens 10, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Fatalf("expected CompletionTokens 5, got %d", resp.Usage.CompletionTokens)
	}
}

func TestResponseFromLibrary_Empty(t *testing.T) {
	libResp := openrouterlib.ChatCompletionResponse{
		ID:      "empty",
		Choices: nil,
		Usage:   nil,
	}

	resp := responseFromLibrary(libResp)
	if resp.ID != "empty" {
		t.Fatalf("expected ID 'empty', got '%s'", resp.ID)
	}
	if len(resp.Choices) != 0 {
		t.Fatalf("expected 0 choices, got %d", len(resp.Choices))
	}
}

func TestResponseFromLibrary_MultipleChoices(t *testing.T) {
	libResp := openrouterlib.ChatCompletionResponse{
		ID: "multi",
		Choices: []openrouterlib.ChatCompletionChoice{
			{
				Index: 0,
				Message: openrouterlib.ChatCompletionMessage{
					Role:    "assistant",
					Content: openrouterlib.Content{Text: "First"},
				},
				FinishReason: "stop",
			},
			{
				Index: 1,
				Message: openrouterlib.ChatCompletionMessage{
					Role:    "assistant",
					Content: openrouterlib.Content{Text: "Second"},
				},
				FinishReason: "stop",
			},
		},
	}

	resp := responseFromLibrary(libResp)
	if len(resp.Choices) != 2 {
		t.Fatalf("expected 2 choices, got %d", len(resp.Choices))
	}
}

func TestTryParseStreamContent(t *testing.T) {
	chunk := `{"id":"test","choices":[{"index":0,"delta":{"content":"Hello"}}],"object":"chat.completion.chunk"}`
	content, err := tryParseStreamContent([]byte(chunk))
	if err != nil {
		t.Fatal(err)
	}
	if content != "Hello" {
		t.Fatalf("expected 'Hello', got '%s'", content)
	}
}

func TestTryParseStreamContent_EmptyChoices(t *testing.T) {
	chunk := `{"id":"test","choices":[],"object":"chat.completion.chunk"}`
	content, err := tryParseStreamContent([]byte(chunk))
	if err != nil {
		t.Fatal(err)
	}
	if content != "" {
		t.Fatalf("expected empty content, got '%s'", content)
	}
}

func TestTryParseStreamContent_Malformed(t *testing.T) {
	_, err := tryParseStreamContent([]byte(`{invalid json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestErrorFunctions(t *testing.T) {
	// Verify typed errors work with errors.Is
	err := ErrUnauthorized
	if err.Error() != "invalid API key" {
		t.Fatalf("expected 'invalid API key', got '%s'", err.Error())
	}

	err = ErrRateLimited
	if err.Error() != "rate limit exceeded" {
		t.Fatalf("expected 'rate limit exceeded', got '%s'", err.Error())
	}

	err = ErrAllModelsFailed
	if err.Error() != "all models failed or unavailable" {
		t.Fatalf("expected 'all models failed or unavailable', got '%s'", err.Error())
	}

	err = ErrModelNotFound
	if err.Error() != "model not found" {
		t.Fatalf("expected 'model not found', got '%s'", err.Error())
	}
}

func TestExtractAPIKey(t *testing.T) {
	key := extractAPIKey(ErrUnauthorized)
	if key != "[REDACTED]" {
		t.Fatalf("expected [REDACTED], got '%s'", key)
	}
}

func TestJSONMarshaling(t *testing.T) {
	req := ChatRequest{
		Model: "test",
		Messages: []models.Message{
			{Role: "user", Content: "hello"},
		},
	}

	// Verify our request type serializes to JSON for the library
	libReq := req.toLibraryRequest()
	data, err := json.Marshal(libReq)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}

func TestIsAuthError(t *testing.T) {
	// isAuthError checks HTTP status codes via the library
	// This just verifies it doesn't panic on nil/normal errors
	if isAuthError(nil) {
		t.Log("isAuthError(nil) = true (expected with nil)")
	}
}

func TestModelInfoTypes(t *testing.T) {
	info := ModelInfo{
		ID:          "test/model",
		Name:        "Test Model",
		ContextSize: 8192,
	}
	if info.ID != "test/model" {
		t.Fatalf("expected 'test/model', got '%s'", info.ID)
	}
	if info.ContextSize != 8192 {
		t.Fatalf("expected 8192, got %d", info.ContextSize)
	}
}
