package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// ============================================================================
// 消息压缩 (Compaction)
// ============================================================================
// 参考 Anthropic 文章 "Effective Harnesses for Long-Running Agents"
// 当消息列表过长时，将旧消息压缩为摘要，保留最近的对话上下文

// CompactionConfig 压缩配置
type CompactionConfig struct {
	// MaxChars 触发压缩的字符数阈值（粗略对应 token 数的 ~4 倍）
	MaxChars int
	// KeepRecentCount 保留最近的消息数量（不压缩）
	KeepRecentCount int
}

// DefaultCompactionConfig 默认压缩配置
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		MaxChars:        60000, // ~15K tokens
		KeepRecentCount: 8,     // 保留最近 8 条消息
	}
}

// EstimateMessageChars 估算消息列表的总字符数
func EstimateMessageChars(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += utf8.RuneCountInString(msg.Content)
		// 工具调用的参数也要计算
		for _, tc := range msg.ToolCalls {
			total += utf8.RuneCountInString(tc.Function.Arguments)
			total += utf8.RuneCountInString(tc.Function.Name)
		}
	}
	return total
}

// CompactMessages 智能压缩消息列表
// 当消息总字符数超过阈值时，保留 system 消息和最近 N 条消息，
// 将中间的旧消息压缩为一条摘要消息
func CompactMessages(ctx context.Context, messages []Message, cfg CompactionConfig) []Message {
	totalChars := EstimateMessageChars(messages)

	// 未超过阈值，不需要压缩
	if totalChars < cfg.MaxChars {
		return messages
	}

	// 至少需要 system + 2 条消息才有压缩意义
	if len(messages) <= cfg.KeepRecentCount+1 {
		return messages
	}

	fmt.Printf("[Compaction] 消息总字符数 %d 超过阈值 %d，开始压缩...\n", totalChars, cfg.MaxChars)

	// 分离消息
	systemMsg := messages[0] // 第一条是 system 消息

	// 找到安全分割点：确保不会切断 assistant+tool 的配对
	// tool 消息必须紧跟在包含 tool_calls 的 assistant 消息之后
	keepStart := len(messages) - cfg.KeepRecentCount
	if keepStart < 2 {
		keepStart = 2
	}
	// 向前调整到安全分割点：确保 keepStart 位置不是 tool 消息
	for keepStart < len(messages) && messages[keepStart].Role == "tool" {
		keepStart--
	}

	oldMsgs := messages[1:keepStart]
	recentMsgs := messages[keepStart:]

	// 生成旧消息摘要
	summary := summarizeMessages(ctx, oldMsgs)

	compactedMsg := Message{
		Role:    "system",
		Content: fmt.Sprintf("[之前的对话摘要]\n%s", summary),
	}

	// 组合: system + 摘要 + 最近消息
	result := make([]Message, 0, 2+len(recentMsgs))
	result = append(result, systemMsg, compactedMsg)
	result = append(result, recentMsgs...)

	newChars := EstimateMessageChars(result)
	fmt.Printf("[Compaction] 压缩完成: %d 条消息 → %d 条, %d 字符 → %d 字符\n",
		len(messages), len(result), totalChars, newChars)

	return result
}

// summarizeMessages 将旧消息生成摘要
// 先构建文本摘要，再尝试用 LLM 精炼。如果 LLM 调用失败则使用原始文本
func summarizeMessages(ctx context.Context, messages []Message) string {
	// 构建摘要文本
	var sb strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("用户: %s\n", truncateForSummary(msg.Content, 200)))
		case "assistant":
			content := msg.Content
			if len(msg.ToolCalls) > 0 {
				toolNames := make([]string, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					toolNames[i] = tc.Function.Name
				}
				content += fmt.Sprintf(" [调用了工具: %s]", strings.Join(toolNames, ", "))
			}
			sb.WriteString(fmt.Sprintf("助手: %s\n", truncateForSummary(content, 300)))
		case "tool":
			sb.WriteString(fmt.Sprintf("工具结果: %s\n", truncateForSummary(msg.Content, 200)))
		}
	}

	rawSummary := sb.String()

	// 尝试用 LLM 生成更精炼的摘要
	summaryPrompt := []Message{
		{
			Role: "system",
			Content: `你是一个信息压缩专家。请将以下对话历史压缩为一段简洁的摘要。
保留关键信息：主要问题、重要发现、已完成的操作、待处理的事项。
摘要应该让后续的对话能够无缝续接，不丢失关键上下文。
用中文回答，控制在 500 字以内。`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("请压缩以下对话历史：\n\n%s", truncateForSummary(rawSummary, 3000)),
		},
	}

	summary, err := sendCompactionRequest(ctx, summaryPrompt)
	if err != nil {
		fmt.Printf("[Compaction] LLM 摘要生成失败，使用原始文本: %v\n", err)
		return truncateForSummary(rawSummary, 1000)
	}

	return summary
}

// sendCompactionRequest 发送压缩摘要请求（无工具调用，避免递归）
func sendCompactionRequest(ctx context.Context, messages []Message) (string, error) {
	modelConfig := GetCurrentModelConfig()
	if modelConfig.APIKey == "" {
		return "", fmt.Errorf("LLM API key not configured")
	}

	apiMessages := convertMessagesToAPI(messages)

	requestBody := map[string]interface{}{
		"model":       modelConfig.Model,
		"messages":    apiMessages,
		"temperature": 0.3, // 摘要用低温度确保稳定
		"stream":      false,
		// 注意：不传 tools，避免模型调用工具导致递归
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal compaction request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", modelConfig.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create compaction request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+modelConfig.APIKey)

	client := &http.Client{Timeout: 60 * time.Second} // 摘要不需要太长时间
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send compaction request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read compaction response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("compaction API error: %d, body: %s", resp.StatusCode, string(body))
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", fmt.Errorf("parse compaction response failed: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return "", fmt.Errorf("empty compaction response")
	}

	return llmResp.Choices[0].Message.Content, nil
}

// truncateForSummary 截断字符串用于摘要
func truncateForSummary(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
