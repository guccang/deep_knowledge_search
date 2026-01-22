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

// LLMConfig holds LLM API configuration
type LLMConfig struct {
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

// Global configuration instance
var llmConfig = LLMConfig{}

// GetConfig returns the current LLM configuration
func GetConfig() *LLMConfig {
	return &llmConfig
}

// InitWithConfig initializes with explicit config values
func InitWithConfig(apiKey, baseURL, model string, temperature float64) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1/chat/completions"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	if temperature == 0 {
		temperature = 0.3
	}

	llmConfig = LLMConfig{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Model:       model,
		Temperature: temperature,
	}

	fmt.Printf("[LLM] 配置完成: model=%s\n", llmConfig.Model)
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

	llmConfig = LLMConfig{
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
