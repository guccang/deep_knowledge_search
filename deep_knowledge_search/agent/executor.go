package agent

import (
	"context"
	"deepknowledgesearch/config"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// TaskExecutor 任务执行器
type TaskExecutor struct {
	root    *TaskNode
	planner *TaskPlanner
	config  *ExecutionConfig
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex

	// 暂停控制
	paused   bool
	pauseCh  chan struct{}
	resumeCh chan struct{}

	// 恢复控制
	recovering bool
	taskFolder string
}

// NewTaskExecutor 创建任务执行器
func NewTaskExecutor(root *TaskNode, planner *TaskPlanner, config *ExecutionConfig) *TaskExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskExecutor{
		root:       root,
		planner:    planner,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		paused:     false,
		pauseCh:    make(chan struct{}),
		resumeCh:   make(chan struct{}),
		recovering: false,
		taskFolder: "",
	}
}

// Execute 执行任务图
func (e *TaskExecutor) Execute() error {
	if e.root == nil {
		return fmt.Errorf("no root node")
	}

	// 设置当前任务的输出目录
	var taskFolderName string
	if e.recovering && e.taskFolder != "" {
		// 恢复模式：使用已有的任务文件夹
		taskFolderName = e.taskFolder
		Display.ShowMessage("🔄", fmt.Sprintf("恢复任务: %s", taskFolderName))
	} else {
		// 正常模式：创建新的任务文件夹
		taskFolderName = fmt.Sprintf("%s_%s", sanitizeForFilename(e.root.Title), time.Now().Format("20060102_150405"))
		e.taskFolder = taskFolderName // 保存以便任务完成后清理检查点
	}

	// mcp.SetTaskOutputDir(taskFolderName) // 移除全局设置
	// defer mcp.ClearTaskOutputDir()       // 移除全局清理

	Display.TaskStart(e.root.Title)
	e.root.AddLog(LogInfo, "starting", fmt.Sprintf("开始执行任务: %s", e.root.Title))

	// Phase 0: 预检与特性清单
	e.preflight()

	// 启动周期性检查点保存（每30秒）
	checkpointTicker := time.NewTicker(30 * time.Second)
	go func() {
		defer checkpointTicker.Stop()
		for {
			select {
			case <-checkpointTicker.C:
				// 只在运行中且未暂停时保存
				if e.root.Status == NodeRunning && !e.IsPaused() {
					if err := e.saveCheckpoint(); err != nil {
						Display.ShowMessage("⚠️", fmt.Sprintf("自动保存检查点失败: %v", err))
					}
				}
			case <-e.ctx.Done():
				return
			}
		}
	}()

	// 执行根节点
	err := e.executeNode(e.root)

	if err != nil {
		Display.TaskFailed(e.root.Title, err)
		// 保存失败日志
		e.saveExecutionLog()
		return err
	}

	// 验证任务结果
	if e.root.Result != nil && e.root.Result.Success {
		Display.ShowMessage("📋", "开始验证任务结果...")

		verifyResult, verifyErr := e.planner.VerifyResult(e.ctx, e.root, e.root.Result.Summary)
		if verifyErr != nil {
			e.root.AddLog(LogError, "verification", fmt.Sprintf("验证失败: %v", verifyErr))
			Display.ShowMessage("⚠️", fmt.Sprintf("验证过程出错: %v", verifyErr))
		} else if !verifyResult.Passed {
			e.root.AddLog(LogWarn, "verification", "任务未通过验证")
			Display.ShowMessage("⚠️", "任务未通过验证，请检查结果")
			e.root.Result.Success = false
		} else {
			e.root.AddLog(LogInfo, "verification", "任务验证通过")
		}
		// 广播验证完成后的树结构
		Display.BroadcastTree(e.root)
	}

	// 生成输出目录的 README 索引
	// outputDir := mcp.GetCurrentOutputDir() // 移除
	outputDir := filepath.Join(config.GetOutputDir(), e.taskFolder)
	if err := GenerateOutputReadme(e.root, outputDir); err != nil {
		Display.ShowMessage("⚠️", fmt.Sprintf("生成索引失败: %v", err))
	} else {
		Display.ShowMessage("📚", fmt.Sprintf("已生成索引: %s/README.md", outputDir))
	}

	// 保存执行日志
	e.saveExecutionLog()

	// 清理检查点文件（任务完成后不再需要恢复）
	if e.taskFolder != "" {
		rm := NewRecoveryManager()
		if err := rm.CleanupCheckpoint(e.taskFolder); err != nil {
			Display.ShowMessage("⚠️", fmt.Sprintf("清理检查点失败: %v", err))
		}
	}

	Display.TaskComplete(e.root.Title)
	return nil
}

