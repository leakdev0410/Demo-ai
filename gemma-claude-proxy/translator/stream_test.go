package translator

import (
	"bytes"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockResponseWriter struct {
	*httptest.ResponseRecorder
}

func (m *mockResponseWriter) Flush() {}

func TestTranslateStream(t *testing.T) {
	// Simulated OpenAI stream chunks
	streamChunks := []string{
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma-4-31b-it","choices":[{"index":0,"delta":{"role":"assistant","content":"Sure"},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma-4-31b-it","choices":[{"index":0,"delta":{"content":", let me "},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma-4-31b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma-4-31b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"location\":"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma-4-31b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Paris\"}"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gemma-4-31b-it","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	}

	mockOpenAIResponse := strings.Join(streamChunks, "\n\n") + "\n\n"

	recorder := httptest.NewRecorder()
	mockWriter := &mockResponseWriter{recorder}

	err := TranslateStream(mockWriter, io.NopCloser(bytes.NewBufferString(mockOpenAIResponse)), "gemma-4-31b-it")
	if err != nil {
		t.Fatalf("TranslateStream failed: %v", err)
	}

	output := recorder.Body.String()

	expectedSubstrings := []string{
		"event: message_start",
		"event: content_block_start",
		`"type":"text"`,
		"event: content_block_delta",
		`"text":"Sure"`,
		`"text":", let me "`,
		"event: content_block_stop",
		"event: content_block_start",
		`"type":"tool_use"`,
		`"name":"get_weather"`,
		"event: content_block_delta",
		`"partial_json":"{\"location\":"`,
		`"partial_json":"\"Paris\"}"`,
		"event: content_block_stop",
		"event: message_delta",
		`"stop_reason":"tool_use"`,
		"event: message_stop",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("Expected output to contain '%s', but it did not.\nFull output:\n%s", sub, output)
		}
	}
}
