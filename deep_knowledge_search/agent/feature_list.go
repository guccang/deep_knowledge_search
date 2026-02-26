package agent

import (
	"context"
	"deepknowledgesearch/config"
	"deepknowledgesearch/llm"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ============================================================================
// Feature List — 特性清单驱动执行
// ============================================================================
// 参考 Anthropic 文章 "Effective Harnesses for Long-Running Agents"
// 初始化 Agent 生成详细特性清单，后续 Agent 逐项推进

// Feature 特性条目
type Feature struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"` // functional, quality, documentation
	Description string   `json:"description"`
	Steps       []string `json:"steps"`    // 验证步骤
	Priority    int      `json:"priority"` // 优先级 1-5 (1最高)
	Passes      bool     `json:"passes"`   // 是否已完成
}

// FeatureList 特性清单
type FeatureList struct {
	TaskID   string    `json:"task_id"`
	Features []Feature `json:"features"`
}

const featureListFileName = "feature_list.json"

// GenerateFeatureList 使用 LLM 从任务描述生成特性清单
func GenerateFeatureList(ctx context.Context, description string) (*FeatureList, error) {
	prompt := fmt.Sprintf(PromptFeatureListGeneration, description)

	messages := []llm.Message{
		{Role: "system", Content: PromptPlanningSystem},
		{Role: "user", Content: prompt},
	}

	response, err := llm.SendSyncLLMRequest(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("生成特性清单失败: %w", err)
	}

	// 解析 JSON 响应
	cleaned := cleanJSONResponse(response)
	var wrapper struct {
		Features []Feature `json:"features"`
	}
	if err := json.Unmarshal([]byte(cleaned), &wrapper); err != nil {
		return nil, fmt.Errorf("解析特性清单 JSON 失败: %w, response: %s", err, cleaned)
	}

	return &FeatureList{
		Features: wrapper.Features,
	}, nil
}

// LoadFeatureList 从文件加载特性清单
func LoadFeatureList(taskFolder string) (*FeatureList, error) {
	featurePath := getFeatureListPath(taskFolder)

	data, err := os.ReadFile(featurePath)
	if err != nil {
		return nil, fmt.Errorf("读取特性清单失败: %w", err)
	}

	var fl FeatureList
	if err := json.Unmarshal(data, &fl); err != nil {
		return nil, fmt.Errorf("解析特性清单失败: %w", err)
	}

	return &fl, nil
}

// SaveFeatureList 保存特性清单到文件
func SaveFeatureList(taskFolder string, fl *FeatureList) error {
	featurePath := getFeatureListPath(taskFolder)

	dir := filepath.Dir(featurePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	data, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化特性清单失败: %w", err)
	}

	return os.WriteFile(featurePath, data, 0644)
}

// MarkFeatureComplete 标记特性为已完成（只修改 passes 字段）
func MarkFeatureComplete(fl *FeatureList, featureID string) bool {
	for i := range fl.Features {
		if fl.Features[i].ID == featureID {
			fl.Features[i].Passes = true
			return true
		}
	}
	return false
}

// GetNextFeature 获取下一个待完成的最高优先级特性
func GetNextFeature(fl *FeatureList) *Feature {
	var best *Feature
	for i := range fl.Features {
		f := &fl.Features[i]
		if !f.Passes {
			if best == nil || f.Priority < best.Priority {
				best = f
			}
		}
	}
	return best
}

// GetFeatureStats 获取特性统计
func GetFeatureStats(fl *FeatureList) (total, passed, remaining int) {
	total = len(fl.Features)
	for _, f := range fl.Features {
		if f.Passes {
			passed++
		}
	}
	remaining = total - passed
	return
}

// BuildFeatureListContext 构建特性清单上下文字符串（注入到 LLM 对话中）
func BuildFeatureListContext(fl *FeatureList) string {
	if fl == nil || len(fl.Features) == 0 {
		return ""
	}

	total, passed, remaining := GetFeatureStats(fl)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 特性清单 (%d/%d 已完成, %d 待完成)\n\n", passed, total, remaining))

	for _, f := range fl.Features {
		status := "[ ]"
		if f.Passes {
			status = "[x]"
		}
		sb.WriteString(fmt.Sprintf("- %s %s: %s (优先级: %d)\n", status, f.ID, f.Description, f.Priority))
	}

	next := GetNextFeature(fl)
	if next != nil {
		sb.WriteString(fmt.Sprintf("\n### 下一个待完成特性\n**%s**: %s\n", next.ID, next.Description))
		if len(next.Steps) > 0 {
			sb.WriteString("验证步骤:\n")
			for i, step := range next.Steps {
				sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
			}
		}
	}

	return sb.String()
}

// ---------- 内部辅助 ----------

func getFeatureListPath(taskFolder string) string {
	return filepath.Join(config.GetOutputDir(), taskFolder, LogSubDir, featureListFileName)
}
