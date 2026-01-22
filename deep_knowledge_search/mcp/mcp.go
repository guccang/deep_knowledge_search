// Package mcp provides Model Context Protocol (MCP) tool management functionality.
package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ToolCall represents a function call from LLM
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents function call details
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// LLMTool represents a tool available to the LLM
type LLMTool struct {
	Type     string      `json:"type"`
	Function LLMFunction `json:"function"`
}

// LLMFunction represents the function definition for LLM
type LLMFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// MCPToolResponse represents the response from a tool call
type MCPToolResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
	Error   string      `json:"error,omitempty"`
}

// Tool callback function type
type ToolCallback func(arguments map[string]interface{}) MCPToolResponse

// Tool registry
var (
	toolRegistry = make(map[string]ToolCallback)
	toolDefs     = make(map[string]LLMTool)
	registryMu   sync.RWMutex
)

// RegisterTool registers a tool with its callback and definition
func RegisterTool(name string, tool LLMTool, callback ToolCallback) {
	registryMu.Lock()
	defer registryMu.Unlock()
	toolRegistry[name] = callback
	toolDefs[name] = tool
}

// GetAvailableLLMTools returns all registered tools in LLM format
func GetAvailableLLMTools() []LLMTool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	tools := make([]LLMTool, 0, len(toolDefs))
	for _, tool := range toolDefs {
		tools = append(tools, tool)
	}
	return tools
}

// CallMCPTool calls a registered MCP tool and returns the result
func CallMCPTool(toolName string, arguments map[string]interface{}) MCPToolResponse {
	registryMu.RLock()
	callback, exists := toolRegistry[toolName]
	registryMu.RUnlock()

	if !exists {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("tool '%s' not found", toolName),
		}
	}

	return callback(arguments)
}

// ParseToolArguments parses JSON arguments string to map
func ParseToolArguments(argsJSON string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	return args, nil
}

// Init initializes the MCP module and registers default tools
func Init() {
	RegisterDefaultTools()
	fmt.Println("[MCP] Initialized with", len(toolRegistry), "tools")
}
