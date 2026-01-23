package main

import (
	"deepknowledgesearch/agent"
	"deepknowledgesearch/config"
	"deepknowledgesearch/llm"
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

	// 注册任务执行器回调，用于Web API任务管理
	agent.OnExecutorCreated = func(taskID string, executor interface{}) {
		web.RegisterTaskExecutor(taskID, executor)
		fmt.Printf("[Main] ✓ 注册任务: %s\n", taskID)
	}
	agent.OnExecutorFinished = func(taskID string, _ interface{}) {
		web.UnregisterTaskExecutor(taskID)
		fmt.Printf("[Main] ✓ 清理任务: %s\n", taskID)
	}

	// 扫描可恢复的任务（在Web启动后）
	if cfg.WebEnabled && cfg.WebPort > 0 {
		// 注册可恢复任务回调
		web.SetListRecoverableTasksCallback(func() ([]web.RecoverableTaskInfo, error) {
			rm := agent.NewRecoveryManager()
			tasks, err := rm.FindRecoverableTasks()
			if err != nil {
				return nil, err
			}
			result := make([]web.RecoverableTaskInfo, len(tasks))
			for i, t := range tasks {
				result[i] = web.RecoverableTaskInfo{
					TaskID:         t.TaskID,
					Title:          t.Title,
					Status:         string(t.Status),
					CheckpointPath: t.CheckpointPath,
					TaskFolder:     t.TaskFolder,
				}
			}
			return result, nil
		})

		// 注册恢复任务回调
		web.SetRecoverTaskCallback(func(taskFolder string) error {
			node, executor, err := agent.RecoverTaskByFolder(taskFolder)
			if err != nil {
				return err
			}
			// 运行恢复的任务
			go func() {
				if err := executor.Execute(); err != nil {
					fmt.Printf("[Main] 恢复任务执行失败: %v\n", err)
				} else {
					fmt.Printf("[Main] 恢复任务完成: %s\n", node.Title)
				}
			}()
			return nil
		})

		// 扫描并显示可恢复任务
		rm := agent.NewRecoveryManager()
		if tasks, err := rm.FindRecoverableTasks(); err == nil && len(tasks) > 0 {
			fmt.Printf("[Main] 📋 发现 %d 个可恢复的任务:\n", len(tasks))
			for i, task := range tasks {
				fmt.Printf("       %d. %s (状态: %s)\n", i+1, task.Title, task.Status)
			}
			fmt.Println("[Main] 💡 可通过 Web 界面或 API 恢复这些任务")
		}
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

	// Auto-completer
	var completer = readline.NewPrefixCompleter(
		readline.PcItem("/help"),
		readline.PcItem("/exit"),
		readline.PcItem("/quit"),
		readline.PcItem("/modules",
			readline.PcItemDynamic(func(string) []string {
				cfg := llm.GetConfig()
				var models []string
				for name := range cfg.Models {
					models = append(models, name)
				}
				return models
			}),
		),
	)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          fmt.Sprintf("🔍 [%s] > ", llm.GetConfig().CurrentModel),
		HistoryFile:     "/tmp/deep_knowledge_search.history",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Readline error: %v\n", err)
		return
	}
	defer rl.Close()

	for {
		// Update prompt with current model
		rl.SetPrompt(fmt.Sprintf("🔍 [%s] > ", llm.GetConfig().CurrentModel))

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

		// Handle commands
		if strings.HasPrefix(input, "/") {
			parts := strings.Fields(input)
			cmd := parts[0]

			switch cmd {
			case "/exit", "/quit", "/q":
				fmt.Println("👋 再见！")
				return
			case "/help":
				fmt.Println("📚 可用命令:")
				fmt.Println("  /modules          - 列出所有可用模型")
				fmt.Println("  /modules <name>   - 切换到指定模型")
				fmt.Println("  /help             - 显示帮助信息")
				fmt.Println("  /exit, /quit      - 退出程序")
				continue
			case "/modules":
				cfg := llm.GetConfig()
				if len(parts) > 1 {
					// Switch model
					targetModel := parts[1]
					if _, ok := cfg.Models[targetModel]; ok {
						cfg.CurrentModel = targetModel
						// Update legacy fields for compatibility
						current := llm.GetCurrentModelConfig()
						cfg.APIKey = current.APIKey
						cfg.BaseURL = current.BaseURL
						cfg.Model = current.Model
						cfg.Temperature = current.Temperature
						fmt.Printf("✅ 已切换到模型: %s (%s)\n", targetModel, current.Model)
					} else {
						fmt.Printf("❌ 未知模型: %s\n", targetModel)
						fmt.Println("💡 可用模型:")
						for name := range cfg.Models {
							fmt.Printf("  - %s\n", name)
						}
					}
				} else {
					// List models
					fmt.Println("🤖 可用模型:")
					for name, m := range cfg.Models {
						prefix := "  "
						if name == cfg.CurrentModel {
							prefix = "* "
						}
						fmt.Printf("%s%s (%s)\n", prefix, name, m.Model)
					}
				}
				continue
			default:
				fmt.Printf("❌ 未知命令: %s\n", cmd)
				continue
			}
		}

		// Run the task
		if err := agent.RunTask(input); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 任务执行失败: %v\n", err)
		}

		fmt.Println()
	}
}
