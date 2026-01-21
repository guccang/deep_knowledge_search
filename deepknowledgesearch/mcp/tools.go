package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 输出目录配置
var (
	baseOutputDir        = "output"
	docSubDir            = "doc" // 文档子目录
	currentTaskOutputDir = ""
	currentNodePath      = "" // 当前节点路径（用于树形目录结构）
	outputDirMu          sync.RWMutex
)

// SetTaskOutputDir 设置当前任务的输出目录
func SetTaskOutputDir(taskFolder string) {
	outputDirMu.Lock()
	defer outputDirMu.Unlock()
	currentTaskOutputDir = taskFolder
	currentNodePath = "" // 重置节点路径
}

// SetNodePath 设置当前节点路径（用于树形目录结构）
func SetNodePath(path string) {
	outputDirMu.Lock()
	defer outputDirMu.Unlock()
	currentNodePath = path
}

// GetTaskRootDir 获取任务根目录（不包含doc/和节点路径）
func GetTaskRootDir() string {
	outputDirMu.RLock()
	defer outputDirMu.RUnlock()
	if currentTaskOutputDir != "" {
		return filepath.Join(baseOutputDir, currentTaskOutputDir)
	}
	return ""
}

// GetCurrentOutputDir 获取当前输出目录（包含doc/和节点路径）
func GetCurrentOutputDir() string {
	outputDirMu.RLock()
	defer outputDirMu.RUnlock()
	if currentTaskOutputDir != "" {
		basePath := filepath.Join(baseOutputDir, currentTaskOutputDir, docSubDir)
		if currentNodePath != "" {
			return filepath.Join(basePath, currentNodePath)
		}
		return basePath
	}
	return baseOutputDir
}

// ClearTaskOutputDir 清除任务输出目录设置
func ClearTaskOutputDir() {
	outputDirMu.Lock()
	defer outputDirMu.Unlock()
	currentTaskOutputDir = ""
	currentNodePath = ""
}

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
func saveToDiskHandler(arguments map[string]interface{}) MCPToolResponse {
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

	// 使用当前任务的输出目录
	outputDir := GetCurrentOutputDir()
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