// saveExecutionLog 保存执行日志
func (e *TaskExecutor) saveExecutionLog() {
	// 传入 taskFolder
	logPath, err := SaveExecutionLog(e.root, e.taskFolder)
	if err != nil {
		Display.ShowMessage("⚠️", fmt.Sprintf("保存日志失败: %v", err))
	} else {
		Display.ShowMessage("📝", fmt.Sprintf("执行日志已保存: %s", logPath))
	}
}

// executeNode 执行单个节点
func (e *TaskExecutor) executeNode(node *TaskNode) error {
	// 设置节点输出路径
	e.setNodeOutputPath(node)

	// 暂停检查点
	e.checkPausePoint()

	// 检查取消
	select {
	case <-e.ctx.Done():
		node.SetStatus(NodeCanceled)
		return fmt.Errorf("execution canceled")
	default:
	}

	if node.IsCanceled() {
		return fmt.Errorf("node canceled")
	}

	// 跳过已完成的节点
	if node.Status == NodeDone {
		return nil
	}

	// 设置运行状态
	node.SetStatus(NodeRunning)
	node.AddLog(LogInfo, "executing", fmt.Sprintf("开始执行: %s", node.Title))
	Display.NodeStart(node)

	// 检查是否需要拆解
	if e.shouldDecompose(node) {
		if err := e.decomposeNode(node); err != nil {
			node.AddLog(LogError, "planning", fmt.Sprintf("任务拆解失败: %v", err))
			return e.handleNodeError(node, err)
		}
	}

	// 如果有子节点，执行子节点
	if len(node.Children) > 0 {
		var err error
		switch node.ExecutionMode {
		case ModeParallel:
			err = e.executeParallel(node)
		default:
			err = e.executeSequential(node)
		}

		if err != nil {
			return e.handleNodeError(node, err)
		}

		// 汇总子节点结果
		e.aggregateChildResults(node)
	} else {
		// 叶子节点，设置节点输出路径 (已在开头设置，这里不再需要)
		// e.setNodeOutputPath(node)

		// 叶子节点，直接执行
		if err := e.executeLeafNode(node); err != nil {
			return e.handleNodeError(node, err)
		}
	}

	// 标记完成
	node.SetStatus(NodeDone)
	node.SetProgress(100)
	node.AddLog(LogInfo, "completed", fmt.Sprintf("执行完成: %s", node.Title))
	Display.NodeComplete(node)

	// 保存进度条目
	if e.taskFolder != "" {
		completed, remaining := CollectNodeStatus(e.root)
		summary := ""
		if node.Result != nil {
			summary = node.Result.Summary
		}
		entry := ProgressEntry{
			Timestamp:      time.Now(),
			SessionID:      e.root.ID,
			NodeID:         node.ID,
			NodeTitle:      node.Title,
			Action:         "completed",
			NodesCompleted: completed,
			NodesRemaining: remaining,
			Summary:        summary,
		}
		if err := SaveProgressEntry(e.taskFolder, entry); err != nil {
			Display.ShowMessage("⚠️", fmt.Sprintf("保存进度失败: %v", err))
		}
	}

	// 广播完整树结构确保前端同步
	Display.BroadcastTree(e.root)

	return nil
}

