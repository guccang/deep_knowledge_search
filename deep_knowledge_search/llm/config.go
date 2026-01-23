// Package llm provides LLM (Large Language Model) client functionality.
package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigFileName 配置文件名
const ConfigFileName = "config.json"

// ModelConfig 模型配置
type ModelConfig struct {
	Name        string  `json:"name"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

// LLMConfig holds LLM API configuration
type LLMConfig struct {
	Models       map[string]ModelConfig
	CurrentModel string

	// 兼容旧字段 (Deprecated)
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

// Global configuration instance
var llmConfig = LLMConfig{
	Models: make(map[string]ModelConfig),
}

// GetConfig returns the current LLM configuration
func GetConfig() *LLMConfig {
	return &llmConfig
}

// GetCurrentModelConfig 获取当前使用的模型配置
func GetCurrentModelConfig() ModelConfig {
	if config, ok := llmConfig.Models[llmConfig.CurrentModel]; ok {
		return config
	}
	// Fallback to legacy fields if no model selected or found
	return ModelConfig{
		Name:        "legacy",
		APIKey:      llmConfig.APIKey,
		BaseURL:     llmConfig.BaseURL,
		Model:       llmConfig.Model,
		Temperature: llmConfig.Temperature,
	}
}

// InitWithConfig initializes with explicit config values
func InitWithConfig(models []ModelConfig, defaultModel string) error {
	llmConfig.Models = make(map[string]ModelConfig)
	for _, m := range models {
		llmConfig.Models[m.Name] = m
	}
	llmConfig.CurrentModel = defaultModel

	// 同时也设置旧字段作为后备
	current := GetCurrentModelConfig()
	llmConfig.APIKey = current.APIKey
	llmConfig.BaseURL = current.BaseURL
	llmConfig.Model = current.Model
	llmConfig.Temperature = current.Temperature

	fmt.Printf("[LLM] 配置完成: current=%s, model=%s\n", llmConfig.CurrentModel, current.Model)
	return nil
}

// InitConfig initializes the LLM configuration
// Priority: config file > environment variables > defaults
func InitConfig() error {
	// 1. 尝试从配置文件读取
	if err := loadConfigFromFile(); err == nil {
		fmt.Printf("[LLM] 从配置文件加载: model=%s, baseURL=%s\n", llmConfig.Model, llmConfig.BaseURL)
		return nil
	}

	// 2. 从环境变量读取
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("DEEPSEEK_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1/chat/completions"
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}

	// 3. 检查 API Key
	if apiKey == "" {
		// 创建示例配置文件
		createExampleConfig()
		return fmt.Errorf("请在 config.json 中配置 api_key，或设置环境变量 OPENAI_API_KEY")
	}

	// 从环境变量构建默认模型配置
	defaultConfig := ModelConfig{
		Name:        "env_default",
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Model:       model,
		Temperature: 0.3,
	}

	llmConfig = LLMConfig{
		Models: map[string]ModelConfig{
			"env_default": defaultConfig,
		},
		CurrentModel: "env_default",
		// Legacy fields
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Model:       model,
		Temperature: 0.3,
	}

	fmt.Printf("[LLM] 从环境变量加载: model=%s, baseURL=%s\n", llmConfig.Model, llmConfig.BaseURL)
	return nil
}

// loadConfigFromFile 从配置文件加载配置
func loadConfigFromFile() error {
	// 查找配置文件路径
	configPath := findConfigFile()
	if configPath == "" {
		return fmt.Errorf("config file not found")
	}

	// 读取文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// 解析 JSON
	var config LLMConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("配置文件格式错误: %w", err)
	}

	// 验证必填字段
	if config.APIKey == "" {
		return fmt.Errorf("config file missing api_key")
	}

	// 设置默认值
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com/v1/chat/completions"
	}
	if config.Model == "" {
		config.Model = "deepseek-chat"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.3
	}

	llmConfig = config
	return nil
}

// findConfigFile 查找配置文件
func findConfigFile() string {
	// 1. 当前目录
	if _, err := os.Stat(ConfigFileName); err == nil {
		return ConfigFileName
	}

	// 2. 可执行文件同目录
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		exeConfig := filepath.Join(exeDir, ConfigFileName)
		if _, err := os.Stat(exeConfig); err == nil {
			return exeConfig
		}
	}

	// 3. 用户目录/.dks/config.json
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".dks", ConfigFileName)
		if _, err := os.Stat(homeConfig); err == nil {
			return homeConfig
		}
	}

	return ""
}

// createExampleConfig 创建示例配置文件
func createExampleConfig() {
	exampleConfig := LLMConfig{
		APIKey:      "your-api-key-here",
		BaseURL:     "https://api.deepseek.com/v1/chat/completions",
		Model:       "deepseek-chat",
		Temperature: 0.3,
	}

	data, err := json.MarshalIndent(exampleConfig, "", "  ")
	if err != nil {
		return
	}

	// 写入示例配置
	if err := os.WriteFile(ConfigFileName, data, 0644); err == nil {
		fmt.Printf("[LLM] 已创建示例配置文件: %s\n", ConfigFileName)
	}
}

// SaveConfig 保存当前配置到文件
func SaveConfig() error {
	data, err := json.MarshalIndent(llmConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFileName, data, 0644)
}
