package config

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

// AppConfig 应用配置
type AppConfig struct {
	// LLM 多模型配置
	Models       []ModelConfig `json:"models"`
	DefaultModel string        `json:"default_model"`

	// 兼容旧配置 (Deprecated)
	APIKey      string  `json:"api_key,omitempty"`
	BaseURL     string  `json:"base_url,omitempty"`
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`

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

	// 兼容性处理：如果 Models 为空但有旧配置，则迁移
	if len(appConfig.Models) == 0 && appConfig.APIKey != "" {
		appConfig.Models = []ModelConfig{
			{
				Name:        "default",
				APIKey:      appConfig.APIKey,
				BaseURL:     appConfig.BaseURL,
				Model:       appConfig.Model,
				Temperature: appConfig.Temperature,
			},
		}
		appConfig.DefaultModel = "default"
	}

	// 设置默认值
	if len(appConfig.Models) == 0 {
		// 如果完全没有配置，添加默认的 DeepSeek 配置
		appConfig.Models = []ModelConfig{
			{
				Name:        "deepseek",
				APIKey:      "", // 需要用户填写
				BaseURL:     "https://api.deepseek.com/v1/chat/completions",
				Model:       "deepseek-chat",
				Temperature: 0.3,
			},
		}
		appConfig.DefaultModel = "deepseek"
	}

	if appConfig.DefaultModel == "" && len(appConfig.Models) > 0 {
		appConfig.DefaultModel = appConfig.Models[0].Name
	}

	if appConfig.WebPort == 0 {
		appConfig.WebPort = 8080
	}
	if appConfig.OutputDir == "" {
		appConfig.OutputDir = "output"
	}

	fmt.Printf("[Config] 加载完成: models=%d, default=%s, web_port=%d\n",
		len(appConfig.Models), appConfig.DefaultModel, appConfig.WebPort)

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
		Models: []ModelConfig{
			{
				Name:        "deepseek",
				APIKey:      "your-api-key-here",
				BaseURL:     "https://api.deepseek.com/v1/chat/completions",
				Model:       "deepseek-chat",
				Temperature: 0.3,
			},
		},
		DefaultModel: "deepseek",
		WebPort:      8080,
		WebEnabled:   true,
		OutputDir:    "output",
	}

	data, _ := json.MarshalIndent(example, "", "  ")
	os.WriteFile(ConfigFileName, data, 0644)
	fmt.Printf("[Config] 已创建示例配置文件: %s\n", ConfigFileName)
}
