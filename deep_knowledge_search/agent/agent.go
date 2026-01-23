// Package agent provides task orchestration for the Deep Knowledge Search system.
package agent

import (
	"deepknowledgesearch/config"
	"deepknowledgesearch/llm"
	"deepknowledgesearch/mcp"
	"fmt"
)

// Init initializes the Agent module (legacy compatibility)
func Init() error {
	// Initialize MCP
	mcp.Init()

	// Initialize LLM
	if err := llm.Init(); err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	fmt.Println("[Agent] Initialized")
	return nil
}

// InitWithConfig initializes the Agent with config
func InitWithConfig(cfg *config.AppConfig) error {
	// Initialize MCP
	mcp.Init()

	// Convert config models to llm models
	var llmModels []llm.ModelConfig
	for _, m := range cfg.Models {
		llmModels = append(llmModels, llm.ModelConfig{
			Name:        m.Name,
			APIKey:      m.APIKey,
			BaseURL:     m.BaseURL,
			Model:       m.Model,
			Temperature: m.Temperature,
		})
	}

	// Initialize LLM with config
	if err := llm.InitWithConfig(llmModels, cfg.DefaultModel); err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	fmt.Println("[Agent] Initialized")
	return nil
}

// ExecutorRegistrationFunc 执行器注册函数类型
type ExecutorRegistrationFunc func(taskID string, executor interface{})

// OnExecutorCreated 执行器创建后的回调（用于外部注册任务管理器）
var OnExecutorCreated ExecutorRegistrationFunc

// OnExecutorFinished 执行器完成后的回调（用于清理）
var OnExecutorFinished ExecutorRegistrationFunc

// RunTask executes a task given its description
func RunTask(description string) error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("[Agent] 开始执行任务: %s\n", description)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Create task planner
	planner := NewTaskPlanner()

	// Execute task
	result, err := planner.ExecuteTask(description)
	if err != nil {
		return fmt.Errorf("任务执行失败: %w", err)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("[Agent] 任务完成")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if result != "" {
		fmt.Println("\n📋 执行结果:")
		fmt.Println(result)
	}

	return nil
}
