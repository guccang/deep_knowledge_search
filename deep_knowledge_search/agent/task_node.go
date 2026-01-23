package agent

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// 执行模式和状态常量
// ============================================================================

// ExecutionMode 执行模式
type ExecutionMode string

const (
	ModeSequential ExecutionMode = "sequential" // 串行执行
	ModeParallel   ExecutionMode = "parallel"   // 并行执行
)

// NodeStatus 节点状态
type NodeStatus string

const (
	NodePending  NodeStatus = "pending"  // 等待中
	NodeRunning  NodeStatus = "running"  // 执行中
	NodePaused   NodeStatus = "paused"   // 已暂停
	NodeDone     NodeStatus = "done"     // 已完成
	NodeFailed   NodeStatus = "failed"   // 失败
	NodeCanceled NodeStatus = "canceled" // 已取消
)

// LogLevel 日志级别
type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// 配置常量
const (
	DefaultMaxDepth   = 3 // 默认最大递归深度
	DefaultMaxRetries = 3 // 默认最大重试次数
)

// ============================================================================
// TaskNode - 任务节点
// ============================================================================

// TaskNode 任务节点（支持递归子任务）
type TaskNode struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Depth    int    `json:"depth"`

	// 任务描述
	Title       string `json:"title"`
	Description string `json:"description"`
	Goal        string `json:"goal"`

	// OutputPath 节点输出目录（绝对路径）
	OutputPath string `json:"output_path,omitempty"`

	// 执行配置
	ExecutionMode ExecutionMode `json:"execution_mode"`
	ToolCalls     []string      `json:"tool_calls,omitempty"`
	MaxRetries    int           `json:"max_retries"`
	RetryCount    int           `json:"retry_count"`
	CanDecompose  bool          `json:"can_decompose"`
	DependsOn     []string      `json:"depends_on,omitempty"`

	// 子节点
	Children []*TaskNode `json:"children,omitempty"`

	// 状态与进度
	Status   NodeStatus `json:"status"`
	Progress float64    `json:"progress"`

	// 上下文与结果
	Context *TaskContext `json:"context"`
	Result  *TaskResult  `json:"result,omitempty"`

	// 验证结果
	Verification *VerificationInfo `json:"verification,omitempty"`

	// 时间信息
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// 日志
	Logs []ExecutionLog `json:"logs"`

	// LLM 调用记录
	LLMCalls []LLMCallRecord `json:"llm_calls,omitempty"`

	// 内部控制
	mu       sync.RWMutex  `json:"-"`
	cancelCh chan struct{} `json:"-"`
}

// VerificationInfo 验证信息
type VerificationInfo struct {
	Passed     bool                  `json:"passed"`
	Iterations int                   `json:"iterations"`
	Attempts   []VerificationAttempt `json:"attempts,omitempty"`
}

// VerificationAttempt 单次验证尝试
type VerificationAttempt struct {
	Iteration int    `json:"iteration"`
	Passed    bool   `json:"passed"`
	Feedback  string `json:"feedback"`
	Timestamp string `json:"timestamp"`
}

// LLMCallRecord LLM 调用记录
type LLMCallRecord struct {
	Type       string                   `json:"type"`        // "plan", "execute", "synthesize", "verify"
	Messages   []map[string]interface{} `json:"messages"`    // 请求消息
	Response   string                   `json:"response"`    // 响应内容
	StartTime  time.Time                `json:"start_time"`  // 开始时间
	DurationMs int64                    `json:"duration_ms"` // 耗时（毫秒）
}

// NewTaskNode 创建新任务节点
func NewTaskNode(title, description string) *TaskNode {
	id := generateNodeID()
	return &TaskNode{
		ID:            id,
		Depth:         0,
		Title:         title,
		Description:   description,
		ExecutionMode: ModeSequential,
		MaxRetries:    DefaultMaxRetries,
		CanDecompose:  true,
		Status:        NodePending,
		Progress:      0,
		Context:       NewTaskContext(description),
		Logs:          []ExecutionLog{},
		CreatedAt:     time.Now(),
		cancelCh:      make(chan struct{}, 1),
	}
}

