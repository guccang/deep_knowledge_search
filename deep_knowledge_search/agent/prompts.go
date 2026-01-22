package agent

import "fmt"

// ============================================================================
// 提示词模板
// ============================================================================

// PromptPlanningSystem 规划系统提示词
var PromptPlanningSystem = `你是一个任务规划专家。你的职责是将复杂任务分解为可执行的子任务。

重要规则:
1. 分析任务的复杂度和依赖关系
2. 选择合适的执行模式（串行/并行）
3. 返回严格的 JSON 格式`

// PromptExecutionSystem 执行系统提示词
var PromptExecutionSystem = `你是一个任务执行助手。

重要规则:
1. 使用可用工具完成任务
2. 调用完工具后返回简洁的执行结果`

// PromptNodePlanning 节点规划提示词模板
var PromptNodePlanning = `请将以下任务分解为子任务。

## 任务信息
标题: %s
描述: %s
目标: %s

## 上下文
%s

## 可用工具
%s

## 规则
1. 子任务 1-10 个
2. **优先使用并行模式**：execution_mode 默认选择 "parallel"
3. 仅当子任务之间有明确的依赖关系时才选择 "sequential"
4. can_decompose: true 表示复杂子任务可继续拆解
5. 简单任务返回空 subtasks 数组

## 返回 JSON 格式（无 markdown 代码块）
{
  "title": "任务标题",
  "goal": "期望目标",
  "execution_mode": "parallel",
  "subtasks": [
    {
      "title": "子任务标题",
      "description": "详细描述",
      "goal": "子任务目标",
      "tools": ["工具名"],
      "can_decompose": false
    }
  ],
  "reasoning": "选择执行模式的原因"
}`

// PromptNodeExecution 节点执行提示词模板
var PromptNodeExecution = `执行以下任务并返回结果。

## 任务信息
标题: %s
描述: %s
目标: %s

## 上下文
%s

## 规则
1. 使用可用工具完成任务
2. 返回简洁明了的结果
3. 如果需要保存内容，使用 saveToDisk 工具`

// PromptResultSynthesis 结果整合提示词模板
var PromptResultSynthesis = `请将以下子任务结果整合为一个清晰的最终结果。

## 父任务
标题: %s
目标: %s

## 子任务结果
%s

## 规则
1. 提取关键信息
2. 合并相关内容
3. 返回简洁的结果摘要`

// PromptVerificationSystem 验证系统提示词
var PromptVerificationSystem = `你是一个任务验证专家。你的职责是验证任务执行结果是否符合预期目标。

重要规则:
1. 仔细检查执行结果是否完整满足任务目标
2. 如果验证通过，必须在响应中包含 "VERIFICATION_PASSED"
3. 如果验证不通过，说明原因并给出改进建议
4. 验证标准要合理，不要过于苛刻`

// PromptVerification 验证提示词模板
var PromptVerification = `请验证以下任务执行结果是否符合目标要求。

## 原始任务
标题: %s
目标: %s

## 执行结果
%s

## 验证规则
1. 检查结果是否完整满足任务目标
2. 检查是否有遗漏或错误
3. 检查输出格式是否正确

## 响应格式
如果验证通过，必须包含: VERIFICATION_PASSED
如果验证不通过，说明:
- 不通过原因
- 改进建议
- 需要补充的内容`

// ============================================================================
// 提示词构建函数
// ============================================================================

// BuildNodePlanningPrompt 构建节点规划提示词
func BuildNodePlanningPrompt(title, description, goal, context, tools string) string {
	return fmt.Sprintf(PromptNodePlanning, title, description, goal, context, tools)
}

// BuildNodeExecutionPrompt 构建节点执行提示词
func BuildNodeExecutionPrompt(title, description, goal, context string) string {
	return fmt.Sprintf(PromptNodeExecution, title, description, goal, context)
}

// BuildResultSynthesisPrompt 构建结果整合提示词
func BuildResultSynthesisPrompt(title, goal, childResults string) string {
	return fmt.Sprintf(PromptResultSynthesis, title, goal, childResults)
}

// BuildVerificationPrompt 构建验证提示词
func BuildVerificationPrompt(title, goal, result string) string {
	return fmt.Sprintf(PromptVerification, title, goal, result)
}
