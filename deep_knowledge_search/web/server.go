package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Server Web 服务器
type Server struct {
	port    int
	hub     *Hub
	started bool
	mu      sync.RWMutex
}

// TaskManager 任务管理器
type TaskManager struct {
	mu        sync.RWMutex
	executors map[string]interface{} // taskID -> executor interface（需要类型断言）
}

var globalServer *Server
var globalTaskManager = &TaskManager{
	executors: make(map[string]interface{}),
}

// NewServer 创建服务器
func NewServer(port int) *Server {
	return &Server{
		port: port,
		hub:  NewHub(),
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	s.mu.Unlock()

	// 启动 WebSocket hub
	go s.hub.Run()

	// 检查并生成缺失的排序索引
	go generateMissingOrderIndexes()

	// 静态文件服务
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 路由
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/history", s.handleHistoryList)
	http.HandleFunc("/api/history/", s.handleHistoryDetail)
	http.HandleFunc("/api/docs", s.handleDocsList)
	http.HandleFunc("/api/docs/", s.handleDocContent)

	// 任务管理 API
	http.HandleFunc("/api/task/pause/", s.handleTaskPause)
	http.HandleFunc("/api/task/resume/", s.handleTaskResume)
	http.HandleFunc("/api/task/recoverable", s.handleTaskRecoverable)
	http.HandleFunc("/api/task/recover/", s.handleTaskRecover)
	http.HandleFunc("/api/task/running", s.handleTaskRunning)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("[Web] Dashboard 启动: http://localhost%s\n", addr)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("[Web] 服务器错误: %v\n", err)
		}
	}()

	return nil
}

// handleIndex 主页
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/static/tree.html")
}

// handleWebSocket WebSocket 连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.hub.ServeWs(w, r)
}

// handleStatus 状态 API
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "running",
		"time":   time.Now().Format("2006-01-02 15:04:05"),
	})
}

// handleHistoryList 列出历史执行记录
func (s *Server) handleHistoryList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从 output 目录读取，每个任务目录下有 logs/execution.json
	outputDir := "output"
	entries, err := ioutil.ReadDir(outputDir)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "无法读取输出目录",
			"history": []interface{}{},
		})
		return
	}

	var history []map[string]interface{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 新结构：output/{task}/logs/execution.json
		execFile := filepath.Join(outputDir, entry.Name(), "logs", "execution.json")
		data, err := ioutil.ReadFile(execFile)
		if err != nil {
			continue
		}

		var execData map[string]interface{}
		if err := json.Unmarshal(data, &execData); err != nil {
			continue
		}

		// 只返回摘要信息
		history = append(history, map[string]interface{}{
			"id":         entry.Name(),
			"task_id":    execData["task_id"],
			"title":      execData["title"],
			"start_time": execData["start_time"],
			"end_time":   execData["end_time"],
			"success":    execData["success"],
		})
	}

	// 按时间倒序排列
	sort.Slice(history, func(i, j int) bool {
		ti, _ := history[i]["start_time"].(string)
		tj, _ := history[j]["start_time"].(string)
		return ti > tj
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
	})
}

// handleHistoryDetail 获取历史执行详情
func (s *Server) handleHistoryDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从路径获取ID
	id := strings.TrimPrefix(r.URL.Path, "/api/history/")
	if id == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "缺少历史记录ID",
		})
		return
	}

	// 新结构：output/{id}/logs/execution.json
	execFile := filepath.Join("output", id, "logs", "execution.json")
	data, err := ioutil.ReadFile(execFile)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "未找到历史记录",
		})
		return
	}

	var execData map[string]interface{}
	if err := json.Unmarshal(data, &execData); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "解析历史记录失败",
		})
		return
	}

	json.NewEncoder(w).Encode(execData)
}

// handleDocsList 列出输出文档
func (s *Server) handleDocsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	outputDir := "output"
	var docs []map[string]interface{}

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(outputDir, path)
		if relPath == "." {
			return nil
		}

		doc := map[string]interface{}{
			"path":   relPath,
			"name":   info.Name(),
			"is_dir": info.IsDir(),
		}

		if !info.IsDir() {
			doc["size"] = info.Size()
			doc["mod_time"] = info.ModTime().Format("2006-01-02 15:04:05")
		}

		docs = append(docs, doc)
		return nil
	})

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "无法读取输出目录",
			"docs":  []interface{}{},
		})
		return
	}

	// 读取所有任务的排序索引
	orderIndexes := make(map[string]interface{})
	entries, _ := ioutil.ReadDir(outputDir)
	for _, entry := range entries {
		if entry.IsDir() {
			orderPath := filepath.Join(outputDir, entry.Name(), "doc", ".order.json")
			data, err := ioutil.ReadFile(orderPath)
			if err == nil {
				var orderIndex interface{}
				if json.Unmarshal(data, &orderIndex) == nil {
					orderIndexes[entry.Name()] = orderIndex
				}
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"docs":         docs,
		"order_index":  orderIndexes,
	})
}