// shouldDecompose 判断是否需要拆解
func (e *TaskExecutor) shouldDecompose(node *TaskNode) bool {
	if len(node.Children) > 0 {
		return false
	}
	if !node.CanDecompose {
		return false
	}
	if node.Depth >= e.config.MaxDepth {
		node.AddLog(LogInfo, "planning", fmt.Sprintf("达到最大深度 %d，不再拆解", e.config.MaxDepth))
		return false
	}
	return true
}

// decomposeNode 拆解节点
func (e *TaskExecutor) decomposeNode(node *TaskNode) error {
	node.AddLog(LogInfo, "planning", "开始任务拆解")
	Display.ShowMessage("🔍", fmt.Sprintf("分析任务: %s", node.Title))

	// 调用 planner 进行拆解
	result, err := e.planner.PlanNode(e.ctx, node)
	if err != nil {
		return err
	}

	// 如果没有子任务，标记为不可拆解
	if len(result.SubTasks) == 0 {
		node.CanDecompose = false
		node.AddLog(LogInfo, "planning", "无需拆解，直接执行")
		return nil
	}

	// 创建子节点
	node.ExecutionMode = result.ExecutionMode
	Display.ShowSubtasks(result.SubTasks, result.ExecutionMode)

	for _, st := range result.SubTasks {
		child := node.NewChildNode(st.Title, st.Description, st.Goal)
		child.ToolCalls = st.Tools
		child.CanDecompose = st.CanDecompose
	}

	node.AddLog(LogInfo, "planning", fmt.Sprintf("任务拆解完成: %d 个子任务，模式: %s", len(node.Children), node.ExecutionMode))
	return nil
}

// executeSequential 串行执行子节点
func (e *TaskExecutor) executeSequential(node *TaskNode) error {
	node.AddLog(LogInfo, "executing", fmt.Sprintf("串行执行 %d 个子任务", len(node.Children)))

	for i, child := range node.Children {
		if err := e.executeNode(child); err != nil {
			if child.CanRetry() {
				child.IncrementRetry()
				child.AddLog(LogWarn, "retry", fmt.Sprintf("重试第 %d 次", child.RetryCount))
				child.SetStatus(NodePending)
				i--
				continue
			}
			return err
		}

		// 更新父节点进度
		progress := float64(i+1) / float64(len(node.Children)) * 100
		node.SetProgress(progress)

		// 添加兄弟结果到上下文
		e.propagateSiblingResult(child, node)
	}

	return nil
}

// executeParallel 并行执行子节点
func (e *TaskExecutor) executeParallel(node *TaskNode) error {
	node.AddLog(LogInfo, "executing", fmt.Sprintf("并行执行 %d 个子任务", len(node.Children)))

	var wg sync.WaitGroup
	errChan := make(chan error, len(node.Children))

	for _, child := range node.Children {
		wg.Add(1)
		go func(c *TaskNode) {
			defer wg.Done()
			if err := e.executeNode(c); err != nil {
				errChan <- err
			}
		}(child)
	}

	wg.Wait()
	close(errChan)

	// 检查错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel execution failed: %v", errors[0])
	}

	return nil
}

// executeLeafNode 执行叶子节点
func (e *TaskExecutor) executeLeafNode(node *TaskNode) error {
	node.AddLog(LogInfo, "executing", fmt.Sprintf("执行叶子节点: %s", node.Title))

	// 调用 planner 执行
	result, err := e.planner.ExecuteNode(e.ctx, node)
	if err != nil {
		node.Result = NewTaskResultError(err.Error())
		return err
	}

	node.Result = result
	node.AddLog(LogInfo, "completed", fmt.Sprintf("执行结果: %s", result.Summary))

	return nil
}

