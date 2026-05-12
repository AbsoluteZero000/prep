package openrouter

import (
	"encoding/json"
	"fmt"

	"github.com/absolutezero000/prep/internal/models"
	openrouterlib "github.com/revrost/go-openrouter"
)

// ClientConfig configures the OpenRouter API client.
type ClientConfig struct {
	Model          string
	FallbackModels []string
	Timeout        int
	BaseURL        string
}

// ChatRequest represents a request to the chat completion API.
type ChatRequest struct {
	Model          string
	Messages       []models.Message
	MaxTokens      int
	Stream         bool
	ResponseFormat *ResponseFormat
}

// ResponseFormat controls the output format of the response.
type ResponseFormat struct {
	Type string
}

// ChatResponse represents a response from the chat completion API.
type ChatResponse struct {
	ID      string
	Choices []Choice
	Usage   Usage
}

// Choice represents a single completion choice.
type Choice struct {
	Message      models.Message
	FinishReason string
	Index        int
}

// Usage reports token usage for a request.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

func (r ChatRequest) toLibraryRequest() openrouterlib.ChatCompletionRequest {
	libMsgs := make([]openrouterlib.ChatCompletionMessage, len(r.Messages))
	for i, msg := range r.Messages {
		libMsgs[i] = messageToLibrary(msg)
	}

	libReq := openrouterlib.ChatCompletionRequest{
		Model:     r.Model,
		Messages:  libMsgs,
		MaxTokens: r.MaxTokens,
		Stream:    r.Stream,
	}

	if r.ResponseFormat != nil {
		formatType := openrouterlib.ChatCompletionResponseFormatTypeText
		switch r.ResponseFormat.Type {
		case "json_object":
			formatType = openrouterlib.ChatCompletionResponseFormatTypeJSONObject
		}
		libReq.ResponseFormat = &openrouterlib.ChatCompletionResponseFormat{
			Type: formatType,
		}
	}

	return libReq
}

func messageToLibrary(msg models.Message) openrouterlib.ChatCompletionMessage {
	return openrouterlib.ChatCompletionMessage{
		Role: msg.Role,
		Content: openrouterlib.Content{
			Text: msg.Content,
		},
	}
}

func responseFromLibrary(libResp openrouterlib.ChatCompletionResponse) ChatResponse {
	resp := ChatResponse{
		ID: libResp.ID,
	}

	if libResp.Usage != nil {
		resp.Usage = Usage{
			PromptTokens:     libResp.Usage.PromptTokens,
			CompletionTokens: libResp.Usage.CompletionTokens,
			TotalTokens:      libResp.Usage.TotalTokens,
		}
	}

	resp.Choices = make([]Choice, len(libResp.Choices))
	for i, ch := range libResp.Choices {
		resp.Choices[i] = Choice{
			Index:        ch.Index,
			FinishReason: string(ch.FinishReason),
			Message: models.Message{
				Role:    ch.Message.Role,
				Content: ch.Message.Content.Text,
			},
		}
	}

	return resp
}

// isAuthError returns true if the error is an authentication error (401).
func isAuthError(err error) bool {
	return openrouterlib.IsHTTPStatus(err, 401)
}

// isNotFoundError returns true if the error is a 404.
func isNotFoundError(err error) bool {
	return openrouterlib.IsHTTPStatus(err, 404)
}

// isRateLimitError returns true if the error is a 429.
func isRateLimitError(err error) bool {
	return openrouterlib.IsHTTPStatus(err, 429)
}

// extractAPIKey safely extracts the API key from error messages for redaction.
// This is a no-op helper to prevent accidental key leakage.
func extractAPIKey(err error) string {
	return "[REDACTED]"
}

// tryParseStreamContent attempts to extract text content from a stream response chunk.
func tryParseStreamContent(data []byte) (string, error) {
	var chunk openrouterlib.ChatCompletionStreamResponse
	if err := json.Unmarshal(data, &chunk); err != nil {
		return "", fmt.Errorf("parse stream chunk: %w", err)
	}
	if len(chunk.Choices) == 0 {
		return "", nil
	}
	return chunk.Choices[0].Delta.Content, nil
}
