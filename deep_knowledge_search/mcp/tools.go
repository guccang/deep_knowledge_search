package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RegisterDefaultTools registers the default set of tools
func RegisterDefaultTools() {
	// Register saveToDisk tool
	RegisterTool("saveToDisk", LLMTool{
		Type: "function",
		Function: LLMFunction{
			Name:        "saveToDisk",
			Description: "将内容保存到本地文件。用于保存LLM生成的数据、搜索结果或任何需要持久化的内容。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "文件标题，将作为文件名的一部分",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "要保存的内容",
					},
				},
				"required": []string{"title", "content"},
			},
		},
	}, saveToDiskHandler)
}

// saveToDiskHandler handles the saveToDisk tool call
func saveToDiskHandler(ctx context.Context, arguments map[string]interface{}) MCPToolResponse {
	title, ok := arguments["title"].(string)
	if !ok || title == "" {
		return MCPToolResponse{
			Success: false,
			Error:   "missing or invalid 'title' parameter",
		}
	}

	content, ok := arguments["content"].(string)
	if !ok {
		return MCPToolResponse{
			Success: false,
			Error:   "missing or invalid 'content' parameter",
		}
	}

	// 从 Context 获取输出目录
	outputDir, ok := ctx.Value(ContextKeyOutputPath).(string)
	if !ok || outputDir == "" {
		return MCPToolResponse{
			Success: false,
			Error:   "output directory not set in context",
		}
	}

	// 确保目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create output directory: %v", err),
		}
	}

	// Generate filename with timestamp
	sanitizedTitle := sanitizeFilename(title)
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.md", sanitizedTitle, timestamp)
	filepath := filepath.Join(outputDir, filename)

	// Write content to file
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}
	}

	fmt.Printf("[MCP] Saved to file: %s\n", filepath)

	return MCPToolResponse{
		Success: true,
		Result:  fmt.Sprintf("内容已保存到文件: %s", filepath),
	}
}

// sanitizeFilename removes or replaces invalid characters in filename
func sanitizeFilename(name string) string {
	// Replace invalid characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}

	return result
}
