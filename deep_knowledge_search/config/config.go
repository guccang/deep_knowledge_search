package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigFileName 配置文件名
const ConfigFileName = "config.json"

// AppConfig 应用配置
type AppConfig struct {
	// LLM 配置
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`

	// Web 配置
	WebPort    int  `json:"web_port"`
	WebEnabled bool `json:"web_enabled"`

	// 输出配置
	OutputDir string `json:"output_dir"` // 文档输出目录，默认为 "output"
}

var appConfig = AppConfig{}

// GetConfig 获取配置
func GetConfig() *AppConfig {
	return &appConfig
}

// GetOutputDir 获取配置的输出目录
func GetOutputDir() string {
	if appConfig.OutputDir == "" {
		return "output" // 默认值
	}
	return appConfig.OutputDir
}

// LoadConfig 加载配置
func LoadConfig() error {
	configPath := findConfigFile()
	if configPath == "" {
		createExampleConfig()
		return fmt.Errorf("请在 config.json 中配置")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &appConfig); err != nil {
		return fmt.Errorf("配置文件格式错误: %w", err)
	}

	// 设置默认值
	if appConfig.BaseURL == "" {
		appConfig.BaseURL = "https://api.deepseek.com/v1/chat/completions"
	}
	if appConfig.Model == "" {
		appConfig.Model = "deepseek-chat"
	}
	if appConfig.Temperature == 0 {
		appConfig.Temperature = 0.3
	}
	if appConfig.WebPort == 0 {
		appConfig.WebPort = 8080
	}
	if appConfig.OutputDir == "" {
		appConfig.OutputDir = "output"
	}

	fmt.Printf("[Config] 加载完成: model=%s, web_port=%d, web_enabled=%v\n",
		appConfig.Model, appConfig.WebPort, appConfig.WebEnabled)

	return nil
}

// findConfigFile 查找配置文件
func findConfigFile() string {
	if _, err := os.Stat(ConfigFileName); err == nil {
		return ConfigFileName
	}

	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		exeConfig := filepath.Join(exeDir, ConfigFileName)
		if _, err := os.Stat(exeConfig); err == nil {
			return exeConfig
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".dks", ConfigFileName)
		if _, err := os.Stat(homeConfig); err == nil {
			return homeConfig
		}
	}

	return ""
}

// createExampleConfig 创建示例配置
func createExampleConfig() {
	example := AppConfig{
		APIKey:      "your-api-key-here",
		BaseURL:     "https://api.deepseek.com/v1/chat/completions",
		Model:       "deepseek-chat",
		Temperature: 0.3,
		WebPort:     8080,
		WebEnabled:  true,
		OutputDir:   "output",
	}

	data, _ := json.MarshalIndent(example, "", "  ")
	os.WriteFile(ConfigFileName, data, 0644)
	fmt.Printf("[Config] 已创建示例配置文件: %s\n", ConfigFileName)
}