// NewChildNode 创建子节点
func (n *TaskNode) NewChildNode(title, description, goal string) *TaskNode {
	child := &TaskNode{
		ID:            generateNodeID(),
		ParentID:      n.ID,
		Depth:         n.Depth + 1,
		Title:         title,
		Description:   description,
		Goal:          goal,
		ExecutionMode: ModeSequential,
		MaxRetries:    DefaultMaxRetries,
		CanDecompose:  true,
		Status:        NodePending,
		Progress:      0,
		Context:       NewTaskContext(description),
		Logs:          []ExecutionLog{},
		CreatedAt:     time.Now(),
		cancelCh:      make(chan struct{}, 1),
	}

	// 继承父节点的用户输入
	child.Context.UserInput = n.Context.UserInput

	n.mu.Lock()
	n.Children = append(n.Children, child)
	n.mu.Unlock()

	return child
}

// AddLog 添加执行日志
func (n *TaskNode) AddLog(level LogLevel, phase, message string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Logs = append(n.Logs, ExecutionLog{
		Time:    time.Now(),
		Level:   level,
		Phase:   phase,
		Message: message,
		NodeID:  n.ID,
	})
}

// AddLLMCall 添加 LLM 调用记录
func (n *TaskNode) AddLLMCall(callType string, messages []map[string]interface{}, response string, startTime time.Time, durationMs int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.LLMCalls = append(n.LLMCalls, LLMCallRecord{
		Type:       callType,
		Messages:   messages,
		Response:   response,
		StartTime:  startTime,
		DurationMs: durationMs,
	})
}

// SetStatus 设置状态
func (n *TaskNode) SetStatus(status NodeStatus) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Status = status
	if status == NodeRunning && n.StartedAt == nil {
		now := time.Now()
		n.StartedAt = &now
	}
	if status == NodeDone || status == NodeFailed || status == NodeCanceled {
		now := time.Now()
		n.FinishedAt = &now
	}
}

// SetProgress 设置进度
func (n *TaskNode) SetProgress(progress float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Progress = progress
}

// GetStatus 获取状态（线程安全）
func (n *TaskNode) GetStatus() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Status
}

// CanRetry 检查是否可以重试
func (n *TaskNode) CanRetry() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.RetryCount < n.MaxRetries
}

// IncrementRetry 增加重试计数
func (n *TaskNode) IncrementRetry() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.RetryCount++
}

// IsCanceled 检查是否已取消
func (n *TaskNode) IsCanceled() bool {
	select {
	case <-n.cancelCh:
		return true
	default:
		return false
	}
}

// Cancel 取消节点
func (n *TaskNode) Cancel() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.Status == NodePending || n.Status == NodeRunning {
		n.Status = NodeCanceled
		close(n.cancelCh)
		for _, child := range n.Children {
			child.Cancel()
		}
	}
}

// Pause 暂停节点
func (n *TaskNode) Pause() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.Status == NodeRunning {
		n.Status = NodePaused
		// 递归暂停所有子节点
		for _, child := range n.Children {
			child.Pause()
		}
	}
}

// Resume 恢复节点
func (n *TaskNode) Resume() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.Status == NodePaused {
		n.Status = NodeRunning
		// 递归恢复所有子节点
		for _, child := range n.Children {
			child.Resume()
		}
	}
}

// IsPaused 检查是否已暂停
func (n *TaskNode) IsPaused() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Status == NodePaused
}

// generateNodeID 生成节点ID
func generateNodeID() string {
	return uuid.New().String()[:8]
}

// ============================================================================
// TaskContext - 任务上下文
// ============================================================================

// TaskContext 任务上下文
type TaskContext struct {
	UserInput      string                 `json:"user_input"`
	ParentResults  []ParentResult         `json:"parent_results,omitempty"`
	SiblingResults []SiblingResult        `json:"sibling_results,omitempty"`
	Variables      map[string]interface{} `json:"variables,omitempty"`
}

