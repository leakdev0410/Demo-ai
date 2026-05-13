package translator

import(
	"encoding/json"
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
				if blockMap, ok := blockIf.(map[string]interface{}); ok {
					if blockType, _ := blockMap["type"].(string); blockType == "text" {
						if text, ok := blockMap["text"].(string); ok {
							systemContent += text
						}
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
				if blockMap, ok := blockIf.(map[string]interface{}); ok {
					blockType, _ := blockMap["type"].(string)
					switch blockType {
					case "text":
						if text, ok := blockMap["text"].(string); ok {
							textContent += text
						}
					case "tool_use":
						id, _ := blockMap["id"].(string)
						name, _ := blockMap["name"].(string)
						var inputStr string
						if input, ok := blockMap["input"]; ok {
							inputBytes, _ := json.Marshal(input)
							inputStr = string(inputBytes)
						}

						toolCall := types.OpenAIToolCall{
							ID:   id,
							Type: "function",
							Function: types.OpenAIFunction{
								Name:      name,
								Arguments: inputStr,
							},
						}
						toolCalls = append(toolCalls, toolCall)
					case "tool_result":
						contentStr, _ := blockMap["content"].(string)
						toolUseID, _ := blockMap["tool_use_id"].(string)

						toolResultMsg := types.OpenAIMessage{
							Role:       "tool",
							Content:    contentStr,
							ToolCallID: toolUseID,
						}
						messages = append(messages, toolResultMsg)
					}
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
