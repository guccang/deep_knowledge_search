package agent

import (
	"deepknowledgesearch/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RecoveryManager 任务恢复管理器
type RecoveryManager struct {
	outputDir string
}

// NewRecoveryManager 创建恢复管理器
func NewRecoveryManager() *RecoveryManager {
	return &RecoveryManager{
		outputDir: config.GetOutputDir(),
	}
}

// FindRecoverableTasks 查找可恢复的任务
func (rm *RecoveryManager) FindRecoverableTasks() ([]RecoverableTask, error) {
	var tasks []RecoverableTask

	// 读取输出目录
	entries, err := os.ReadDir(rm.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return tasks, nil
		}
		return nil, fmt.Errorf("读取输出目录失败: %w", err)
	}

	// 遍历每个任务目录
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskDir := filepath.Join(rm.outputDir, entry.Name())
		checkpointPath := filepath.Join(taskDir, LogSubDir, "checkpoint.json")

		// 检查是否存在检查点文件
		if _, err := os.Stat(checkpointPath); err == nil {
			// 加载检查点查看状态
			node, err := LoadCheckpoint(checkpointPath)
			if err != nil {
				fmt.Printf("[Recovery] 加载检查点失败 %s: %v\n", entry.Name(), err)
				continue
			}

			// 只恢复未完成的任务
			if node.Status == NodeRunning || node.Status == NodePaused {
				tasks = append(tasks, RecoverableTask{
					TaskID:         node.ID,
					Title:          node.Title,
					Status:         node.Status,
					CheckpointPath: checkpointPath,
					TaskFolder:     entry.Name(),
				})
			}
		}
	}

	return tasks, nil
}

// RecoverTask 恢复任务
func (rm *RecoveryManager) RecoverTask(taskFolder string) (*TaskNode, *TaskExecutor, error) {
	checkpointPath := filepath.Join(rm.outputDir, taskFolder, LogSubDir, "checkpoint.json")

	// 加载检查点
	node, err := LoadCheckpoint(checkpointPath)
	if err != nil {
		return nil, nil, fmt.Errorf("加载检查点失败: %w", err)
	}

	// 重新初始化节点的内部状态
	rm.reinitializeNode(node)

	// 创建执行器
	planner := NewTaskPlanner()
	executor := NewTaskExecutor(node, planner, DefaultExecutionConfig())

	// 设置恢复模式，使用原有的任务文件夹
	executor.SetRecoveryMode(taskFolder)

	return node, executor, nil
}

// reinitializeNode 重新初始化节点的内部状态（channel等）
func (rm *RecoveryManager) reinitializeNode(node *TaskNode) {
	// 重新创建 cancelCh
	node.cancelCh = make(chan struct{}, 1)

	// 处理节点状态：
	// - done/failed/canceled 状态保持不变
	// - running 状态：如果有结果设为 done，否则重置为 pending
	// - paused 状态：重置为 pending 继续执行
	if node.Status == NodeRunning {
		if node.Result != nil && node.Result.Success {
			node.Status = NodeDone
		} else {
			node.Status = NodePending
		}
	} else if node.Status == NodePaused {
		node.Status = NodePending
	}

	// 递归处理子节点
	for _, child := range node.Children {
		rm.reinitializeNode(child)
	}
}

// CleanupCheckpoint 清理检查点（任务完成后）
func (rm *RecoveryManager) CleanupCheckpoint(taskFolder string) error {
	checkpointPath := filepath.Join(rm.outputDir, taskFolder, LogSubDir, "checkpoint.json")
	if err := os.Remove(checkpointPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除检查点失败: %w", err)
	}
	return nil
}

// RecoverableTask 可恢复的任务信息
type RecoverableTask struct {
	TaskID         string     `json:"task_id"`
	Title          string     `json:"title"`
	Status         NodeStatus `json:"status"`
	CheckpointPath string     `json:"checkpoint_path"`
	TaskFolder     string     `json:"task_folder"`
}

// ListRecoverableTasks 列出所有可恢复的任务（用于 Web API）
func ListRecoverableTasks() ([]RecoverableTask, error) {
	rm := NewRecoveryManager()
	return rm.FindRecoverableTasks()
}

// RecoverTaskByFolder 根据任务文件夹恢复任务
func RecoverTaskByFolder(taskFolder string) (*TaskNode, *TaskExecutor, error) {
	rm := NewRecoveryManager()
	return rm.RecoverTask(taskFolder)
}

// GetTaskFolderFromTitle 从标题生成任务文件夹名（用于查找）
func GetTaskFolderFromTitle(title string) string {
	sanitized := sanitizeForFilename(title)
	// 注意：这只返回标题部分，实际的文件夹还包含时间戳
	// 可能需要模糊匹配
	return sanitized
}

// FindTaskFolderByPrefix 通过前缀查找任务文件夹
func FindTaskFolderByPrefix(prefix string) (string, error) {
	outputDir := config.GetOutputDir()
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return "", fmt.Errorf("读取输出目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			return entry.Name(), nil
		}
	}

	return "", fmt.Errorf("未找到匹配的任务文件夹: %s", prefix)
}
