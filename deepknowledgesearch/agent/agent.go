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

	// Initialize LLM with config
	if err := llm.InitWithConfig(cfg.APIKey, cfg.BaseURL, cfg.Model, cfg.Temperature); err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	fmt.Println("[Agent] Initialized")
	return nil
}

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
