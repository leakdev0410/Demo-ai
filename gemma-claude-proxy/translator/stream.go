package translator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/user/gemma-claude-proxy/types"
)

func TranslateStream(w http.ResponseWriter, r io.Reader, originalModel string) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("expected http.ResponseWriter to be an http.Flusher")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	scanner := bufio.NewScanner(r)

	// Send initial message_start event
	messageStart := types.AnthropicStreamEvent{
		Type: "message_start",
		Message: &types.AnthropicResponse{
			ID:      "msg_id_stream", // Provide a dummy ID or generate one
			Type:    "message",
			Role:    "assistant",
			Model:   originalModel,
			Content: []types.AnthropicContentBlock{},
			Usage: types.AnthropicUsage{
				InputTokens:  0,
				OutputTokens: 0,
			},
		},
	}
	if err := sendStreamEvent(w, messageStart, flusher); err != nil {
		return err
	}

	contentBlockIndex := 0
	isTextActive := false
	var currentToolCallID string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var oaiStreamResp types.OpenAIStreamResponse
		if err := json.Unmarshal([]byte(data), &oaiStreamResp); err != nil {
			fmt.Printf("Error unmarshaling stream chunk: %v\n", err)
			continue
		}

		if len(oaiStreamResp.Choices) == 0 {
			continue
		}

		choice := oaiStreamResp.Choices[0]
		delta := choice.Delta

		// Handle Text
		if delta.Content != "" {
			if !isTextActive {
				// Start a text block
				startEvent := types.AnthropicStreamEvent{
					Type: "content_block_start",
					Index: &contentBlockIndex,
					ContentBlock: &types.AnthropicContentBlock{
						Type: "text",
						Text: "", // Initial text is empty
					},
				}
				sendStreamEvent(w, startEvent, flusher)
				isTextActive = true
			}

			// Send text delta
			deltaEvent := types.AnthropicStreamEvent{
				Type: "content_block_delta",
				Index: &contentBlockIndex,
				Delta: &types.AnthropicDelta{
					Type: "text_delta",
					Text: delta.Content,
				},
			}
			sendStreamEvent(w, deltaEvent, flusher)
		}

		// Handle Tool Calls
		if len(delta.ToolCalls) > 0 {
			for _, toolCall := range delta.ToolCalls {
				if toolCall.ID != "" && toolCall.ID != currentToolCallID {
					// Finish previous block (text or previous tool call)
					if isTextActive || currentToolCallID != "" {
						stopEvent := types.AnthropicStreamEvent{
							Type: "content_block_stop",
							Index: &contentBlockIndex,
						}
						sendStreamEvent(w, stopEvent, flusher)
						contentBlockIndex++
						isTextActive = false
					}

					currentToolCallID = toolCall.ID
					// Start new tool_use block
					startEvent := types.AnthropicStreamEvent{
						Type: "content_block_start",
						Index: &contentBlockIndex,
						ContentBlock: &types.AnthropicContentBlock{
							Type: "tool_use",
							ID:   toolCall.ID,
							Name: toolCall.Function.Name,
						},
					}
					sendStreamEvent(w, startEvent, flusher)
				}

				// Send input JSON delta
				if toolCall.Function.Arguments != "" {
					deltaEvent := types.AnthropicStreamEvent{
						Type: "content_block_delta",
						Index: &contentBlockIndex,
						Delta: &types.AnthropicDelta{
							Type:        "input_json_delta",
							PartialJson: toolCall.Function.Arguments,
						},
					}
					sendStreamEvent(w, deltaEvent, flusher)
				}
			}
		}

		// Handle Finish Reason
		if choice.FinishReason != nil {
			// Stop current block
			stopEvent := types.AnthropicStreamEvent{
				Type: "content_block_stop",
				Index: &contentBlockIndex,
			}
			sendStreamEvent(w, stopEvent, flusher)

			stopReason := "end_turn"
			if *choice.FinishReason == "tool_calls" {
				stopReason = "tool_use"
			} else if *choice.FinishReason == "length" {
				stopReason = "max_tokens"
			}

			// Send message_delta with stop_reason
			messageDeltaEvent := types.AnthropicStreamEvent{
				Type: "message_delta",
				Delta: &types.AnthropicDelta{
					StopReason: stopReason,
				},
			}
			sendStreamEvent(w, messageDeltaEvent, flusher)

			// Send message_stop
			messageStopEvent := types.AnthropicStreamEvent{
				Type: "message_stop",
			}
			sendStreamEvent(w, messageStopEvent, flusher)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %v", err)
	}

	return nil
}

func sendStreamEvent(w io.Writer, event types.AnthropicStreamEvent, flusher http.Flusher) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Anthropic SSE format
	var buf bytes.Buffer
	buf.WriteString("event: " + event.Type + "\n")
	buf.WriteString("data: " + string(eventBytes) + "\n\n")

	_, err = w.Write(buf.Bytes())
	if err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
