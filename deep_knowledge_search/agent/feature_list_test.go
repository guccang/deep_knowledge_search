package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFeatureListSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, LogSubDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	featurePath := filepath.Join(logDir, featureListFileName)

	fl := &FeatureList{
		TaskID: "task-1",
		Features: []Feature{
			{ID: "F001", Category: "functional", Description: "用户登录", Steps: []string{"输入账号", "点击登录"}, Priority: 1, Passes: false},
			{ID: "F002", Category: "quality", Description: "性能优化", Steps: []string{"加载时间<1s"}, Priority: 3, Passes: false},
			{ID: "F003", Category: "documentation", Description: "API文档", Steps: []string{"检查文档完整性"}, Priority: 2, Passes: true},
		},
	}

	data, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if err = os.WriteFile(featurePath, data, 0644); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	data, err = os.ReadFile(featurePath)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}

	var loaded FeatureList
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(loaded.Features) != 3 {
		t.Fatalf("期望 3 个特性, 实际 %d", len(loaded.Features))
	}
	if loaded.Features[0].ID != "F001" {
		t.Errorf("特性 ID 不匹配: %s", loaded.Features[0].ID)
	}
}

func TestMarkFeatureComplete(t *testing.T) {
	fl := &FeatureList{
		Features: []Feature{
			{ID: "F001", Passes: false},
			{ID: "F002", Passes: false},
		},
	}

	if !MarkFeatureComplete(fl, "F001") {
		t.Error("标记 F001 应该返回 true")
	}
	if !fl.Features[0].Passes {
		t.Error("F001 应已标记为完成")
	}
	if fl.Features[1].Passes {
		t.Error("F002 不应被标记为完成")
	}

	if MarkFeatureComplete(fl, "F999") {
		t.Error("不存在的特性应返回 false")
	}
}

func TestGetNextFeature(t *testing.T) {
	fl := &FeatureList{
		Features: []Feature{
			{ID: "F001", Priority: 3, Passes: false},
			{ID: "F002", Priority: 1, Passes: false},
			{ID: "F003", Priority: 2, Passes: true},
		},
	}

	next := GetNextFeature(fl)
	if next == nil {
		t.Fatal("不应返回 nil")
	}
	if next.ID != "F002" {
		t.Errorf("应返回优先级最高的未完成特性 F002, 实际: %s", next.ID)
	}

	// 全部完成后
	MarkFeatureComplete(fl, "F001")
	MarkFeatureComplete(fl, "F002")
	next = GetNextFeature(fl)
	if next != nil {
		t.Error("全部完成后应返回 nil")
	}
}

func TestGetFeatureStats(t *testing.T) {
	fl := &FeatureList{
		Features: []Feature{
			{ID: "F001", Passes: true},
			{ID: "F002", Passes: false},
			{ID: "F003", Passes: true},
		},
	}

	total, passed, remaining := GetFeatureStats(fl)
	if total != 3 {
		t.Errorf("总数应为 3, 实际: %d", total)
	}
	if passed != 2 {
		t.Errorf("已完成应为 2, 实际: %d", passed)
	}
	if remaining != 1 {
		t.Errorf("待完成应为 1, 实际: %d", remaining)
	}
}

func TestBuildFeatureListContext(t *testing.T) {
	fl := &FeatureList{
		Features: []Feature{
			{ID: "F001", Description: "用户登录", Priority: 1, Passes: false, Steps: []string{"输入账号", "点击登录"}},
			{ID: "F002", Description: "性能优化", Priority: 2, Passes: true},
		},
	}

	ctx := BuildFeatureListContext(fl)
	if ctx == "" {
		t.Error("上下文不应为空")
	}
	if !stringContains(ctx, "F001") {
		t.Error("上下文应包含 F001")
	}
	if !stringContains(ctx, "1/2 已完成") {
		t.Error("上下文应包含完成统计")
	}
	if !stringContains(ctx, "下一个待完成特性") {
		t.Error("上下文应包含下一个特性建议")
	}
}

func TestBuildFeatureListContextEmpty(t *testing.T) {
	ctx := BuildFeatureListContext(nil)
	if ctx != "" {
		t.Error("nil 应返回空")
	}
	ctx = BuildFeatureListContext(&FeatureList{})
	if ctx != "" {
		t.Error("空清单应返回空")
	}
}
