package agent

import (
	"context"
	"deepknowledgesearch/llm"
	"deepknowledgesearch/mcp"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TaskPlanner 任务规划器
type TaskPlanner struct {
	maxDepth int
}

// NewTaskPlanner 创建任务规划器
func NewTaskPlanner() *TaskPlanner {
	return &TaskPlanner{
		maxDepth: DefaultMaxDepth,
	}
}

// ============================================================================
// 规划结果结构
// ============================================================================

// NodePlanningResult 节点规划结果
type NodePlanningResult struct {
	Title         string        `json:"title"`
	Goal          string        `json:"goal"`
	ExecutionMode ExecutionMode `json:"execution_mode"`
	SubTasks      []SubTaskPlan `json:"subtasks"`
	Reasoning     string        `json:"reasoning"`
}

// SubTaskPlan 子任务规划
type SubTaskPlan struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Goal         string   `json:"goal"`
	Tools        []string `json:"tools"`
	CanDecompose bool     `json:"can_decompose"`
}

// ============================================================================
// 规划方法
// ============================================================================

// PlanNode 规划任务节点
func (p *TaskPlanner) PlanNode(ctx context.Context, node *TaskNode) (*NodePlanningResult, error) {
	// 获取可用工具列表
	tools := p.getAvailableToolsDescription()

	// 构建上下文
	contextStr := node.Context.BuildLLMContext()

	// 构建 prompt
	prompt := BuildNodePlanningPrompt(
		node.Title,
		node.Description,
		node.Goal,
		contextStr,
		tools,
	)

	// 调用 LLM
	messages := []llm.Message{
		{Role: "system", Content: PromptPlanningSystem},
		{Role: "user", Content: prompt},
	}

	// 记录开始时间
	startTime := time.Now()

	// 注入 OutputPath 到 Context
	if node.OutputPath != "" {
		ctx = context.WithValue(ctx, mcp.ContextKeyOutputPath, node.OutputPath)
	}

	response, err := llm.SendSyncLLMRequest(ctx, messages)

	// 计算耗时并记录 LLM 调用
	durationMs := time.Since(startTime).Milliseconds()
	llmMessages := []map[string]interface{}{
		{"role": "system", "content": PromptPlanningSystem},
		{"role": "user", "content": prompt},
	}
	node.AddLLMCall("plan", llmMessages, response, startTime, durationMs)

	if err != nil {
		return nil, fmt.Errorf("LLM 规划失败: %w", err)
	}

	// 解析 JSON 响应
	result, err := p.parsePlanningResponse(response)
	if err != nil {
		// 如果解析失败，返回空子任务（直接执行）
		node.AddLog(LogWarn, "planning", fmt.Sprintf("规划响应解析失败，直接执行: %v", err))
		return &NodePlanningResult{
			Title:         node.Title,
			Goal:          node.Goal,
			ExecutionMode: ModeSequential,
			SubTasks:      []SubTaskPlan{},
		}, nil
	}

	return result, nil
}

// ExecuteNode 执行任务节点
func (p *TaskPlanner) ExecuteNode(ctx context.Context, node *TaskNode) (*TaskResult, error) {
	// 构建上下文
	contextStr := node.Context.BuildLLMContext()

	// 构建 prompt
	prompt := BuildNodeExecutionPrompt(
		node.Title,
		node.Description,
		node.Goal,
		contextStr,
	)

	// 调用 LLM（带工具）
	messages := []llm.Message{
		{Role: "system", Content: PromptExecutionSystem},
		{Role: "user", Content: prompt},
	}

	// 记录开始时间 (已移至循环内)
	// startTime := time.Now()

	// 注入 OutputPath 到 Context
	if node.OutputPath != "" {
		ctx = context.WithValue(ctx, mcp.ContextKeyOutputPath, node.OutputPath)
	}

	var response string
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		// 每次重试重新计时
		callStartTime := time.Now()

		response, err = llm.SendSyncLLMRequest(ctx, messages)

		// 计算耗时并记录 LLM 调用
		durationMs := time.Since(callStartTime).Milliseconds()
		llmMessages := []map[string]interface{}{
			{"role": "system", "content": PromptExecutionSystem},
			{"role": "user", "content": prompt},
		}

		// 记录调用（包含重试信息）
		callType := "execute"
		if i > 0 {
			callType = fmt.Sprintf("execute_retry_%d", i)
		}
		node.AddLLMCall(callType, llmMessages, response, callStartTime, durationMs)

		if err == nil {
			break
		}

		if i < maxRetries-1 {
			node.AddLog(LogWarn, "retry", fmt.Sprintf("LLM 执行失败，准备重试 (%d/%d): %v", i+1, maxRetries, err))
			time.Sleep(time.Second * 2)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("LLM 执行失败 (重试 %d 次后): %w", maxRetries, err)
	}

	// 生成摘要
	summary := p.summarizeResponse(response)

	return NewTaskResult(response, summary), nil
}