// handleDocContent 获取文档内容
func (s *Server) handleDocContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 从路径获取文件路径
	docPath := strings.TrimPrefix(r.URL.Path, "/api/docs/")
	if docPath == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "缺少文档路径",
		})
		return
	}

	// URL 解码
	decodedPath, err := url.PathUnescape(docPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "路径解码失败",
		})
		return
	}

	// 安全检查：防止路径遍历
	cleanPath := filepath.Clean(decodedPath)
	// 检查路径是否试图跳出 output 目录
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "非法路径",
		})
		return
	}

	fullPath := filepath.Join("output", cleanPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "文件不存在",
		})
		return
	}

	if info.IsDir() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "不能读取目录",
		})
		return
	}

	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "读取文件失败",
		})
		return
	}

	// 根据文件类型设置Content-Type
	if strings.HasSuffix(cleanPath, ".md") {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	} else if strings.HasSuffix(cleanPath, ".json") {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}

	w.Write(content)
}

// Broadcast 广播消息到所有客户端
func (s *Server) Broadcast(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	s.hub.Broadcast(data)
}

// ===========================================================================
// 全局函数
// ===========================================================================

// InitServer 初始化全局服务器
func InitServer(port int) {
	globalServer = NewServer(port)
}

// StartServer 启动全局服务器
func StartServer() error {
	if globalServer == nil {
		return fmt.Errorf("server not initialized")
	}
	return globalServer.Start()
}

// BroadcastEvent 广播事件
func BroadcastEvent(eventType string, data interface{}) {
	if globalServer == nil {
		return
	}
	msg := map[string]interface{}{
		"type": eventType,
		"data": data,
		"time": time.Now().Format("15:04:05"),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// 缓存 tree_update 类型的消息，用于新连接时发送
	if eventType == "tree_update" || eventType == "node_data" {
		globalServer.hub.SetLastState(msgBytes)
	}

	globalServer.hub.Broadcast(msgBytes)
}

// ===========================================================================
// 排序索引生成
// ===========================================================================

// OrderIndex 目录排序索引
type OrderIndex struct {
	Order    []string              `json:"order"`
	Children map[string]OrderIndex `json:"children"`
}

// generateMissingOrderIndexes 检查并生成缺失的排序索引
func generateMissingOrderIndexes() {
	outputDir := "output"
	entries, err := ioutil.ReadDir(outputDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskDir := filepath.Join(outputDir, entry.Name())
		orderPath := filepath.Join(taskDir, "doc", ".order.json")

		// 检查 .order.json 是否存在
		if _, err := os.Stat(orderPath); err == nil {
			continue // 已存在，跳过
		}

		// 尝试从 execution.json 生成
		execPath := filepath.Join(taskDir, "logs", "execution.json")
		data, err := ioutil.ReadFile(execPath)
		if err != nil {
			continue
		}

		var execData map[string]interface{}
		if err := json.Unmarshal(data, &execData); err != nil {
			continue
		}

		// 构建排序索引
		orderIndex := buildOrderIndexFromExecution(execData)

		// 确保 doc 目录存在
		docDir := filepath.Join(taskDir, "doc")
		if err := os.MkdirAll(docDir, 0755); err != nil {
			continue
		}

		// 保存排序索引
		orderData, err := json.MarshalIndent(orderIndex, "", "  ")
		if err != nil {
			continue
		}

		if err := ioutil.WriteFile(orderPath, orderData, 0644); err != nil {
			fmt.Printf("[Web] 生成排序索引失败 %s: %v\n", entry.Name(), err)
		} else {
			fmt.Printf("[Web] 已生成排序索引: %s\n", entry.Name())
		}
	}
}

// buildOrderIndexFromExecution 从 execution.json 构建排序索引
func buildOrderIndexFromExecution(data map[string]interface{}) OrderIndex {
	index := OrderIndex{
		Order:    []string{},
		Children: make(map[string]OrderIndex),
	}

	children, ok := data["children"].([]interface{})
	if !ok {
		return index
	}

	for _, child := range children {
		childMap, ok := child.(map[string]interface{})
		if !ok {
			continue
		}

		title, ok := childMap["title"].(string)
		if !ok {
			continue
		}

		// 清理文件名
		dirName := sanitizeForFilename(title)
		index.Order = append(index.Order, dirName)

		// 递归处理子节点
		if childChildren, ok := childMap["children"].([]interface{}); ok && len(childChildren) > 0 {
			index.Children[dirName] = buildOrderIndexFromExecution(childMap)
		}
	}

	return index
}

// sanitizeForFilename 清理文件名中的非法字符
func sanitizeForFilename(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// 限制长度
	runes := []rune(result)
	if len(runes) > 50 {
		result = string(runes[:50])
	}
	return result
}

// ===========================================================================
// 静态资源已移至 static/ 目录
// ===========================================================================
