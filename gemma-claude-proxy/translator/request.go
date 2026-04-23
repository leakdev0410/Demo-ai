package translator

import(
	"encoding/json"
	"fmt"
	"github.com/user/gemma-claude-proxy/types"
)

func TranslateRequest(anthropicReq *types.AnthropicRequest, targetModel string) (*types.OpenAIRequest, error) {
	openAIReq := &types.OpenAIRequest{
		Model:       targetModel,
		MaxTokens:   anthropicReq.MaxTokens,
		Temperature: anthropicReq.Temperature,
		Stream:      anthropicReq.Stream,
		Stop:        anthropicReq.StopSequences,
	}

	var messages []types.OpenAIMessage

	if anthropicReq.System != nil {
		var systemContent string
		switch sys := anthropicReq.System.(type) {
		case string:
			systemContent = sys
		case []interface{}:
			for _, blockIf := range sys {
				blockBytes, err := json.Marshal(blockIf)
				if err == nil {
					var block types.AnthropicContentBlock
					if err := json.Unmarshal(blockBytes, &block); err == nil && block.Type == "text" {
						systemContent += block.Text
					}
				}
			}
		}
		if systemContent != "" {
			messages = append(messages, types.OpenAIMessage{
				Role:    "system",
				Content: systemContent,
			})
		}
	}

	for _, msg := range anthropicReq.Messages {
		switch content := msg.Content.(type) {
		case string:
			messages = append(messages, types.OpenAIMessage{
				Role:    msg.Role,
				Content: content,
			})
		case []interface{}:
			// Complex message with blocks (e.g. text, tool_use, tool_result)
			// Anthropic content blocks need to be parsed
			var textContent string
			var toolCalls []types.OpenAIToolCall

			for _, blockIf := range content {
				blockBytes, err := json.Marshal(blockIf)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal content block: %w", err)
				}

				var block types.AnthropicContentBlock
				if err := json.Unmarshal(blockBytes, &block); err != nil {
					return nil, fmt.Errorf("failed to unmarshal content block: %w", err)
				}

				switch block.Type {
				case "text":
					textContent += block.Text
				case "tool_use":
					toolCall := types.OpenAIToolCall{
						ID:   block.ID,
						Type: "function",
						Function: types.OpenAIFunction{
							Name:      block.Name,
							Arguments: string(block.Input),
						},
					}
					toolCalls = append(toolCalls, toolCall)
				case "tool_result":
					// In OpenAI, tool results are separate messages with role="tool"
					// We might need to split this message if there are multiple tool results
					// or if it's mixed with other content.
					toolResultMsg := types.OpenAIMessage{
						Role:       "tool",
						Content:    block.Content,
						ToolCallID: block.ToolUseID,
					}
					messages = append(messages, toolResultMsg)
				}
			}

			// If there's text or tool_calls, add it as a message
			if textContent != "" || len(toolCalls) > 0 {
				oaiMsg := types.OpenAIMessage{
					Role:      msg.Role,
					Content:   textContent,
					ToolCalls: toolCalls,
				}
				messages = append(messages, oaiMsg)
			}
		case []types.AnthropicContentBlock:
			var textContent string
			var toolCalls []types.OpenAIToolCall

			for _, block := range content {
				switch block.Type {
				case "text":
					textContent += block.Text
				case "tool_use":
					toolCall := types.OpenAIToolCall{
						ID:   block.ID,
						Type: "function",
						Function: types.OpenAIFunction{
							Name:      block.Name,
							Arguments: string(block.Input),
						},
					}
					toolCalls = append(toolCalls, toolCall)
				case "tool_result":
					toolResultMsg := types.OpenAIMessage{
						Role:       "tool",
						Content:    block.Content,
						ToolCallID: block.ToolUseID,
					}
					messages = append(messages, toolResultMsg)
				}
			}

			if textContent != "" || len(toolCalls) > 0 {
				oaiMsg := types.OpenAIMessage{
					Role:      msg.Role,
					Content:   textContent,
					ToolCalls: toolCalls,
				}
				messages = append(messages, oaiMsg)
			}
		default:
			// Try to handle it as raw json unmarshalling directly to []AnthropicContentBlock
			contentBytes, err := json.Marshal(content)
			if err == nil {
				var blocks []types.AnthropicContentBlock
				if err := json.Unmarshal(contentBytes, &blocks); err == nil {
					var textContent string
					var toolCalls []types.OpenAIToolCall

					for _, block := range blocks {
						switch block.Type {
						case "text":
							textContent += block.Text
						case "tool_use":
							toolCall := types.OpenAIToolCall{
								ID:   block.ID,
								Type: "function",
								Function: types.OpenAIFunction{
									Name:      block.Name,
									Arguments: string(block.Input),
								},
							}
							toolCalls = append(toolCalls, toolCall)
						case "tool_result":
							toolResultMsg := types.OpenAIMessage{
								Role:       "tool",
								Content:    block.Content,
								ToolCallID: block.ToolUseID,
							}
							messages = append(messages, toolResultMsg)
						}
					}

					if textContent != "" || len(toolCalls) > 0 {
						oaiMsg := types.OpenAIMessage{
							Role:      msg.Role,
							Content:   textContent,
							ToolCalls: toolCalls,
						}
						messages = append(messages, oaiMsg)
					}
				}
			}
		}
	}

	openAIReq.Messages = messages

	// Translate tools
	if len(anthropicReq.Tools) > 0 {
		var tools []types.OpenAITool
		for _, anthropicTool := range anthropicReq.Tools {
			tool := types.OpenAITool{
				Type: "function",
				Function: types.OpenAIFunctionDef{
					Name:        anthropicTool.Name,
					Description: anthropicTool.Description,
					Parameters:  anthropicTool.InputSchema,
				},
			}
			tools = append(tools, tool)
		}
		openAIReq.Tools = tools
	}

	return openAIReq, nil
}
