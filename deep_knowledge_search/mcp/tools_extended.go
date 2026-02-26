package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// ============================================================================
// 扩展工具集 — fetchWebPage, readFile, listFiles
// ============================================================================

// RegisterExtendedTools 注册扩展工具
func RegisterExtendedTools() {
	// fetchWebPage — 抓取网页内容
	RegisterTool("fetchWebPage", LLMTool{
		Type: "function",
		Function: LLMFunction{
			Name:        "fetchWebPage",
			Description: "抓取指定URL的网页内容，返回纯文本。用于在线搜索和获取网络信息。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "要抓取的网页URL",
					},
				},
				"required": []string{"url"},
			},
		},
	}, fetchWebPageHandler)

	// readFile — 读取本地文件
	RegisterTool("readFile", LLMTool{
		Type: "function",
		Function: LLMFunction{
			Name:        "readFile",
			Description: "读取本地文件的内容。用于检查已保存的输出文件或读取配置。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "要读取的文件路径（相对于输出目录或绝对路径）",
					},
				},
				"required": []string{"path"},
			},
		},
	}, readFileHandler)

	// listFiles — 列出目录文件
	RegisterTool("listFiles", LLMTool{
		Type: "function",
		Function: LLMFunction{
			Name:        "listFiles",
			Description: "列出指定目录下的文件和子目录。用于预检阶段了解已有输出。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "要列出的目录路径（相对于输出目录或绝对路径）",
					},
				},
				"required": []string{"path"},
			},
		},
	}, listFilesHandler)
}

// ============================================================================
// fetchWebPage — 网页抓取
// ============================================================================

func fetchWebPageHandler(ctx context.Context, arguments map[string]interface{}) MCPToolResponse {
	url, ok := arguments["url"].(string)
	if !ok || url == "" {
		return MCPToolResponse{
			Success: false,
			Error:   "missing or invalid 'url' parameter",
		}
	}

	fmt.Printf("[MCP] Fetching web page: %s\n", url)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("create request failed: %v", err),
		}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; DeepKnowledgeSearch/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("fetch failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("HTTP error: %d", resp.StatusCode),
		}
	}

	// 读取 body（限制 1MB）
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("read body failed: %v", err),
		}
	}

	// 尝试提取纯文本
	text := extractTextFromHTML(string(body))

	// 截断过长的内容
	if utf8.RuneCountInString(text) > 5000 {
		runes := []rune(text)
		text = string(runes[:5000]) + "\n\n... (内容已截断)"
	}

	fmt.Printf("[MCP] Fetched %d chars from %s\n", utf8.RuneCountInString(text), url)

	return MCPToolResponse{
		Success: true,
		Result:  text,
	}
}

// extractTextFromHTML 从 HTML 提取纯文本
func extractTextFromHTML(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		// 如果解析失败，返回原始内容的简单清理版本
		return stripHTMLTags(htmlStr)
	}

	var sb strings.Builder
	extractText(doc, &sb)
	return strings.TrimSpace(sb.String())
}

func extractText(n *html.Node, sb *strings.Builder) {
	// 跳过 script 和 style 标签
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "noscript") {
		return
	}

	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
	}

	// 在块级元素之后添加换行
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6",
			"li", "tr", "blockquote", "pre", "article", "section":
			sb.WriteString("\n")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractText(c, sb)
	}
}

// stripHTMLTags 简单地去除 HTML 标签
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ============================================================================
// readFile — 读取文件
// ============================================================================

func readFileHandler(ctx context.Context, arguments map[string]interface{}) MCPToolResponse {
	filePath, ok := arguments["path"].(string)
	if !ok || filePath == "" {
		return MCPToolResponse{
			Success: false,
			Error:   "missing or invalid 'path' parameter",
		}
	}

	// 如果是相对路径，尝试相对于输出目录解析
	if !filepath.IsAbs(filePath) {
		outputDir, ok := ctx.Value(ContextKeyOutputPath).(string)
		if ok && outputDir != "" {
			filePath = filepath.Join(outputDir, filePath)
		}
	}

	fmt.Printf("[MCP] Reading file: %s\n", filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("read file failed: %v", err),
		}
	}

	content := string(data)

	// 截断过长的文件
	if utf8.RuneCountInString(content) > 8000 {
		runes := []rune(content)
		content = string(runes[:8000]) + "\n\n... (文件内容已截断)"
	}

	return MCPToolResponse{
		Success: true,
		Result:  content,
	}
}

// ============================================================================
// listFiles — 列出目录
// ============================================================================

func listFilesHandler(ctx context.Context, arguments map[string]interface{}) MCPToolResponse {
	dirPath, ok := arguments["path"].(string)
	if !ok || dirPath == "" {
		return MCPToolResponse{
			Success: false,
			Error:   "missing or invalid 'path' parameter",
		}
	}

	// 如果是相对路径，尝试相对于输出目录解析
	if !filepath.IsAbs(dirPath) {
		outputDir, ok := ctx.Value(ContextKeyOutputPath).(string)
		if ok && outputDir != "" {
			dirPath = filepath.Join(outputDir, dirPath)
		}
	}

	fmt.Printf("[MCP] Listing files: %s\n", dirPath)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return MCPToolResponse{
			Success: false,
			Error:   fmt.Sprintf("read directory failed: %v", err),
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("目录: %s\n\n", dirPath))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			sb.WriteString(fmt.Sprintf("📁 %s/\n", entry.Name()))
		} else {
			sb.WriteString(fmt.Sprintf("📄 %s (%s)\n", entry.Name(), formatSize(info.Size())))
		}
	}

	if len(entries) == 0 {
		sb.WriteString("(空目录)")
	}

	return MCPToolResponse{
		Success: true,
		Result:  sb.String(),
	}
}

// formatSize 格式化文件大小
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