// propagateSiblingResult 传播兄弟结果
func (e *TaskExecutor) propagateSiblingResult(completed *TaskNode, parent *TaskNode) {
	if completed.Result == nil {
		return
	}

	for _, sibling := range parent.Children {
		if sibling.ID != completed.ID && sibling.Status == NodePending {
			sibling.Context.AddSiblingResult(
				completed.ID,
				completed.Title,
				completed.Status,
				completed.Result.Summary,
			)
		}
	}
}

// aggregateChildResults 汇总子节点结果
func (e *TaskExecutor) aggregateChildResults(node *TaskNode) {
	var summaries []string
	var allSuccess = true

	for _, child := range node.Children {
		if child.Result != nil {
			summaries = append(summaries, fmt.Sprintf("%s: %s", child.Title, child.Result.Summary))
			if !child.Result.Success {
				allSuccess = false
			}
		}
	}

	// 尝试使用 LLM 整合结果
	synthesized, err := e.planner.SynthesizeResults(e.ctx, node, summaries)
	if err != nil {
		synthesized = fmt.Sprintf("完成 %d 个子任务", len(node.Children))
	}

	node.Result = &TaskResult{
		Success: allSuccess,
		Summary: synthesized,
		Output:  joinStrings(summaries, "\n"),
	}
}

// handleNodeError 处理节点错误
func (e *TaskExecutor) handleNodeError(node *TaskNode, err error) error {
	node.SetStatus(NodeFailed)
	node.Result = NewTaskResultError(err.Error())
	node.AddLog(LogError, "failed", fmt.Sprintf("执行失败: %v", err))
	Display.NodeFailed(node, err)
	return err
}

// Cancel 取消执行
func (e *TaskExecutor) Cancel() {
	e.cancel()
	e.root.Cancel()
}

// Pause 暂停执行
func (e *TaskExecutor) Pause() {
	e.mu.Lock()
	if e.paused {
		e.mu.Unlock()
		return
	}
	e.paused = true
	e.mu.Unlock()

	// 暂停根节点
	e.root.Pause()

	// 发送暂停信号
	select {
	case e.pauseCh <- struct{}{}:
	default:
	}

	Display.ShowMessage("⏸️", "任务已暂停")
}

// Resume 继续执行
func (e *TaskExecutor) Resume() {
	e.mu.Lock()
	if !e.paused {
		e.mu.Unlock()
		return
	}
	e.paused = false
	e.mu.Unlock()

	// 恢复根节点
	e.root.Resume()

	// 发送继续信号
	select {
	case e.resumeCh <- struct{}{}:
	default:
	}

	Display.ShowMessage("▶️", "任务继续执行")
}

// IsPaused 检查是否已暂停
func (e *TaskExecutor) IsPaused() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.paused
}

// checkPausePoint 检查暂停点
func (e *TaskExecutor) checkPausePoint() {
	// 检查是否收到暂停信号
	select {
	case <-e.pauseCh:
		// 保存检查点
		if err := e.saveCheckpoint(); err != nil {
			Display.ShowMessage("⚠️", fmt.Sprintf("保存检查点失败: %v", err))
		}
		// 等待继续信号
		<-e.resumeCh
	default:
		// 没有暂停信号，继续执行
	}
}

// preflight 预检阶段：加载进度、特性清单，注入上下文
func (e *TaskExecutor) preflight() {
	// 1. 加载进度文件
	if e.taskFolder != "" {
		if pf, err := LoadProgress(e.taskFolder); err == nil && len(pf.Entries) > 0 {
			progressCtx := BuildProgressContext(pf)
			if progressCtx != "" {
				e.root.Context.Variables["progress_context"] = progressCtx
				Display.ShowMessage("📋", fmt.Sprintf("已加载 %d 条进度记录", len(pf.Entries)))
			}
		}
	}

	// 2. 加载或生成特性清单
	e.loadOrGenerateFeatureList()
}

