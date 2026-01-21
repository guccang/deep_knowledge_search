package main

import (
	"deepknowledgesearch/agent"
	"deepknowledgesearch/config"
	"deepknowledgesearch/web"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           知识深度搜索 - Deep Knowledge Search             ║")
	fmt.Println("║                     v1.0.0                               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 加载配置
	if err := config.LoadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ 配置加载: %v\n", err)
	}
	cfg := config.GetConfig()

	// 启动 Web Dashboard
	if cfg.WebEnabled && cfg.WebPort > 0 {
		web.InitServer(cfg.WebPort)
		if err := web.StartServer(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Web服务启动失败: %v\n", err)
		}
	}

	// Initialize agent
	if err := agent.InitWithConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 初始化失败: %v\n", err)
		fmt.Println("\n💡 提示: 请在 config.json 中配置 api_key")
		os.Exit(1)
	}

	// Check for command line arguments
	if len(os.Args) > 1 {
		// Join all arguments as the task description
		taskDescription := strings.Join(os.Args[1:], " ")
		if err := agent.RunTask(taskDescription); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 任务执行失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Interactive mode
	fmt.Println("📝 请输入您的任务描述（输入 'exit' 或 'quit' 退出）:")
	fmt.Println()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "🔍 > ",
		HistoryFile:     "/tmp/deep_knowledge_search.history",
		AutoComplete:    nil,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Readline error: %v\n", err)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					break
				}
				continue
			} else if err == io.EOF {
				break
			}
			continue
		}

		// Trim whitespace
		input := strings.TrimSpace(line)

		// Check for exit commands
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" || input == "q" {
			fmt.Println("👋 再见！")
			break
		}

		// Run the task
		if err := agent.RunTask(input); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 任务执行失败: %v\n", err)
		}

		fmt.Println()
	}
}
