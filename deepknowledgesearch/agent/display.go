package agent

import (
	"deepknowledgesearch/web"
	"fmt"
	"strings"
)

// ConsoleDisplay 控制台显示器
type ConsoleDisplay struct{}

// Display 全局显示器实例
var Display = &ConsoleDisplay{}

// TaskStart 显示任务开始
func (d *ConsoleDisplay) TaskStart(title string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Printf("║  🚀 任务开始: %-44s║\n", truncateString(title, 44))
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 广播到 Web
	web.BroadcastEvent("task_start", map[string]interface{}{
		"title": title,
	})
}

// TaskComplete 显示任务完成
func (d *ConsoleDisplay) TaskComplete(title string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Printf("║  ✅ 任务完成: %-44s║\n", truncateString(title, 44))
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	web.BroadcastEvent("task_complete", map[string]interface{}{
		"title": title,
	})
}

// TaskFailed 显示任务失败
func (d *ConsoleDisplay) TaskFailed(title string, err error) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Printf("║  ❌ 任务失败: %-44s║\n", truncateString(title, 44))
	fmt.Printf("║  错误: %-51s║\n", truncateString(err.Error(), 51))
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	web.BroadcastEvent("task_failed", map[string]interface{}{
		"title": title,
		"error": err.Error(),
	})
}

// NodeStart 显示节点开始
func (d *ConsoleDisplay) NodeStart(node *TaskNode) {
	indent := strings.Repeat("  ", node.Depth)
	fmt.Printf("%s├─ 🔄 [%s] %s\n", indent, node.ID[:4], node.Title)

	web.BroadcastEvent("node_start", buildNodeData(node))
}

// NodeComplete 显示节点完成
func (d *ConsoleDisplay) NodeComplete(node *TaskNode) {
	indent := strings.Repeat("  ", node.Depth)
	summary := ""
	if node.Result != nil && node.Result.Summary != "" {
		summary = " → " + truncateString(node.Result.Summary, 40)
	}
	fmt.Printf("%s├─ ✅ [%s] %s%s\n", indent, node.ID[:4], node.Title, summary)

	web.BroadcastEvent("node_complete", buildNodeData(node))
}

// NodeFailed 显示节点失败
func (d *ConsoleDisplay) NodeFailed(node *TaskNode, err error) {
	indent := strings.Repeat("  ", node.Depth)
	fmt.Printf("%s├─ ❌ [%s] %s: %s\n", indent, node.ID[:4], node.Title, err.Error())

	web.BroadcastEvent("node_failed", map[string]interface{}{
		"title": node.Title,
		"error": err.Error(),
	})
}

// ShowSubtasks 显示子任务
func (d *ConsoleDisplay) ShowSubtasks(subtasks []SubTaskPlan, mode ExecutionMode) {
	modeStr := "串行"
	if mode == ModeParallel {
		modeStr = "并行"
	}
	fmt.Printf("   📋 分解为 %d 个子任务 (%s执行):\n", len(subtasks), modeStr)
	for i, st := range subtasks {
		fmt.Printf("      %d. %s\n", i+1, st.Title)
	}
	fmt.Println()

	web.BroadcastEvent("subtasks", map[string]interface{}{
		"count": len(subtasks),
		"mode":  mode,
	})
}

// ShowMessage 显示消息
func (d *ConsoleDisplay) ShowMessage(icon string, message string) {
	fmt.Printf("   %s %s\n", icon, message)

	web.BroadcastEvent("log", map[string]interface{}{
		"level":   "info",
		"message": message,
	})
}

// ShowProgress 显示进度
func (d *ConsoleDisplay) ShowProgress(current, total int, message string) {
	percent := float64(current) / float64(total) * 100
	bar := generateProgressBar(percent, 20)
	fmt.Printf("\r   [%s] %.0f%% %s", bar, percent, message)
	if current == total {
		fmt.Println()
	}
}

// ShowResult 显示结果
func (d *ConsoleDisplay) ShowResult(result string) {
	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println("📋 执行结果:")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println(result)
	fmt.Println("────────────────────────────────────────────────────────────")
}

// buildNodeData 构建节点数据用于广播
func buildNodeData(node *TaskNode) map[string]interface{} {
	data := map[string]interface{}{
		"id":     node.ID,
		"title":  node.Title,
		"status": string(node.Status),
		"depth":  node.Depth,
	}
	if len(node.Children) > 0 {
		children := make([]map[string]interface{}, 0, len(node.Children))
		for _, child := range node.Children {
			children = append(children, buildNodeData(child))
		}
		data["children"] = children
	}
	return data
}

// ============================================================================
// 辅助函数
// ============================================================================

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// generateProgressBar 生成进度条
func generateProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}
