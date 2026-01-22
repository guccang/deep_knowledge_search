package llm

import (
	"bytes"
	"deepknowledgesearch/mcp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SendSyncLLMRequest sends a synchronous LLM request with tool calling support
func SendSyncLLMRequest(messages []Message) (string, error) {
	config := GetConfig()
	if config.APIKey == "" {
		return "", fmt.Errorf("LLM API key not configured")
	}

	// Get available MCP tools
	availableTools := mcp.GetAvailableLLMTools()
	fmt.Printf("[LLM] Available tools: %d\n", len(availableTools))

	// Keep track of messages
	currentMessages := make([]Message, len(messages))
	copy(currentMessages, messages)

	// Tool calling loop
	maxIterations := 10
	var finalResponse string

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Convert messages to API format
		apiMessages := convertMessagesToAPI(currentMessages)

		// Build request body
		requestBody := map[string]interface{}{
			"model":       config.Model,
			"messages":    apiMessages,
			"tools":       availableTools,
			"temperature": config.Temperature,
			"stream":      false,
		}

		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return "", fmt.Errorf("marshal request failed: %w", err)
		}

		fmt.Printf("[LLM] Sending request (iteration %d)...\n", iteration+1)

		// Create HTTP request
		req, err := http.NewRequest("POST", config.BaseURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("create request failed: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+config.APIKey)

		// Send request (1 hour timeout)
		client := &http.Client{Timeout: 3600 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("send request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read response failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("LLM API error: %d, body: %s", resp.StatusCode, string(body))
		}

		// Parse response
		var llmResp LLMResponse
		if err := json.Unmarshal(body, &llmResp); err != nil {
			return "", fmt.Errorf("parse response failed: %w, body: %s", err, string(body))
		}

		if len(llmResp.Choices) == 0 {
			return "", fmt.Errorf("empty response from LLM")
		}

		choice := llmResp.Choices[0]
		toolCalls := choice.Message.ToolCalls

		// If no tool calls, return content
		if len(toolCalls) == 0 {
			finalResponse = choice.Message.Content
			fmt.Printf("[LLM] Response received (no tool calls)\n")
			break
		}

		// Process tool calls
		fmt.Printf("[LLM] Processing %d tool call(s)...\n", len(toolCalls))

		// Add assistant message (with tool calls) to history
		assistantMsg := Message{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		}
		currentMessages = append(currentMessages, assistantMsg)

		// Execute each tool call
		for _, toolCall := range toolCalls {
			toolName := toolCall.Function.Name
			toolArgs := toolCall.Function.Arguments

			fmt.Printf("[LLM] Calling tool: %s\n", toolName)

			// Parse tool arguments
			parsedArgs, err := mcp.ParseToolArguments(toolArgs)
			if err != nil {
				parsedArgs = make(map[string]interface{})
			}

			// Call tool
			result := mcp.CallMCPTool(toolName, parsedArgs)

			// Add tool result to messages
			toolResult := fmt.Sprintf("%v", result.Result)
			if !result.Success {
				toolResult = "Error: " + result.Error
			}

			toolMsg := Message{
				Role:       "tool",
				ToolCallId: toolCall.ID,
				Content:    toolResult,
			}
			currentMessages = append(currentMessages, toolMsg)
		}

		// If last iteration, return what we have
		if iteration == maxIterations-1 {
			finalResponse = "工具调用已完成（达到最大迭代限制）"
		}
	}

	return finalResponse, nil
}

// convertMessagesToAPI converts Message slice to API format
func convertMessagesToAPI(messages []Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.ToolCallId != "" {
			m["tool_call_id"] = msg.ToolCallId
		}
		if len(msg.ToolCalls) > 0 {
			m["tool_calls"] = msg.ToolCalls
		}
		result[i] = m
	}
	return result
}

// Init initializes the LLM module
func Init() error {
	return InitConfig()
}
