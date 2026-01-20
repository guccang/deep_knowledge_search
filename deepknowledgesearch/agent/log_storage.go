package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogDir 日志目录
const LogDir = "logs"

// TaskExecutionLog 任务执行日志（用于保存和回放）
type TaskExecutionLog struct {
	TaskID      string             `json:"task_id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	Success     bool               `json:"success"`
	Logs        []ExecutionLog     `json:"logs"`
	Result      *TaskResult        `json:"result,omitempty"`
	Children    []TaskExecutionLog `json:"children,omitempty"`
}

// SaveExecutionLog 保存任务执行日志
func SaveExecutionLog(node *TaskNode) (string, error) {
	// 生成任务文件夹名
	timestamp := time.Now().Format("20060102_150405")
	sanitizedTitle := sanitizeForFilename(node.Title)
	taskFolderName := fmt.Sprintf("%s_%s", sanitizedTitle, timestamp)
	taskDir := filepath.Join(LogDir, taskFolderName)

	// 确保任务日志目录存在
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("创建任务日志目录失败: %w", err)
	}

	// 构建日志结构
	execLog := buildExecutionLog(node)

	// 保存主日志文件
	mainLogPath := filepath.Join(taskDir, "execution.json")
	data, err := json.MarshalIndent(execLog, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化日志失败: %w", err)
	}

	if err := os.WriteFile(mainLogPath, data, 0644); err != nil {
		return "", fmt.Errorf("保存日志失败: %w", err)
	}

	// 保存简要摘要
	summaryPath := filepath.Join(taskDir, "summary.txt")
	summary := buildSummary(node)
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		fmt.Printf("保存摘要失败: %v\n", err)
	}

	// 生成文章索引
	indexPath := filepath.Join(taskDir, "INDEX.md")
	index := GenerateArticleIndex(node, taskFolderName)
	if err := os.WriteFile(indexPath, []byte(index), 0644); err != nil {
		fmt.Printf("保存索引失败: %v\n", err)
	}

	return taskDir, nil
}

// GenerateArticleIndex 生成文章索引（基于任务图结构）
func GenerateArticleIndex(node *TaskNode, taskFolder string) string {
	var sb strings.Builder

	// 标题
	sb.WriteString("# 📚 任务索引\n\n")
	sb.WriteString(fmt.Sprintf("**任务:** %s\n\n", node.Title))
	sb.WriteString(fmt.Sprintf("**执行时间:** %s\n\n", node.CreatedAt.Format("2006-01-02 15:04:05")))

	// 状态
	if node.Result != nil {
		if node.Result.Success {
			sb.WriteString("**状态:** ✅ 完成\n\n")
		} else {
			sb.WriteString("**状态:** ❌ 失败\n\n")
		}
	}

	sb.WriteString("---\n\n")

	// 任务图结构
	sb.WriteString("## 📊 任务结构\n\n")
	sb.WriteString("```\n")
	buildTaskTree(&sb, node, 0)
	sb.WriteString("```\n\n")

	// 输出文件列表
	sb.WriteString("## 📁 输出文件\n\n")
	outputDir := filepath.Join("output", taskFolder)
	files := listOutputFiles(outputDir)
	if len(files) > 0 {
		for _, f := range files {
			sb.WriteString(fmt.Sprintf("- [%s](%s)\n", f, f))
		}
	} else {
		sb.WriteString("*无输出文件*\n")
	}
	sb.WriteString("\n")

	// 详细任务列表
	sb.WriteString("## 📋 任务详情\n\n")
	buildTaskDetails(&sb, node, 1)

	// 结果摘要
	if node.Result != nil && node.Result.Summary != "" {
		sb.WriteString("---\n\n")
		sb.WriteString("## 📝 执行结果\n\n")
		sb.WriteString(node.Result.Summary)
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildTaskTree 构建任务树状图
func buildTaskTree(sb *strings.Builder, node *TaskNode, depth int) {
	indent := strings.Repeat("  ", depth)
	status := "✅"
	if node.Result == nil || !node.Result.Success {
		if node.Status == NodeFailed {
			status = "❌"
		} else if node.Status == NodeRunning {
			status = "🔄"
		} else {
			status = "⏳"
		}
	}
	sb.WriteString(fmt.Sprintf("%s%s %s\n", indent, status, node.Title))

	for _, child := range node.Children {
		buildTaskTree(sb, child, depth+1)
	}
}

// buildTaskDetails 构建任务详情
func buildTaskDetails(sb *strings.Builder, node *TaskNode, level int) {
	prefix := strings.Repeat("#", level+2)
	sb.WriteString(fmt.Sprintf("%s %s\n\n", prefix, node.Title))

	if node.Description != "" {
		sb.WriteString(fmt.Sprintf("**描述:** %s\n\n", node.Description))
	}
	if node.Goal != "" {
		sb.WriteString(fmt.Sprintf("**目标:** %s\n\n", node.Goal))
	}

	if node.Result != nil && node.Result.Summary != "" {
		sb.WriteString(fmt.Sprintf("**结果:** %s\n\n", node.Result.Summary))
	}

	for _, child := range node.Children {
		buildTaskDetails(sb, child, level+1)
	}
}

// listOutputFiles 列出输出目录中的文件
func listOutputFiles(dir string) []string {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files
}

// buildSummary 构建任务摘要
func buildSummary(node *TaskNode) string {
	status := "成功"
	if node.Result == nil || !node.Result.Success {
		status = "失败"
	}

	summary := fmt.Sprintf("任务: %s\n", node.Title)
	summary += fmt.Sprintf("状态: %s\n", status)
	summary += fmt.Sprintf("开始时间: %s\n", node.CreatedAt.Format("2006-01-02 15:04:05"))
	if node.FinishedAt != nil {
		summary += fmt.Sprintf("结束时间: %s\n", node.FinishedAt.Format("2006-01-02 15:04:05"))
	}
	summary += fmt.Sprintf("子任务数: %d\n", len(node.Children))

	if node.Result != nil && node.Result.Summary != "" {
		summary += fmt.Sprintf("\n结果摘要:\n%s\n", node.Result.Summary)
	}

	return summary
}

// buildExecutionLog 从 TaskNode 构建执行日志
func buildExecutionLog(node *TaskNode) TaskExecutionLog {
	log := TaskExecutionLog{
		TaskID:      node.ID,
		Title:       node.Title,
		Description: node.Description,
		StartTime:   node.CreatedAt,
		Logs:        node.Logs,
		Result:      node.Result,
	}

	if node.FinishedAt != nil {
		log.EndTime = *node.FinishedAt
	} else {
		log.EndTime = time.Now()
	}

	if node.Result != nil {
		log.Success = node.Result.Success
	}

	// 递归处理子节点
	for _, child := range node.Children {
		log.Children = append(log.Children, buildExecutionLog(child))
	}

	return log
}

// sanitizeForFilename 清理文件名中的非法字符（支持中文）
func sanitizeForFilename(name string) string {
	// 使用 strings.ReplaceAll 正确处理 Unicode
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// 限制长度（按 rune 计算以正确处理中文）
	runes := []rune(result)
	if len(runes) > 30 {
		result = string(runes[:30])
	}
	return result
}

// LoadExecutionLog 加载执行日志（用于回放）
func LoadExecutionLog(filepath string) (*TaskExecutionLog, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}

	var log TaskExecutionLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("解析日志文件失败: %w", err)
	}

	return &log, nil
}

// PrintExecutionLog 打印执行日志（用于调试）
func PrintExecutionLog(log *TaskExecutionLog, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	status := "✅"
	if !log.Success {
		status = "❌"
	}

	fmt.Printf("%s%s [%s] %s\n", prefix, status, log.TaskID[:4], log.Title)

	for _, l := range log.Logs {
		levelIcon := "ℹ️"
		switch l.Level {
		case LogWarn:
			levelIcon = "⚠️"
		case LogError:
			levelIcon = "❌"
		case LogDebug:
			levelIcon = "🔍"
		}
		fmt.Printf("%s  %s %s: %s\n", prefix, levelIcon, l.Phase, l.Message)
	}

	for _, child := range log.Children {
		PrintExecutionLog(&child, indent+1)
	}
}
