package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadProgress(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, LogSubDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	progressPath := filepath.Join(logDir, progressFileName)

	entry := ProgressEntry{
		Timestamp:      time.Now(),
		SessionID:      "test-session",
		NodeID:         "node-1",
		NodeTitle:      "测试节点",
		Action:         "completed",
		NodesCompleted: []string{"子任务1", "子任务2"},
		NodesRemaining: []string{"子任务3"},
		Summary:        "完成了前两个子任务",
		NextSteps:      "继续执行子任务3",
	}

	pf := &ProgressFile{
		TaskID:    "task-1",
		TaskTitle: "测试任务",
		Entries:   []ProgressEntry{entry},
	}

	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if err = os.WriteFile(progressPath, data, 0644); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	data, err = os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}

	var loaded ProgressFile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(loaded.Entries) != 1 {
		t.Fatalf("期望 1 条记录, 实际 %d", len(loaded.Entries))
	}
	if loaded.Entries[0].NodeTitle != "测试节点" {
		t.Errorf("节点标题不匹配: %s", loaded.Entries[0].NodeTitle)
	}
	if loaded.Entries[0].Action != "completed" {
		t.Errorf("动作不匹配: %s", loaded.Entries[0].Action)
	}
}

func TestBuildProgressContext(t *testing.T) {
	pf := &ProgressFile{
		TaskID: "task-1",
		Entries: []ProgressEntry{
			{
				Timestamp:      time.Now(),
				NodeTitle:      "研究阶段",
				Action:         "completed",
				NodesCompleted: []string{"研究阶段"},
				NodesRemaining: []string{"编码阶段", "测试阶段"},
				Summary:        "完成了调研",
				NextSteps:      "开始编码",
			},
		},
	}

	ctx := BuildProgressContext(pf)
	if ctx == "" {
		t.Error("上下文不应为空")
	}
	if !stringContains(ctx, "研究阶段") {
		t.Error("上下文应包含节点标题")
	}
	if !stringContains(ctx, "编码阶段") {
		t.Error("上下文应包含待完成节点")
	}
}

func TestBuildProgressContextEmpty(t *testing.T) {
	ctx := BuildProgressContext(nil)
	if ctx != "" {
		t.Error("nil 进度文件应返回空字符串")
	}

	ctx = BuildProgressContext(&ProgressFile{})
	if ctx != "" {
		t.Error("空条目应返回空字符串")
	}
}

func TestCollectNodeStatus(t *testing.T) {
	root := NewTaskNode("根任务", "描述")
	child1 := root.NewChildNode("子任务1", "描述1", "目标1")
	child1.SetStatus(NodeDone)
	root.NewChildNode("子任务2", "描述2", "目标2") // pending

	completed, remaining := CollectNodeStatus(root)

	if len(completed) != 1 || completed[0] != "子任务1" {
		t.Errorf("已完成节点不正确: %v", completed)
	}
	if len(remaining) != 1 || remaining[0] != "子任务2" {
		t.Errorf("待完成节点不正确: %v", remaining)
	}
}

// stringContains 简单字符串包含检查
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