// SynthesizeResults 整合子任务结果
func (p *TaskPlanner) SynthesizeResults(ctx context.Context, node *TaskNode, summaries []string) (string, error) {
	if len(summaries) == 0 {
		return "无子任务结果", nil
	}

	childResults := strings.Join(summaries, "\n")

	prompt := BuildResultSynthesisPrompt(
		node.Title,
		node.Goal,
		childResults,
	)

	messages := []llm.Message{
		{Role: "system", Content: "你是一个结果整合专家。"},
		{Role: "user", Content: prompt},
	}

	// 记录开始时间
	startTime := time.Now()

	// 注入 OutputPath 到 Context (虽然整合阶段可能不需要写文件，但保持一致)
	if node.OutputPath != "" {
		ctx = context.WithValue(ctx, mcp.ContextKeyOutputPath, node.OutputPath)
	}

	response, err := llm.SendSyncLLMRequest(ctx, messages)

	// 计算耗时并记录 LLM 调用
	durationMs := time.Since(startTime).Milliseconds()
	llmMessages := []map[string]interface{}{
		{"role": "system", "content": "你是一个结果整合专家。"},
		{"role": "user", "content": prompt},
	}
	node.AddLLMCall("synthesize", llmMessages, response, startTime, durationMs)

	if err != nil {
		return childResults, err
	}

	return response, nil
}

// VerificationResult 验证结果
type VerificationResult struct {
	Passed      bool   `json:"passed"`
	Feedback    string `json:"feedback"`
	Suggestions string `json:"suggestions"`
}

// VerifyResult 验证任务执行结果（迭代验证直到通过）
func (p *TaskPlanner) VerifyResult(ctx context.Context, node *TaskNode, result string) (*VerificationResult, error) {
	const maxVerificationIterations = 5
	currentResult := result

	// 初始化验证信息
	node.Verification = &VerificationInfo{
		Passed:     false,
		Iterations: 0,
		Attempts:   []VerificationAttempt{},
	}

	// 注入 OutputPath 到 Context
	if node.OutputPath != "" {
		ctx = context.WithValue(ctx, mcp.ContextKeyOutputPath, node.OutputPath)
	}

	for iteration := 0; iteration < maxVerificationIterations; iteration++ {
		Display.ShowMessage("🔍", fmt.Sprintf("验证任务结果 (第 %d 次)...", iteration+1))
		node.AddLog(LogInfo, "verification", fmt.Sprintf("开始第 %d 次验证", iteration+1))

		// 构建验证 prompt
		prompt := BuildVerificationPrompt(
			node.Title,
			node.Goal,
			currentResult,
		)

		messages := []llm.Message{
			{Role: "system", Content: PromptVerificationSystem},
			{Role: "user", Content: prompt},
		}

		// 记录开始时间
		startTime := time.Now()

		response, err := llm.SendSyncLLMRequest(ctx, messages)

		// 记录 LLM 调用
		durationMs := time.Since(startTime).Milliseconds()
		llmMessages := []map[string]interface{}{
			{"role": "system", "content": PromptVerificationSystem},
			{"role": "user", "content": prompt},
		}
		node.AddLLMCall("verify", llmMessages, response, startTime, durationMs)

		if err != nil {
			// 记录验证尝试（失败）
			node.Verification.Attempts = append(node.Verification.Attempts, VerificationAttempt{
				Iteration: iteration + 1,
				Passed:    false,
				Feedback:  fmt.Sprintf("验证调用失败: %v", err),
				Timestamp: time.Now().Format("15:04:05"),
			})
			node.Verification.Iterations = iteration + 1
			Display.BroadcastTree(findRootNode(node))
			return nil, fmt.Errorf("验证调用失败: %w", err)
		}

		// 检查是否通过验证
		if strings.Contains(response, "VERIFICATION_PASSED") {
			Display.ShowMessage("✅", "验证通过!")
			node.AddLog(LogInfo, "verification", "验证通过")

			// 记录验证通过
			node.Verification.Passed = true
			node.Verification.Iterations = iteration + 1
			node.Verification.Attempts = append(node.Verification.Attempts, VerificationAttempt{
				Iteration: iteration + 1,
				Passed:    true,
				Feedback:  p.summarizeResponse(response),
				Timestamp: time.Now().Format("15:04:05"),
			})
			Display.BroadcastTree(findRootNode(node))

			return &VerificationResult{
				Passed:   true,
				Feedback: response,
			}, nil
		}

		// 验证未通过，记录反馈
		Display.ShowMessage("⚠️", fmt.Sprintf("验证未通过，需要改进 (第 %d 次)", iteration+1))
		node.AddLog(LogWarn, "verification", fmt.Sprintf("验证未通过: %s", p.summarizeResponse(response)))

		// 记录验证尝试
		node.Verification.Iterations = iteration + 1
		node.Verification.Attempts = append(node.Verification.Attempts, VerificationAttempt{
			Iteration: iteration + 1,
			Passed:    false,
			Feedback:  p.summarizeResponse(response),
			Timestamp: time.Now().Format("15:04:05"),
		})
		Display.BroadcastTree(findRootNode(node))

		// 如果还有迭代机会，尝试改进
		if iteration < maxVerificationIterations-1 {
			// 让 LLM 根据反馈改进结果
			improvePrompt := fmt.Sprintf(`根据以下验证反馈改进任务结果。

## 原始任务
标题: %s
目标: %s

## 当前结果
%s

## 验证反馈
%s

请根据反馈改进结果，确保满足任务目标。`, node.Title, node.Goal, currentResult, response)

			improveMessages := []llm.Message{
				{Role: "system", Content: PromptExecutionSystem},
				{Role: "user", Content: improvePrompt},
			}

			improvedResult, err := llm.SendSyncLLMRequest(ctx, improveMessages)
			if err != nil {
				node.AddLog(LogError, "verification", fmt.Sprintf("改进失败: %v", err))
				continue
			}

			currentResult = improvedResult
			node.AddLog(LogInfo, "verification", "已根据反馈改进结果")
		}
	}

	// 达到最大迭代次数仍未通过
	return &VerificationResult{
		Passed:      false,
		Feedback:    "达到最大验证次数，验证未通过",
		Suggestions: "请检查任务目标设定是否合理",
	}, nil
}