// ParentResult 父任务结果摘要
type ParentResult struct {
	NodeID  string `json:"node_id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// SiblingResult 兄弟任务结果
type SiblingResult struct {
	NodeID  string     `json:"node_id"`
	Title   string     `json:"title"`
	Status  NodeStatus `json:"status"`
	Summary string     `json:"summary"`
}

// NewTaskContext 创建任务上下文
func NewTaskContext(userInput string) *TaskContext {
	return &TaskContext{
		UserInput:      userInput,
		ParentResults:  []ParentResult{},
		SiblingResults: []SiblingResult{},
		Variables:      make(map[string]interface{}),
	}
}

// AddParentResult 添加父任务结果
func (c *TaskContext) AddParentResult(nodeID, title, summary string) {
	c.ParentResults = append(c.ParentResults, ParentResult{
		NodeID:  nodeID,
		Title:   title,
		Summary: summary,
	})
}

// AddSiblingResult 添加兄弟任务结果
func (c *TaskContext) AddSiblingResult(nodeID, title string, status NodeStatus, summary string) {
	c.SiblingResults = append(c.SiblingResults, SiblingResult{
		NodeID:  nodeID,
		Title:   title,
		Status:  status,
		Summary: summary,
	})
}

// BuildLLMContext 构建 LLM 请求的上下文字符串
func (c *TaskContext) BuildLLMContext() string {
	var sb strings.Builder

	sb.WriteString("## 原始用户请求\n")
	sb.WriteString(c.UserInput)
	sb.WriteString("\n\n")

	if len(c.ParentResults) > 0 {
		sb.WriteString("## 父任务执行结果\n")
		for _, pr := range c.ParentResults {
			sb.WriteString("- ")
			sb.WriteString(pr.Title)
			sb.WriteString(": ")
			sb.WriteString(pr.Summary)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(c.SiblingResults) > 0 {
		sb.WriteString("## 已完成的同级任务\n")
		for _, sr := range c.SiblingResults {
			sb.WriteString("- ")
			sb.WriteString(sr.Title)
			sb.WriteString(" [")
			sb.WriteString(string(sr.Status))
			sb.WriteString("]: ")
			sb.WriteString(sr.Summary)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ============================================================================
// TaskResult - 任务结果
// ============================================================================

// TaskResult 任务执行结果
type TaskResult struct {
	Success   bool                   `json:"success"`
	Output    string                 `json:"output"`
	Summary   string                 `json:"summary"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Artifacts []string               `json:"artifacts,omitempty"`
}

// NewTaskResult 创建成功结果
func NewTaskResult(output, summary string) *TaskResult {
	return &TaskResult{
		Success: true,
		Output:  output,
		Summary: summary,
		Data:    make(map[string]interface{}),
	}
}

// NewTaskResultError 创建失败结果
func NewTaskResultError(err string) *TaskResult {
	return &TaskResult{
		Success: false,
		Error:   err,
	}
}

// ============================================================================
// ExecutionLog - 执行日志
// ============================================================================

// ExecutionLog 执行日志
type ExecutionLog struct {
	Time    time.Time `json:"time"`
	Level   LogLevel  `json:"level"`
	Phase   string    `json:"phase"`
	Message string    `json:"message"`
	NodeID  string    `json:"node_id"`
}

// ============================================================================
// ExecutionConfig - 执行配置
// ============================================================================

// ExecutionConfig 执行配置
type ExecutionConfig struct {
	MaxDepth      int  `json:"max_depth"`
	MaxRetries    int  `json:"max_retries"`
	EnableLogging bool `json:"enable_logging"`
}

// DefaultExecutionConfig 默认执行配置
func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		MaxDepth:      DefaultMaxDepth,
		MaxRetries:    DefaultMaxRetries,
		EnableLogging: true,
	}
}