// loadOrGenerateFeatureList 加载已有特性清单，或为新任务生成
func (e *TaskExecutor) loadOrGenerateFeatureList() {
	if e.taskFolder == "" {
		return
	}

	// 尝试加载已有特性清单
	fl, err := LoadFeatureList(e.taskFolder)
	if err == nil && fl != nil && len(fl.Features) > 0 {
		flCtx := BuildFeatureListContext(fl)
		e.root.Context.Variables["feature_list"] = flCtx
		total, passed, remaining := GetFeatureStats(fl)
		Display.ShowMessage("📋", fmt.Sprintf("特性清单: %d/%d 已完成, %d 待完成", passed, total, remaining))
		return
	}

	// 新任务：用 LLM 生成特性清单
	Display.ShowMessage("🔍", "正在生成特性清单...")
	fl, err = GenerateFeatureList(e.ctx, e.root.Description)
	if err != nil {
		Display.ShowMessage("⚠️", fmt.Sprintf("生成特性清单失败: %v", err))
		return
	}

	fl.TaskID = e.root.ID

	if err := SaveFeatureList(e.taskFolder, fl); err != nil {
		Display.ShowMessage("⚠️", fmt.Sprintf("保存特性清单失败: %v", err))
	}

	flCtx := BuildFeatureListContext(fl)
	e.root.Context.Variables["feature_list"] = flCtx
	Display.ShowMessage("✅", fmt.Sprintf("已生成 %d 个特性清单项", len(fl.Features)))
}

// saveCheckpoint 保存检查点
func (e *TaskExecutor) saveCheckpoint() error {
	// 传入 taskFolder
	checkpointPath, err := SaveCheckpoint(e.root, e.taskFolder)
	if err != nil {
		return fmt.Errorf("保存检查点失败: %w", err)
	}
	Display.ShowMessage("💾", fmt.Sprintf("检查点已保存: %s", checkpointPath))
	return nil
}

// SetRecoveryMode 设置恢复模式
func (e *TaskExecutor) SetRecoveryMode(taskFolder string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.recovering = true
	e.taskFolder = taskFolder
}

// joinStrings 连接字符串
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// setNodeOutputPath 设置节点输出路径（用于树形目录结构）
func (e *TaskExecutor) setNodeOutputPath(node *TaskNode) {
	nodePath := e.buildNodePath(node)
	// 计算绝对路径: OutputDir/TaskFolder/doc/NodePath
	baseDir := filepath.Join(config.GetOutputDir(), e.taskFolder, "doc")
	if nodePath != "" {
		node.OutputPath = filepath.Join(baseDir, nodePath)
	} else {
		node.OutputPath = baseDir
	}
	fmt.Printf("[DEBUG] setNodeOutputPath: node=%s, path=%s\n", node.Title, node.OutputPath)
	// mcp.SetNodePath(path) // 移除
}

// buildNodePath 构建节点路径（从根节点的直接子节点到当前节点）
func (e *TaskExecutor) buildNodePath(node *TaskNode) string {
	// 根节点不需要路径
	if node.ID == e.root.ID {
		return ""
	}

	// 收集从当前节点向上到根节点的路径（不包含根节点）
	var pathParts []string
	current := node

	for current != nil && current.ID != e.root.ID {
		pathParts = append([]string{sanitizeForFilename(current.Title)}, pathParts...)
		current = e.findParentNode(current)
	}

	if len(pathParts) == 0 {
		return ""
	}

	return joinStrings(pathParts, "/")
}

// findParentNode 查找父节点
func (e *TaskExecutor) findParentNode(node *TaskNode) *TaskNode {
	if node.ParentID == "" {
		return nil
	}
	return e.findNodeByID(e.root, node.ParentID)
}

// findNodeByID 递归查找节点
func (e *TaskExecutor) findNodeByID(root *TaskNode, id string) *TaskNode {
	if root.ID == id {
		return root
	}
	for _, child := range root.Children {
		if found := e.findNodeByID(child, id); found != nil {
			return found
		}
	}
	return nil
}
