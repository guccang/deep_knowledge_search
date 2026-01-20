package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
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

var globalServer *Server

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

	// 路由
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/api/status", s.handleStatus)

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
	tmpl := template.Must(template.New("index").Parse(indexHTML))
	tmpl.Execute(w, nil)
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
	globalServer.Broadcast(map[string]interface{}{
		"type": eventType,
		"data": data,
		"time": time.Now().Format("15:04:05"),
	})
}

// ===========================================================================
// 内嵌 HTML 模板
// ===========================================================================

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Deep Knowledge Search - Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #e8e8e8;
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        header {
            text-align: center;
            padding: 30px 0;
            border-bottom: 1px solid #2a2a4a;
        }
        h1 {
            color: #4ade80;
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        .status-badge {
            display: inline-block;
            padding: 8px 20px;
            border-radius: 20px;
            background: #22c55e;
            color: white;
            font-weight: bold;
        }
        .status-badge.disconnected {
            background: #ef4444;
        }
        .task-panel {
            margin-top: 30px;
            background: rgba(255,255,255,0.05);
            border-radius: 12px;
            padding: 20px;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .panel-title {
            font-size: 1.2em;
            color: #60a5fa;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .task-tree {
            font-family: 'Consolas', monospace;
            background: rgba(0,0,0,0.3);
            padding: 15px;
            border-radius: 8px;
            white-space: pre-wrap;
            min-height: 100px;
        }
        .log-panel {
            margin-top: 20px;
            max-height: 400px;
            overflow-y: auto;
        }
        .log-entry {
            padding: 8px 12px;
            border-left: 3px solid #4ade80;
            margin-bottom: 5px;
            background: rgba(0,0,0,0.2);
            border-radius: 0 4px 4px 0;
            font-size: 0.9em;
        }
        .log-entry.warn { border-left-color: #fbbf24; }
        .log-entry.error { border-left-color: #ef4444; }
        .log-time { color: #888; margin-right: 10px; }
        .empty-state {
            text-align: center;
            color: #666;
            padding: 40px;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🔍 Deep Knowledge Search</h1>
            <span id="status" class="status-badge disconnected">连接中...</span>
        </header>
        
        <div class="task-panel">
            <div class="panel-title">📊 任务结构</div>
            <div id="taskTree" class="task-tree">
                <div class="empty-state">等待任务开始...</div>
            </div>
        </div>
        
        <div class="task-panel">
            <div class="panel-title">📋 执行日志</div>
            <div id="logs" class="log-panel">
                <div class="empty-state">暂无日志</div>
            </div>
        </div>
    </div>
    
    <script>
        let ws;
        const statusEl = document.getElementById('status');
        const taskTreeEl = document.getElementById('taskTree');
        const logsEl = document.getElementById('logs');
        let logCount = 0;
        
        function connect() {
            ws = new WebSocket('ws://' + location.host + '/ws');
            
            ws.onopen = () => {
                statusEl.textContent = '已连接';
                statusEl.classList.remove('disconnected');
            };
            
            ws.onclose = () => {
                statusEl.textContent = '已断开';
                statusEl.classList.add('disconnected');
                setTimeout(connect, 2000);
            };
            
            ws.onmessage = (e) => {
                const msg = JSON.parse(e.data);
                handleMessage(msg);
            };
        }
        
        function handleMessage(msg) {
            switch(msg.type) {
                case 'task_start':
                    taskTreeEl.innerHTML = '🚀 ' + msg.data.title;
                    clearLogs();
                    addLog('info', '任务开始: ' + msg.data.title, msg.time);
                    break;
                case 'task_complete':
                    addLog('info', '✅ 任务完成: ' + msg.data.title, msg.time);
                    break;
                case 'task_failed':
                    addLog('error', '❌ 任务失败: ' + msg.data.error, msg.time);
                    break;
                case 'node_start':
                    updateTree(msg.data);
                    addLog('info', '▶ 开始: ' + msg.data.title, msg.time);
                    break;
                case 'node_complete':
                    updateTree(msg.data);
                    addLog('info', '✓ 完成: ' + msg.data.title, msg.time);
                    break;
                case 'node_failed':
                    addLog('error', '✗ 失败: ' + msg.data.title, msg.time);
                    break;
                case 'subtasks':
                    addLog('info', '📋 分解为 ' + msg.data.count + ' 个子任务', msg.time);
                    break;
                case 'log':
                    addLog(msg.data.level, msg.data.message, msg.time);
                    break;
            }
        }
        
        function updateTree(node) {
            let html = buildTree(node, 0);
            taskTreeEl.innerHTML = html || '<div class="empty-state">等待任务开始...</div>';
        }
        
        function buildTree(node, depth) {
            let indent = '  '.repeat(depth);
            let icon = node.status === 'done' ? '✅' : 
                       node.status === 'running' ? '🔄' : 
                       node.status === 'failed' ? '❌' : '⏳';
            let html = indent + icon + ' ' + node.title + '\n';
            if (node.children) {
                for (let child of node.children) {
                    html += buildTree(child, depth + 1);
                }
            }
            return html;
        }
        
        function addLog(level, message, time) {
            if (logCount === 0) {
                logsEl.innerHTML = '';
            }
            const entry = document.createElement('div');
            entry.className = 'log-entry ' + level;
            entry.innerHTML = '<span class="log-time">' + time + '</span>' + message;
            logsEl.insertBefore(entry, logsEl.firstChild);
            logCount++;
            if (logCount > 100) {
                logsEl.removeChild(logsEl.lastChild);
            }
        }
        
        function clearLogs() {
            logsEl.innerHTML = '';
            logCount = 0;
        }
        
        connect();
    </script>
</body>
</html>`