// findRootNode 查找根节点（用于广播）
func findRootNode(node *TaskNode) *TaskNode {
	// 由于节点只存储 ParentID，无法向上遍历
	// 这里返回当前节点，实际广播时需要从 executor 获取根节点
	return node
}

// ============================================================================
// 辅助方法
// ============================================================================

// getAvailableToolsDescription 获取可用工具描述
func (p *TaskPlanner) getAvailableToolsDescription() string {
	tools := mcp.GetAvailableLLMTools()
	if len(tools) == 0 {
		return "无可用工具"
	}

	var sb strings.Builder
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Function.Name, tool.Function.Description))
	}
	return sb.String()
}

// parsePlanningResponse 解析规划响应
func (p *TaskPlanner) parsePlanningResponse(response string) (*NodePlanningResult, error) {
	// 清理 JSON
	cleaned := cleanJSONResponse(response)

	var result NodePlanningResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始响应: %s", err, cleaned)
	}

	return &result, nil
}

// summarizeResponse 生成响应摘要
func (p *TaskPlanner) summarizeResponse(response string) string {
	// 简单截断作为摘要
	runes := []rune(response)
	if len(runes) > 100 {
		return string(runes[:100]) + "..."
	}
	return response
}

// cleanJSONResponse 清理 JSON 响应
func cleanJSONResponse(response string) string {
	// 移除 markdown 代码块标记
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// 尝试找到 JSON 开始和结束位置
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start != -1 && end != -1 && end > start {
		return response[start : end+1]
	}

	return response
}

// ============================================================================
// 旧 API 兼容
// ============================================================================

// extractTaskTitle 从任务描述中提取简短标题
func extractTaskTitle(description string) string {
	// 取描述的前15个字符或到第一个标点符号
	runes := []rune(description)

	// 定义截止标点
	punctuations := []rune{'。', '，', '、', '；', '：', '？', '！', '\n', '.', ',', ';', ':', '?', '!'}

	maxLen := 50
	if len(runes) < maxLen {
		maxLen = len(runes)
	}

	// 查找第一个标点位置
	endPos := maxLen
	for i := 0; i < maxLen; i++ {
		for _, p := range punctuations {
			if runes[i] == p {
				if i > 0 {
					endPos = i
				}
				goto done
			}
		}
	}
done:

	title := string(runes[:endPos])
	if len(runes) > endPos {
		title += "..."
	}
	return strings.TrimSpace(title)
}

// ExecuteTask 执行任务（旧 API，使用新的执行器）
func (p *TaskPlanner) ExecuteTask(description string) (string, error) {
	// 创建根节点 - 使用任务描述提取标题
	taskTitle := extractTaskTitle(description)
	node := NewTaskNode(taskTitle, description)
	node.Goal = "完成用户请求的任务"

	// 创建执行配置
	config := DefaultExecutionConfig()
	config.MaxDepth = p.maxDepth

	// 创建执行器
	executor := NewTaskExecutor(node, p, config)

	// 注册执行器（如果设置了回调）
	if OnExecutorCreated != nil {
		OnExecutorCreated(node.ID, executor)
		// 确保任务完成后清理
		if OnExecutorFinished != nil {
			defer OnExecutorFinished(node.ID, nil)
		}
	}

	// 执行
	if err := executor.Execute(); err != nil {
		return "", err
	}

	// 返回结果
	if node.Result != nil {
		Display.ShowResult(node.Result.Summary)
		return node.Result.Summary, nil
	}

	return "任务已完成", nil
}
