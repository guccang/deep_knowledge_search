package web

// 此文件包含任务管理相关的 API 处理函数

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// TaskExecutorInterface 任务执行器接口（避免导入循环）
type TaskExecutorInterface interface {
	Pause()
	Resume()
	IsPaused() bool
}

// RecoverableTaskInfo 可恢复的任务信息（避免导入 agent 包）
type RecoverableTaskInfo struct {
	TaskID         string `json:"task_id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	CheckpointPath string `json:"checkpoint_path"`
	TaskFolder     string `json:"task_folder"`
}

// 回调函数类型
type ListRecoverableTasksFunc func() ([]RecoverableTaskInfo, error)
type RecoverTaskFunc func(taskFolder string) error

// 回调函数变量
var (
	listRecoverableTasksCallback ListRecoverableTasksFunc
	recoverTaskCallback          RecoverTaskFunc
)

// SetListRecoverableTasksCallback 设置列出可恢复任务的回调函数
func SetListRecoverableTasksCallback(fn ListRecoverableTasksFunc) {
	listRecoverableTasksCallback = fn
}

// SetRecoverTaskCallback 设置恢复任务的回调函数
func SetRecoverTaskCallback(fn RecoverTaskFunc) {
	recoverTaskCallback = fn
}

// RegisterTaskExecutor 注册任务执行器
func RegisterTaskExecutor(taskID string, executor interface{}) {
	globalTaskManager.mu.Lock()
	defer globalTaskManager.mu.Unlock()
	globalTaskManager.executors[taskID] = executor
}

// UnregisterTaskExecutor 注销任务执行器
func UnregisterTaskExecutor(taskID string) {
	globalTaskManager.mu.Lock()
	defer globalTaskManager.mu.Unlock()
	delete(globalTaskManager.executors, taskID)
}

// GetTaskExecutor 获取任务执行器
func GetTaskExecutor(taskID string) interface{} {
	globalTaskManager.mu.RLock()
	defer globalTaskManager.mu.RUnlock()
	return globalTaskManager.executors[taskID]
}

// RunningTaskInfo 运行中任务信息
type RunningTaskInfo struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title,omitempty"`
}

// GetAllRunningTasks 获取所有运行中的任务
func GetAllRunningTasks() []RunningTaskInfo {
	globalTaskManager.mu.RLock()
	defer globalTaskManager.mu.RUnlock()

	tasks := make([]RunningTaskInfo, 0, len(globalTaskManager.executors))
	for taskID := range globalTaskManager.executors {
		tasks = append(tasks, RunningTaskInfo{
			TaskID: taskID,
			Title:  "", // 可以从executor中获取，但需要类型断言
		})
	}
	return tasks
}

// IsTaskRunning 检查任务是否正在运行
func IsTaskRunning(taskID string) bool {
	globalTaskManager.mu.RLock()
	defer globalTaskManager.mu.RUnlock()
	_, exists := globalTaskManager.executors[taskID]
	return exists
}

// GetRunningTaskIDs 获取所有正在运行的任务ID
func GetRunningTaskIDs() []string {
	globalTaskManager.mu.RLock()
	defer globalTaskManager.mu.RUnlock()

	ids := make([]string, 0, len(globalTaskManager.executors))
	for taskID := range globalTaskManager.executors {
		ids = append(ids, taskID)
	}
	return ids
}

// callMethod 使用反射调用方法（避免类型断言）
func callMethod(obj interface{}, methodName string) error {
	val := reflect.ValueOf(obj)
	method := val.MethodByName(methodName)
	if !method.IsValid() {
		return fmt.Errorf("方法 %s 不存在", methodName)
	}
	method.Call(nil)
	return nil
}

// handleTaskPause 处理暂停任务请求
func (s *Server) handleTaskPause(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从路径获取 taskID
	taskID := strings.TrimPrefix(r.URL.Path, "/api/task/pause/")
	if taskID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "缺少任务ID",
		})
		return
	}

	// 获取执行器
	executor := GetTaskExecutor(taskID)
	if executor == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "任务不存在或已完成",
		})
		return
	}

	// 调用暂停方法
	if err := callMethod(executor, "Pause"); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "任务已暂停",
	})
}

// handleTaskResume 处理继续任务请求
func (s *Server) handleTaskResume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从路径获取 taskID
	taskID := strings.TrimPrefix(r.URL.Path, "/api/task/resume/")
	if taskID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "缺少任务ID",
		})
		return
	}

	// 获取执行器
	executor := GetTaskExecutor(taskID)
	if executor == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "任务不存在或已完成",
		})
		return
	}

	// 调用继续方法
	if err := callMethod(executor, "Resume"); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "任务继续执行",
	})
}

// handleTaskRecoverable 列出可恢复的任务
func (s *Server) handleTaskRecoverable(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if listRecoverableTasksCallback == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "恢复功能未初始化",
			"tasks":   []interface{}{},
		})
		return
	}

	tasks, err := listRecoverableTasksCallback()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"tasks":   []interface{}{},
		})
		return
	}

	// 过滤掉正在运行的任务
	runningIDs := GetRunningTaskIDs()
	filteredTasks := make([]RecoverableTaskInfo, 0)
	for _, task := range tasks {
		isRunning := false
		for _, runningID := range runningIDs {
			if task.TaskID == runningID {
				isRunning = true
				break
			}
		}
		if !isRunning {
			filteredTasks = append(filteredTasks, task)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"tasks":   filteredTasks,
	})
}

// handleTaskRecover 恢复任务
func (s *Server) handleTaskRecover(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从路径获取 taskFolder
	taskFolder := strings.TrimPrefix(r.URL.Path, "/api/task/recover/")
	if taskFolder == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "缺少任务文件夹名",
		})
		return
	}

	if recoverTaskCallback == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "恢复功能未初始化",
		})
		return
	}

	// 异步启动恢复任务
	go func() {
		if err := recoverTaskCallback(taskFolder); err != nil {
			fmt.Printf("[Web] 恢复任务失败: %v\n", err)
		}
	}()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "任务恢复已启动",
	})
}

// handleTaskRunning 返回所有运行中的任务
func (s *Server) handleTaskRunning(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	tasks := GetAllRunningTasks()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"tasks":   tasks,
	})
}
