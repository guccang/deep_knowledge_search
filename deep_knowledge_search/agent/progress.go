package agent

import (
	"deepknowledgesearch/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ============================================================================
// 进度文件管理 (Progress File)
// ============================================================================
// 参考 Anthropic 文章 "Effective Harnesses for Long-Running Agents"
// 类似 claude-progress.txt，为 Agent 提供跨会话/跨上下文窗口的记忆

// ProgressEntry 进度条目 — 每个节点完成后或每次会话结束时写入
type ProgressEntry struct {
	Timestamp      time.Time `json:"timestamp"`
	SessionID      string    `json:"session_id"` // 执行器实例 ID
	NodeID         string    `json:"node_id"`
	NodeTitle      string    `json:"node_title"`
	Action         string    `json:"action"`          // completed / failed / paused
	NodesCompleted []string  `json:"nodes_completed"` // 本次完成的节点标题列表
	NodesRemaining []string  `json:"nodes_remaining"` // 待完成的节点标题列表
	Summary        string    `json:"summary"`         // 本次完成的工作摘要
	NextSteps      string    `json:"next_steps"`      // 建议后续步骤
}

// ProgressFile 进度文件
type ProgressFile struct {
	TaskID    string          `json:"task_id"`
	TaskTitle string          `json:"task_title"`
	Entries   []ProgressEntry `json:"entries"`
}

const progressFileName = "progress.json"

// SaveProgressEntry 追加一条进度到进度文件
func SaveProgressEntry(taskFolder string, entry ProgressEntry) error {
	// 加载已有进度或创建新的
	pf, _ := LoadProgress(taskFolder) // 忽略错误，可能是新文件
	if pf == nil {
		pf = &ProgressFile{
			Entries: []ProgressEntry{},
		}
	}

	pf.Entries = append(pf.Entries, entry)

	return saveProgress(taskFolder, pf)
}

// LoadProgress 加载进度文件
func LoadProgress(taskFolder string) (*ProgressFile, error) {
	progressPath := getProgressPath(taskFolder)

	data, err := os.ReadFile(progressPath)
	if err != nil {
		return nil, fmt.Errorf("读取进度文件失败: %w", err)
	}

	var pf ProgressFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("解析进度文件失败: %w", err)
	}

	return &pf, nil
}

// BuildProgressContext 构建进度上下文字符串（注入到 LLM 对话中）
func BuildProgressContext(pf *ProgressFile) string {
	if pf == nil || len(pf.Entries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 之前的执行进度\n\n")

	// 只取最近 5 条进度
	start := 0
	if len(pf.Entries) > 5 {
		start = len(pf.Entries) - 5
	}

	for _, entry := range pf.Entries[start:] {
		sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n",
			entry.Timestamp.Format("15:04:05"),
			entry.Action,
			entry.NodeTitle,
		))
		if entry.Summary != "" {
			sb.WriteString(fmt.Sprintf("  摘要: %s\n", entry.Summary))
		}
	}

	// 添加最新一条的待完成信息
	latest := pf.Entries[len(pf.Entries)-1]
	if len(latest.NodesRemaining) > 0 {
		sb.WriteString("\n### 待完成节点\n")
		for _, name := range latest.NodesRemaining {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", name))
		}
	}

	if latest.NextSteps != "" {
		sb.WriteString(fmt.Sprintf("\n### 建议后续步骤\n%s\n", latest.NextSteps))
	}

	return sb.String()
}

// CollectNodeStatus 从任务树收集已完成和待完成的节点标题
func CollectNodeStatus(root *TaskNode) (completed []string, remaining []string) {
	collectNodeStatusRecursive(root, &completed, &remaining)
	return
}

func collectNodeStatusRecursive(node *TaskNode, completed, remaining *[]string) {
	// 跳过根节点本身
	if node.Depth > 0 {
		switch node.Status {
		case NodeDone:
			*completed = append(*completed, node.Title)
		case NodePending, NodeRunning, NodePaused:
			*remaining = append(*remaining, node.Title)
		}
	}

	for _, child := range node.Children {
		collectNodeStatusRecursive(child, completed, remaining)
	}
}

// ---------- 内部辅助 ----------

func getProgressPath(taskFolder string) string {
	return filepath.Join(config.GetOutputDir(), taskFolder, LogSubDir, progressFileName)
}

func saveProgress(taskFolder string, pf *ProgressFile) error {
	progressPath := getProgressPath(taskFolder)

	// 确保目录存在
	dir := filepath.Dir(progressPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建进度目录失败: %w", err)
	}

	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化进度失败: %w", err)
	}

	if err := os.WriteFile(progressPath, data, 0644); err != nil {
		return fmt.Errorf("写入进度文件失败: %w", err)
	}

	return nil
}
