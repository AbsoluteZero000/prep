// Package openrouter provides a client adapter for the OpenRouter API.
package openrouter

import (
	"context"
	"errors"
	"fmt"

	"github.com/absolutezero000/prep/internal/models"
	openrouterlib "github.com/revrost/go-openrouter"
)

var (
	ErrUnauthorized      = errors.New("invalid API key")
	ErrRateLimited       = errors.New("rate limit exceeded")
	ErrAllModelsFailed   = errors.New("all models failed or unavailable")
	ErrMalformedResponse = errors.New("malformed response from API")
	ErrContextExceeded   = errors.New("context window exceeded")
	ErrModelNotFound     = errors.New("model not found")
	ErrStreamInterrupted = errors.New("stream disconnected before completion")
)

// Client wraps the OpenRouter API client.
type Client struct {
	inner *openrouterlib.Client
	cfg   ClientConfig
}

// NewClient creates a new OpenRouter client adapter.
func NewClient(apiKey string, cfg ClientConfig) *Client {
	libCfg := openrouterlib.DefaultConfig(apiKey)
	libCfg.HttpReferer = "https://github.com/absolutezero000/prep"
	libCfg.XTitle = "prep-cli"
	if cfg.BaseURL != "" {
		libCfg.BaseURL = cfg.BaseURL
	}

	inner := openrouterlib.NewClientWithConfig(*libCfg)

	return &Client{
		inner: inner,
		cfg:   cfg,
	}
}

// Chat sends a chat completion request and returns the response.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.cfg.Model
	}

	libReq := req.toLibraryRequest()
	libReq.Stream = false

	resp, err := c.inner.CreateChatCompletion(ctx, libReq)
	if err != nil {
		if isAuthError(err) {
			return nil, ErrUnauthorized
		}
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", ErrModelNotFound, req.Model)
		}
		if isRateLimitError(err) {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, ErrMalformedResponse
	}

	result := responseFromLibrary(resp)
	return &result, nil
}

// ChatWithFallback sends a chat request with model fallback on failure.
func (c *Client) ChatWithFallback(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.cfg.Model
	}

	libReq := req.toLibraryRequest()
	libReq.Stream = false

	fallbackModels := c.cfg.FallbackModels
	if len(fallbackModels) == 0 {
		fallbackModels = append(fallbackModels, c.cfg.Model)
	}

	resp, err := c.inner.CreateChatCompletionWithFallback(ctx, libReq, fallbackModels...)
	if err != nil {
		if isAuthError(err) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("all models failed: %w", ErrAllModelsFailed)
	}

	if len(resp.Choices) == 0 {
		return nil, ErrMalformedResponse
	}

	result := responseFromLibrary(resp)
	return &result, nil
}

// ChatStream sends a streaming chat completion request.
// The returned channel delivers text deltas and is closed when the stream completes.
func (c *Client) ChatStream(ctx context.Context, req ChatRequest) (<-chan string, error) {
	if req.Model == "" {
		req.Model = c.cfg.Model
	}

	libReq := req.toLibraryRequest()
	libReq.Stream = true

	stream, err := c.inner.CreateChatCompletionStream(ctx, libReq)
	if err != nil {
		if isAuthError(err) {
			return nil, ErrUnauthorized
		}
		if isRateLimitError(err) {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("stream creation failed: %w", err)
	}

	out := make(chan string, 64)

	go func() {
		defer close(out)
		defer stream.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				chunk, err := stream.Recv()
				if err != nil {
					return
				}
				for _, choice := range chunk.Choices {
					if choice.Delta.Content != "" {
						select {
						case out <- choice.Delta.Content:
						default:
							// Drop if consumer is slow
						}
					}
				}
			}
		}
	}()

	return out, nil
}

// ValidateModel checks whether the given model ID is available via OpenRouter.
func (c *Client) ValidateModel(ctx context.Context, model string) error {
	models, err := c.inner.ListModels(ctx)
	if err != nil {
		// If API call fails, skip validation (don't block the user)
		return nil
	}
	for _, m := range models {
		if m.ID == model {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrModelNotFound, model)
}

// ModelInfo holds metadata about an OpenRouter model.
type ModelInfo struct {
	ID          string
	Name        string
	ContextSize int64
}

// ListModels returns all available models from OpenRouter.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	models, err := c.inner.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing models: %w", err)
	}

	result := make([]ModelInfo, len(models))
	for i, m := range models {
		ctxSize := int64(0)
		if m.ContextLength != nil {
			ctxSize = *m.ContextLength
		}
		result[i] = ModelInfo{
			ID:          m.ID,
			Name:        m.Name,
			ContextSize: ctxSize,
		}
	}
	return result, nil
}

// toMessageSlice creates models.Message slice from role-content pairs (for testing).
func toMessageSlice(pairs ...string) []models.Message {
	if len(pairs)%2 != 0 {
		return nil
	}
	msgs := make([]models.Message, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		msgs[i/2] = models.Message{Role: pairs[i], Content: pairs[i+1]}
	}
	return msgs
}

// Model returns the configured model name.
func (c *Client) Model() string {
	return c.cfg.Model
}
