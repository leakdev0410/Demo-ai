package translator

import (
	"encoding/json"
	"fmt"
	"github.com/user/gemma-claude-proxy/types"
)

func TranslateResponse(openAIResp *types.OpenAIResponse, originalModel string) (*types.AnthropicResponse, error) {
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	choice := openAIResp.Choices[0]
	message := choice.Message

	// Pre-allocate contentBlocks
	capacity := 0
	if message.Content != "" {
		capacity++
	}
	capacity += len(message.ToolCalls)

	contentBlocks := make([]types.AnthropicContentBlock, 0, capacity)

	// Handle text content
	if message.Content != "" {
		contentBlocks = append(contentBlocks, types.AnthropicContentBlock{
			Type: "text",
			Text: message.Content,
		})
	}

	// Handle tool calls
	for _, toolCall := range message.ToolCalls {
		if toolCall.Type == "function" {
			// Try to unmarshal the arguments, Anthropic expects it as an object
			var input json.RawMessage
			if toolCall.Function.Arguments != "" {
				// Avoid string -> []byte -> string conversion during marshal later by casting directly
				// Actually json.RawMessage is just []byte
				input = []byte(toolCall.Function.Arguments)
			} else {
				input = []byte("{}")
			}

			contentBlocks = append(contentBlocks, types.AnthropicContentBlock{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: input,
			})
		}
	}

	// Map stop reason
	stopReason := "end_turn"
	if choice.FinishReason == "tool_calls" {
		stopReason = "tool_use"
	} else if choice.FinishReason == "length" {
		stopReason = "max_tokens"
	}

	anthropicResp := &types.AnthropicResponse{
		ID:    openAIResp.ID,
		Type:  "message",
		Role:  "assistant", // Anthropic messages response role is always assistant
		Model: originalModel,
		Content: contentBlocks,
		StopReason: stopReason,
		Usage: types.AnthropicUsage{
			InputTokens:  openAIResp.Usage.PromptTokens,
			OutputTokens: openAIResp.Usage.CompletionTokens,
		},
	}

	return anthropicResp, nil
}
