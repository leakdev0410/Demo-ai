package translator

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/gemma-claude-proxy/types"
)

func BenchmarkTranslateRequest(b *testing.B) {
	req := &types.AnthropicRequest{
		Model: "claude-3",
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "Hello world!",
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TranslateRequest(req, "gemma-4-31b-it")
	}
}

func BenchmarkTranslateRequestComplex(b *testing.B) {
	blockJSON := `[{"type":"text","text":"Let me help you with that."},{"type":"tool_use","id":"tool123","name":"get_weather","input":{"location":"San Francisco"}}]`
	var blocks interface{}
	json.Unmarshal([]byte(blockJSON), &blocks)

	req := &types.AnthropicRequest{
		Model: "claude-3",
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: blocks,
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TranslateRequest(req, "gemma-4-31b-it")
	}
}

func BenchmarkTranslateResponse(b *testing.B) {
	resp := &types.OpenAIResponse{
		ID:      "test_id",
		Model:   "gemma-4-31b-it",
		Choices: []types.OpenAIChoice{
			{
				Message: types.OpenAIMessage{
					Role:    "assistant",
					Content: "Hello, this is a test response.",
				},
				FinishReason: "stop",
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TranslateResponse(resp, "claude-3")
	}
}

func BenchmarkTranslateStream(b *testing.B) {
	streamData := []string{
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: [DONE]`,
	}
	streamStr := strings.Join(streamData, "\n\n") + "\n\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(streamStr)
		recorder := httptest.NewRecorder()
		writer := &mockResponseWriter{recorder}
		TranslateStream(writer, io.NopCloser(reader), "claude-3")
	}
}
