package web

// indexHTMLTree 新的可折叠树形结构 HTML 模板
const indexHTMLTree = `<!DOCTYPE html>
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
        .app-container { display: flex; height: 100vh; }
        .main-content {
            flex: 1;
            display: flex;
            flex-direction: column;
            padding: 20px;
            overflow: hidden;
            transition: margin-right 0.3s ease;
        }
        .main-content.panel-open { margin-right: 480px; }
        header {
            text-align: center;
            padding: 15px 0;
            border-bottom: 1px solid #2a2a4a;
            flex-shrink: 0;
        }
        h1 { color: #4ade80; font-size: 1.6em; margin-bottom: 8px; }
        .status-badge {
            display: inline-block;
            padding: 5px 14px;
            border-radius: 15px;
            background: #22c55e;
            color: white;
            font-weight: bold;
            font-size: 0.8em;
        }
        .status-badge.disconnected { background: #ef4444; }
        
        /* Tree Container */
        .tree-container {
            flex: 1;
            overflow: auto;
            padding: 15px;
            background: rgba(0,0,0,0.2);
            border-radius: 10px;
            margin: 15px 0;
        }
        
        /* Tree Node */
        .tree-node {
            margin-left: 20px;
            border-left: 1px solid #3a3a5a;
            padding-left: 15px;
            position: relative;
        }
        .tree-node:before {
            content: '';
            position: absolute;
            left: 0;
            top: 18px;
            width: 15px;
            height: 1px;
            background: #3a3a5a;
        }
        .tree-node.root {
            margin-left: 0;
            border-left: none;
            padding-left: 0;
        }
        .tree-node.root:before { display: none; }
        
        .node-header {
            display: flex;
            align-items: center;
            padding: 8px 12px;
            margin: 4px 0;
            background: rgba(30,30,46,0.8);
            border-radius: 8px;
            cursor: pointer;
            border: 1px solid #3a3a5a;
            transition: all 0.2s;
        }
        .node-header:hover {
            background: rgba(50,50,70,0.9);
            border-color: #5a5a7a;
        }
        .node-header.selected {
            border-color: #89b4fa;
            background: rgba(137,180,250,0.15);
        }
        
        .toggle-btn {
            width: 20px;
            height: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-right: 8px;
            color: #888;
            font-size: 12px;
            flex-shrink: 0;
        }
        .toggle-btn.has-children { cursor: pointer; }
        .toggle-btn.has-children:hover { color: #fff; }
        
        .status-icon {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            margin-right: 10px;
            flex-shrink: 0;
        }
        .status-pending { background: #6b7280; }
        .status-running { background: #3b82f6; animation: pulse 1.5s infinite; }
        .status-done { background: #22c55e; }
        .status-failed { background: #ef4444; }
        .status-canceled { background: #f59e0b; }
        
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
        
        .node-title {
            flex: 1;
            font-size: 0.9em;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }
        
        .node-badge {
            font-size: 0.7em;
            padding: 2px 6px;
            border-radius: 4px;
            margin-left: 8px;
            background: rgba(137,180,250,0.2);
            color: #89b4fa;
        }
        
        .children-container { display: block; }
        .children-container.collapsed { display: none; }
        
        /* Log Panel */
        .log-panel {
            max-height: 120px;
            overflow-y: auto;
            background: rgba(0,0,0,0.2);
            border-radius: 8px;
            padding: 8px;
            flex-shrink: 0;
        }
        .log-entry {
            padding: 4px 8px;
            border-left: 2px solid #4ade80;
            margin-bottom: 3px;
            background: rgba(0,0,0,0.2);
            border-radius: 0 4px 4px 0;
            font-size: 0.8em;
        }
        .log-entry.warn { border-left-color: #fbbf24; }
        .log-entry.error { border-left-color: #ef4444; }
        .log-time { color: #666; margin-right: 6px; }
        
        /* Detail Panel */
        .detail-panel {
            position: fixed;
            right: -480px;
            top: 0;
            width: 480px;
            height: 100vh;
            background: linear-gradient(180deg, #1e1e2e 0%, #181825 100%);
            border-left: 1px solid #313244;
            transition: right 0.3s ease;
            z-index: 100;
            display: flex;
            flex-direction: column;
        }
        .detail-panel.open { right: 0; }
        .panel-header {
            padding: 12px 16px;
            border-bottom: 1px solid #313244;
            display: flex;
            justify-content: space-between;
            align-items: center;
            background: rgba(0,0,0,0.2);
        }
        .panel-header h3 { color: #cdd6f4; font-size: 1em; }
        .close-btn {
            background: none;
            border: none;
            color: #888;
            font-size: 1.3em;
            cursor: pointer;
            padding: 4px;
        }
        .close-btn:hover { color: #fff; }
        .panel-content {
            flex: 1;
            overflow-y: auto;
            padding: 12px;
        }
        .panel-section { margin-bottom: 16px; }
        .section-title {
            color: #89b4fa;
            font-size: 0.85em;
            font-weight: 600;
            margin-bottom: 8px;
        }
        .node-info {
            background: rgba(0,0,0,0.3);
            border-radius: 6px;
            padding: 10px;
        }
        .info-row {
            display: flex;
            margin-bottom: 6px;
            font-size: 0.85em;
        }
        .info-label { color: #888; width: 60px; flex-shrink: 0; }
        .info-value { color: #cdd6f4; word-break: break-all; }
        
        .llm-call {
            background: rgba(0,0,0,0.3);
            border-radius: 6px;
            margin-bottom: 8px;
            overflow: hidden;
        }
        .llm-call-header {
            padding: 8px 10px;
            background: rgba(137,180,250,0.1);
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 0.85em;
        }
        .llm-call-header:hover { background: rgba(137,180,250,0.2); }
        .llm-type { color: #89b4fa; font-weight: 600; }
        .llm-duration { color: #888; font-size: 0.8em; }
        .llm-call-body {
            display: none;
            padding: 10px;
            border-top: 1px solid #313244;
        }
        .llm-call-body.open { display: block; }
        .code-block {
            background: #11111b;
            border-radius: 4px;
            padding: 8px;
            font-family: 'Consolas', monospace;
            font-size: 0.75em;
            overflow-x: auto;
            white-space: pre-wrap;
            word-break: break-all;
            color: #a6adc8;
        }
        .code-block.response { border-left: 2px solid #4ade80; }
        .code-block.request { border-left: 2px solid #89b4fa; }
        .sub-label { color: #888; font-size: 0.75em; margin: 6px 0 3px; }
        .empty-state { text-align: center; color: #666; padding: 30px; }
        
        /* Toolbar */
        .toolbar {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 10px 0;
            flex-shrink: 0;
        }
        .toolbar-left, .toolbar-right {
            display: flex;
            gap: 8px;
            align-items: center;
        }
        .btn {
            padding: 6px 14px;
            border-radius: 6px;
            border: 1px solid #3a3a5a;
            background: rgba(30,30,46,0.8);
            color: #cdd6f4;
            cursor: pointer;
            font-size: 0.85em;
            transition: all 0.2s;
        }
        .btn:hover {
            background: rgba(50,50,70,0.9);
            border-color: #5a5a7a;
        }
        .btn:active {
            transform: scale(0.98);
        }
        .btn-icon {
            padding: 6px 10px;
        }
        
        /* Stats Bar */
        .stats-bar {
            display: flex;
            gap: 12px;
            align-items: center;
        }
        .stat-item {
            display: flex;
            align-items: center;
            gap: 5px;
            font-size: 0.85em;
            padding: 4px 10px;
            border-radius: 12px;
            background: rgba(0,0,0,0.3);
        }
        .stat-icon {
            width: 8px;
            height: 8px;
            border-radius: 50%;
        }
        .stat-done { background: #22c55e; }
        .stat-running { background: #3b82f6; }
        .stat-pending { background: #6b7280; }
        .stat-failed { background: #ef4444; }
        .stat-value { color: #fff; font-weight: 600; }
        .stat-label { color: #888; }
        
        /* Update animation */
        @keyframes nodeUpdate {
            0% { background-color: rgba(74, 222, 128, 0.3); }
            100% { background-color: transparent; }
        }
        .node-updated {
            animation: nodeUpdate 1s ease-out;
        }
        
        /* Navigation Tabs */
        .nav-tabs {
            display: flex;
            gap: 4px;
            padding: 10px 0;
            border-bottom: 1px solid #3a3a5a;
            margin-bottom: 10px;
        }
        .nav-tab {
            padding: 8px 16px;
            border-radius: 6px 6px 0 0;
            background: rgba(30,30,46,0.5);
            border: 1px solid #3a3a5a;
            border-bottom: none;
            color: #888;
            cursor: pointer;
            font-size: 0.9em;
            transition: all 0.2s;
        }
        .nav-tab:hover {
            background: rgba(50,50,70,0.7);
            color: #ccc;
        }
        .nav-tab.active {
            background: rgba(137,180,250,0.15);
            border-color: #89b4fa;
            color: #89b4fa;
        }
        .tab-content { display: none; flex: 1; overflow: hidden; flex-direction: column; }
        .tab-content.active { display: flex; }
        
        /* History List */
        .history-list {
            flex: 1;
            overflow: auto;
            padding: 10px;
        }
        .history-item {
            padding: 12px 15px;
            background: rgba(30,30,46,0.8);
            border-radius: 8px;
            margin-bottom: 8px;
            cursor: pointer;
            border: 1px solid #3a3a5a;
            transition: all 0.2s;
        }
        .history-item:hover {
            background: rgba(50,50,70,0.9);
            border-color: #5a5a7a;
        }
        .history-item.selected {
            border-color: #89b4fa;
            background: rgba(137,180,250,0.15);
        }
        .history-title {
            font-size: 1em;
            margin-bottom: 6px;
            color: #cdd6f4;
        }
        .history-meta {
            font-size: 0.8em;
            color: #888;
            display: flex;
            gap: 15px;
        }
        .history-status {
            display: inline-block;
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-right: 6px;
        }
        .history-status.success { background: #22c55e; }
        .history-status.failed { background: #ef4444; }
        
        /* Document Browser */
        .doc-browser {
            display: flex;
            flex: 1;
            overflow: hidden;
            gap: 15px;
        }
        .doc-tree {
            width: 280px;
            overflow: auto;
            background: rgba(0,0,0,0.2);
            border-radius: 8px;
            padding: 10px;
            flex-shrink: 0;
        }
        .doc-viewer {
            flex: 1;
            overflow: auto;
            background: rgba(0,0,0,0.2);
            border-radius: 8px;
            padding: 20px;
        }
        .doc-item {
            padding: 6px 10px;
            cursor: pointer;
            border-radius: 4px;
            margin-bottom: 2px;
            font-size: 0.85em;
            transition: background 0.2s;
        }
        .doc-item:hover { background: rgba(137,180,250,0.1); }
        .doc-item.selected { background: rgba(137,180,250,0.2); color: #89b4fa; }
        .doc-item.folder { color: #fbbf24; }
        .doc-item.file { color: #cdd6f4; padding-left: 20px; }
        .doc-folder-items { margin-left: 15px; display: none; }
        .doc-folder-items.open { display: block; }
        
        /* Markdown Styles */
        .markdown-content {
            line-height: 1.7;
            color: #cdd6f4;
        }
        .markdown-content h1, .markdown-content h2, .markdown-content h3 {
            color: #89b4fa;
            margin: 20px 0 10px;
        }
        .markdown-content h1 { font-size: 1.6em; border-bottom: 1px solid #3a3a5a; padding-bottom: 10px; }
        .markdown-content h2 { font-size: 1.3em; }
        .markdown-content h3 { font-size: 1.1em; }
        .markdown-content p { margin: 10px 0; }
        .markdown-content ul, .markdown-content ol { margin: 10px 0; padding-left: 25px; }
        .markdown-content li { margin: 5px 0; }
        .markdown-content code {
            background: #11111b;
            padding: 2px 6px;
            border-radius: 4px;
            font-family: 'Consolas', monospace;
            font-size: 0.9em;
        }
        .markdown-content pre {
            background: #11111b;
            padding: 15px;
            border-radius: 8px;
            overflow-x: auto;
            margin: 15px 0;
        }
        .markdown-content pre code {
            background: none;
            padding: 0;
        }
        .markdown-content table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0;
        }
        .markdown-content th, .markdown-content td {
            border: 1px solid #3a3a5a;
            padding: 8px 12px;
            text-align: left;
        }
        .markdown-content th { background: rgba(137,180,250,0.1); }
        .markdown-content blockquote {
            border-left: 3px solid #89b4fa;
            padding-left: 15px;
            margin: 15px 0;
            color: #a6adc8;
        }
    </style>
</head>
<body>
    <div class="app-container">
        <div class="main-content" id="mainContent">
            <header>
                <h1>🔍 Deep Knowledge Search</h1>
                <span id="status" class="status-badge disconnected">连接中...</span>
            </header>
            
            <div class="nav-tabs">
                <div class="nav-tab active" onclick="switchTab('current')">📊 当前任务</div>
                <div class="nav-tab" onclick="switchTab('history')">📜 历史记录</div>
                <div class="nav-tab" onclick="switchTab('docs')">📄 输出文档</div>
            </div>
            
            <!-- 当前任务标签 -->
            <div class="tab-content active" id="tab-current">
                <div class="toolbar">
                    <div class="toolbar-left">
                        <button class="btn btn-icon" onclick="expandAll()" title="展开全部">📂 展开</button>
                        <button class="btn btn-icon" onclick="collapseAll()" title="折叠全部">📁 折叠</button>
                    </div>
                    <div class="stats-bar" id="statsBar">
                        <div class="stat-item">
                            <span class="stat-icon stat-done"></span>
                            <span class="stat-value" id="statDone">0</span>
                            <span class="stat-label">完成</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-icon stat-running"></span>
                            <span class="stat-value" id="statRunning">0</span>
                            <span class="stat-label">执行中</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-icon stat-pending"></span>
                            <span class="stat-value" id="statPending">0</span>
                            <span class="stat-label">待处理</span>
                        </div>
                        <div class="stat-item">
                            <span class="stat-icon stat-failed"></span>
                            <span class="stat-value" id="statFailed">0</span>
                            <span class="stat-label">失败</span>
                        </div>
                    </div>
                </div>
                
                <div class="tree-container" id="treeContainer">
                    <div class="empty-state">等待任务开始...</div>
                </div>
                
                <div class="log-panel" id="logs">
                    <div class="empty-state" style="padding:15px">暂无日志</div>
                </div>
            </div>
            
            <!-- 历史记录标签 -->
            <div class="tab-content" id="tab-history">
                <div class="toolbar">
                    <div class="toolbar-left">
                        <button class="btn" onclick="loadHistory()">🔄 刷新</button>
                    </div>
                </div>
                <div class="history-list" id="historyList">
                    <div class="empty-state">加载中...</div>
                </div>
            </div>
            
            <!-- 输出文档标签 -->
            <div class="tab-content" id="tab-docs">
                <div class="toolbar">
                    <div class="toolbar-left">
                        <button class="btn" onclick="loadDocs()">🔄 刷新</button>
                    </div>
                </div>
                <div class="doc-browser">
                    <div class="doc-tree" id="docTree">
                        <div class="empty-state">加载中...</div>
                    </div>
                    <div class="doc-viewer" id="docViewer">
                        <div class="empty-state">选择文档查看</div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="detail-panel" id="detailPanel">
            <div class="panel-header">
                <h3 id="panelTitle">节点详情</h3>
                <button class="close-btn" onclick="closePanel()">×</button>
            </div>
            <div class="panel-content" id="panelContent"></div>
        </div>
    </div>
    
    <script>
        let ws;
        const statusEl = document.getElementById('status');
        const logsEl = document.getElementById('logs');
        const treeContainer = document.getElementById('treeContainer');
        const mainContent = document.getElementById('mainContent');
        const detailPanel = document.getElementById('detailPanel');
        const panelContent = document.getElementById('panelContent');
        const panelTitle = document.getElementById('panelTitle');
        
        let logCount = 0;
        let taskData = null;
        let selectedNodeId = null;
        let collapsedNodes = new Set();
        
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
                    taskData = { title: msg.data.title, status: 'running', children: [] };
                    clearLogs();
                    addLog('info', '任务开始: ' + msg.data.title, msg.time);
                    renderTree();
                    break;
                case 'task_complete':
                    if (taskData) taskData.status = 'done';
                    addLog('info', '✅ 任务完成', msg.time);
                    renderTree();
                    break;
                case 'task_failed':
                    if (taskData) taskData.status = 'failed';
                    addLog('error', '❌ 任务失败: ' + msg.data.error, msg.time);
                    renderTree();
                    break;
                case 'node_start':
                case 'node_complete':
                case 'node_failed':
                    updateTaskData(msg.data);
                    addLog(msg.type === 'node_failed' ? 'error' : 'info', 
                           (msg.type === 'node_start' ? '▶ ' : msg.type === 'node_complete' ? '✓ ' : '✗ ') + msg.data.title, 
                           msg.time);
                    renderTree();
                    break;
                case 'tree_update':
                case 'node_data':
                    if (msg.type === 'tree_update') {
                        taskData = msg.data;
                    } else {
                        updateTaskData(msg.data);
                    }
                    renderTree();
                    break;
                case 'log':
                    addLog(msg.data.level, msg.data.message, msg.time);
                    break;
            }
        }
        
        function updateTaskData(nodeData) {
            if (!taskData) {
                taskData = nodeData;
            } else if (!taskData.id && nodeData.depth === 0) {
                taskData = nodeData;
            } else {
                mergeNodeData(taskData, nodeData);
            }
        }
        
        function mergeNodeData(target, source) {
            if (target.id === source.id) {
                const existingChildren = target.children || [];
                Object.assign(target, source);
                if (!source.children && existingChildren.length > 0) {
                    target.children = existingChildren;
                }
                return true;
            }
            if (source.parent_id === target.id) {
                if (!target.children) target.children = [];
                const existing = target.children.find(c => c.id === source.id);
                if (existing) {
                    const existingChildren = existing.children || [];
                    Object.assign(existing, source);
                    if (!source.children && existingChildren.length > 0) {
                        existing.children = existingChildren;
                    }
                } else {
                    target.children.push(source);
                }
                return true;
            }
            if (target.children) {
                for (let child of target.children) {
                    if (mergeNodeData(child, source)) return true;
                }
            }
            return false;
        }
        
        function renderTree() {
            if (!taskData) {
                treeContainer.innerHTML = '<div class="empty-state">等待任务开始...</div>';
                return;
            }
            treeContainer.innerHTML = renderNode(taskData, true);
            updateStats();
        }
        
        function renderNode(node, isRoot) {
            const hasChildren = node.children && node.children.length > 0;
            const isCollapsed = collapsedNodes.has(node.id);
            const isSelected = selectedNodeId === node.id;
            const status = node.status || 'pending';
            
            let html = '<div class="tree-node' + (isRoot ? ' root' : '') + '">';
            html += '<div class="node-header' + (isSelected ? ' selected' : '') + '" onclick="selectNode(\'' + node.id + '\')">';
            
            if (hasChildren) {
                html += '<span class="toggle-btn has-children" onclick="event.stopPropagation();toggleNode(\'' + node.id + '\')">' + (isCollapsed ? '▶' : '▼') + '</span>';
            } else {
                html += '<span class="toggle-btn">•</span>';
            }
            
            html += '<span class="status-icon status-' + status + '"></span>';
            html += '<span class="node-title">' + escapeHtml(node.title || 'Task') + '</span>';
            
            if (node.llm_calls && node.llm_calls.length > 0) {
                html += '<span class="node-badge">LLM: ' + node.llm_calls.length + '</span>';
            }
            
            // 显示执行模式徽章（仅对有子节点的节点显示）
            if (hasChildren && node.execution_mode) {
                if (node.execution_mode === 'parallel') {
                    html += '<span class="node-badge" style="background:rgba(59,130,246,0.2);color:#3b82f6">🔀 并行</span>';
                } else {
                    html += '<span class="node-badge" style="background:rgba(156,163,175,0.2);color:#9ca3af">➡️ 串行</span>';
                }
            }
            
            // 显示验证徽章
            if (node.verification) {
                if (node.verification.passed) {
                    html += '<span class="node-badge" style="background:rgba(34,197,94,0.2);color:#22c55e">✓ 验证通过</span>';
                } else if (node.verification.iterations > 0) {
                    html += '<span class="node-badge" style="background:rgba(251,191,36,0.2);color:#fbbf24">验证中(' + node.verification.iterations + ')</span>';
                }
            }
            
            html += '</div>';
            
            if (hasChildren) {
                html += '<div class="children-container' + (isCollapsed ? ' collapsed' : '') + '">';
                for (const child of node.children) {
                    html += renderNode(child, false);
                }
                html += '</div>';
            }
            
            html += '</div>';
            return html;
        }
        
        function toggleNode(nodeId) {
            if (collapsedNodes.has(nodeId)) {
                collapsedNodes.delete(nodeId);
            } else {
                collapsedNodes.add(nodeId);
            }
            renderTree();
        }
        
        // 收集所有节点ID
        function collectAllNodeIds(node, ids) {
            if (!node) return;
            if (node.id) ids.push(node.id);
            if (node.children) {
                for (const child of node.children) {
                    collectAllNodeIds(child, ids);
                }
            }
        }
        
        // 展开全部
        function expandAll() {
            collapsedNodes.clear();
            renderTree();
        }
        
        // 折叠全部
        function collapseAll() {
            if (!taskData) return;
            const allIds = [];
            collectAllNodeIds(taskData, allIds);
            collapsedNodes = new Set(allIds);
            renderTree();
        }
        
        // 统计节点数量
        function countNodes(node) {
            const stats = { done: 0, running: 0, pending: 0, failed: 0, canceled: 0, total: 0 };
            if (!node) return stats;
            
            function count(n) {
                if (!n) return;
                stats.total++;
                const status = n.status || 'pending';
                if (status === 'done') stats.done++;
                else if (status === 'running') stats.running++;
                else if (status === 'failed') stats.failed++;
                else if (status === 'canceled') stats.canceled++;
                else stats.pending++;
                
                if (n.children) {
                    for (const child of n.children) {
                        count(child);
                    }
                }
            }
            count(node);
            return stats;
        }
        
        // 更新统计显示
        function updateStats() {
            const stats = countNodes(taskData);
            document.getElementById('statDone').textContent = stats.done;
            document.getElementById('statRunning').textContent = stats.running;
            document.getElementById('statPending').textContent = stats.pending;
            document.getElementById('statFailed').textContent = stats.failed + stats.canceled;
        }
        
        function findNode(node, id) {
            if (node.id === id) return node;
            if (node.children) {
                for (const child of node.children) {
                    const found = findNode(child, id);
                    if (found) return found;
                }
            }
            return null;
        }
        
        function selectNode(nodeId) {
            selectedNodeId = nodeId;
            renderTree();
            const node = findNode(taskData, nodeId);
            if (node) showNodeDetail(node);
        }
        
        function showNodeDetail(node) {
            panelTitle.textContent = node.title || 'Task Node';
            
            let html = '';
            html += '<div class="panel-section">';
            html += '<div class="section-title">📋 基本信息</div>';
            html += '<div class="node-info">';
            html += '<div class="info-row"><span class="info-label">ID:</span><span class="info-value">' + node.id + '</span></div>';
            html += '<div class="info-row"><span class="info-label">状态:</span><span class="info-value"><span class="status-icon status-' + (node.status || 'pending') + '" style="display:inline-block;vertical-align:middle"></span> ' + (node.status || 'pending') + '</span></div>';
            if (node.description) {
                html += '<div class="info-row"><span class="info-label">描述:</span><span class="info-value">' + escapeHtml(node.description) + '</span></div>';
            }
            if (node.goal) {
                html += '<div class="info-row"><span class="info-label">目标:</span><span class="info-value">' + escapeHtml(node.goal) + '</span></div>';
            }
            html += '</div></div>';
            
            if (node.llm_calls && node.llm_calls.length > 0) {
                html += '<div class="panel-section">';
                html += '<div class="section-title">🤖 LLM 调用记录 (' + node.llm_calls.length + ')</div>';
                
                const typeLabels = { plan: '规划', execute: '执行', synthesize: '整合', verify: '验证' };
                node.llm_calls.forEach((call, idx) => {
                    html += '<div class="llm-call">';
                    html += '<div class="llm-call-header" onclick="toggleLLMCall(' + idx + ')">';
                    html += '<span class="llm-type">' + (typeLabels[call.type] || call.type) + '</span>';
                    html += '<span class="llm-duration">' + call.duration_ms + 'ms</span>';
                    html += '</div>';
                    html += '<div class="llm-call-body" id="llm-call-' + idx + '">';
                    html += '<div class="sub-label">请求:</div>';
                    html += '<div class="code-block request">' + escapeHtml(JSON.stringify(call.messages, null, 2)) + '</div>';
                    html += '<div class="sub-label">响应:</div>';
                    html += '<div class="code-block response">' + escapeHtml(call.response) + '</div>';
                    html += '</div></div>';
                });
                html += '</div>';
            }
            
            // 验证信息
            if (node.verification) {
                html += '<div class="panel-section">';
                html += '<div class="section-title">🔍 验证结果</div>';
                html += '<div class="node-info">';
                html += '<div class="info-row"><span class="info-label">状态:</span><span class="info-value">' + (node.verification.passed ? '<span style="color:#22c55e">✓ 通过</span>' : '<span style="color:#fbbf24">未通过</span>') + '</span></div>';
                html += '<div class="info-row"><span class="info-label">次数:</span><span class="info-value">' + node.verification.iterations + ' 次</span></div>';
                html += '</div>';
                
                if (node.verification.attempts && node.verification.attempts.length > 0) {
                    html += '<div style="margin-top:10px">';
                    node.verification.attempts.forEach((attempt, idx) => {
                        const bgColor = attempt.passed ? 'rgba(34,197,94,0.1)' : 'rgba(251,191,36,0.1)';
                        const borderColor = attempt.passed ? '#22c55e' : '#fbbf24';
                        html += '<div style="background:' + bgColor + ';border-left:2px solid ' + borderColor + ';padding:8px;margin-bottom:6px;border-radius:0 4px 4px 0">';
                        html += '<div style="font-size:0.8em;color:#888;margin-bottom:4px">第 ' + attempt.iteration + ' 次验证 (' + attempt.timestamp + ')</div>';
                        html += '<div style="font-size:0.85em">' + escapeHtml(attempt.feedback) + '</div>';
                        html += '</div>';
                    });
                    html += '</div>';
                }
                html += '</div>';
            }
            
            if (node.result) {
                html += '<div class="panel-section">';
                html += '<div class="section-title">📝 执行结果</div>';
                html += '<div class="code-block">' + escapeHtml(node.result.summary || node.result.output || JSON.stringify(node.result)) + '</div>';
                html += '</div>';
            }
            
            panelContent.innerHTML = html;
            detailPanel.classList.add('open');
            mainContent.classList.add('panel-open');
        }
        
        function toggleLLMCall(idx) {
            const body = document.getElementById('llm-call-' + idx);
            body.classList.toggle('open');
        }
        
        function closePanel() {
            detailPanel.classList.remove('open');
            mainContent.classList.remove('panel-open');
            selectedNodeId = null;
            renderTree();
        }
        
        function escapeHtml(text) {
            if (!text) return '';
            return text.toString().replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
        }
        
        function addLog(level, message, time) {
            if (logCount === 0) logsEl.innerHTML = '';
            const entry = document.createElement('div');
            entry.className = 'log-entry ' + level;
            entry.innerHTML = '<span class="log-time">' + time + '</span>' + escapeHtml(message);
            logsEl.insertBefore(entry, logsEl.firstChild);
            logCount++;
            if (logCount > 30) logsEl.removeChild(logsEl.lastChild);
        }
        
        function clearLogs() {
            logsEl.innerHTML = '';
            logCount = 0;
        }
        
        // =========================================
        // 标签页切换
        // =========================================
        function switchTab(tabName) {
            // 更新标签样式
            document.querySelectorAll('.nav-tab').forEach(tab => {
                tab.classList.remove('active');
            });
            event.target.classList.add('active');
            
            // 切换内容
            document.querySelectorAll('.tab-content').forEach(content => {
                content.classList.remove('active');
            });
            document.getElementById('tab-' + tabName).classList.add('active');
            
            // 加载数据
            if (tabName === 'history') loadHistory();
            if (tabName === 'docs') loadDocs();
        }
        
        // =========================================
        // 历史记录
        // =========================================
        let historyData = [];
        let selectedHistoryId = null;
        
        async function loadHistory() {
            try {
                const response = await fetch('/api/history');
                const data = await response.json();
                historyData = data.history || [];
                renderHistoryList();
            } catch (e) {
                document.getElementById('historyList').innerHTML = '<div class="empty-state">加载失败</div>';
            }
        }
        
        function renderHistoryList() {
            const container = document.getElementById('historyList');
            if (!historyData || historyData.length === 0) {
                container.innerHTML = '<div class="empty-state">暂无历史记录</div>';
                return;
            }
            
            let html = '';
            historyData.forEach(item => {
                const isSelected = selectedHistoryId === item.id;
                const statusClass = item.success ? 'success' : 'failed';
                const startTime = item.start_time ? new Date(item.start_time).toLocaleString('zh-CN') : '';
                
                html += '<div class="history-item' + (isSelected ? ' selected' : '') + '" onclick="selectHistory(\'' + item.id + '\')">';
                html += '<div class="history-title"><span class="history-status ' + statusClass + '"></span>' + escapeHtml(item.title || '未命名任务') + '</div>';
                html += '<div class="history-meta">';
                html += '<span>⏱️ ' + startTime + '</span>';
                html += '<span>' + (item.success ? '✅ 成功' : '❌ 失败') + '</span>';
                html += '</div></div>';
            });
            container.innerHTML = html;
        }
        
        async function selectHistory(id) {
            selectedHistoryId = id;
            renderHistoryList();
            
            try {
                const response = await fetch('/api/history/' + encodeURIComponent(id));
                const data = await response.json();
                
                // 使用历史数据渲染树
                if (data && !data.error) {
                    // 转换为树结构
                    const treeData = convertHistoryToTree(data);
                    taskData = treeData;
                    renderTree();
                    switchTab('current');
                    document.querySelectorAll('.nav-tab')[0].classList.add('active');
                }
            } catch (e) {
                console.error('加载历史详情失败', e);
            }
        }
        
        function convertHistoryToTree(data) {
            const node = {
                id: data.task_id,
                title: data.title,
                description: data.description,
                status: data.success ? 'done' : 'failed',
                result: data.result,
                children: []
            };
            
            if (data.children) {
                node.children = data.children.map(child => convertHistoryToTree(child));
            }
            
            return node;
        }
        
        // =========================================
        // 文档浏览
        // =========================================
        let docsData = [];
        let selectedDocPath = null;
        
        async function loadDocs() {
            try {
                const response = await fetch('/api/docs');
                const data = await response.json();
                docsData = data.docs || [];
                renderDocTree();
            } catch (e) {
                document.getElementById('docTree').innerHTML = '<div class="empty-state">加载失败</div>';
            }
        }
        
        function renderDocTree() {
            const container = document.getElementById('docTree');
            if (!docsData || docsData.length === 0) {
                container.innerHTML = '<div class="empty-state">暂无文档</div>';
                return;
            }
            
            // 构建树结构
            const tree = {};
            docsData.forEach(doc => {
                const parts = doc.path.split('/');
                let current = tree;
                parts.forEach((part, idx) => {
                    if (!current[part]) {
                        current[part] = { _info: doc, _children: {} };
                    }
                    current = current[part]._children;
                });
            });
            
            container.innerHTML = renderDocFolder(tree, '');
        }
        
        function renderDocFolder(folder, prefix) {
            let html = '';
            const entries = Object.entries(folder).sort((a, b) => {
                const aIsDir = Object.keys(a[1]._children).length > 0;
                const bIsDir = Object.keys(b[1]._children).length > 0;
                if (aIsDir !== bIsDir) return bIsDir - aIsDir;
                return a[0].localeCompare(b[0]);
            });
            
            entries.forEach(([name, data]) => {
                const path = prefix ? prefix + '/' + name : name;
                const hasChildren = Object.keys(data._children).length > 0;
                const isDir = data._info && data._info.is_dir;
                
                if (isDir || hasChildren) {
                    html += '<div class="doc-item folder" onclick="toggleDocFolder(this)">📁 ' + escapeHtml(name) + '</div>';
                    html += '<div class="doc-folder-items">';
                    html += renderDocFolder(data._children, path);
                    html += '</div>';
                } else {
                    const isSelected = selectedDocPath === path;
                    html += '<div class="doc-item file' + (isSelected ? ' selected' : '') + '" onclick="viewDoc(\'' + path.replace(/'/g, "\\'") + '\')">📄 ' + escapeHtml(name) + '</div>';
                }
            });
            return html;
        }
        
        function toggleDocFolder(el) {
            const folder = el.nextElementSibling;
            folder.classList.toggle('open');
            el.textContent = (folder.classList.contains('open') ? '📂 ' : '📁 ') + el.textContent.slice(3);
        }
        
        async function viewDoc(path) {
            // 更新选中状态（不重新渲染整个树）
            document.querySelectorAll('.doc-item.file').forEach(el => {
                el.classList.remove('selected');
            });
            // 找到对应的文件元素并添加选中状态
            document.querySelectorAll('.doc-item.file').forEach(el => {
                const elPath = el.getAttribute('onclick').match(/viewDoc\('(.+?)'\)/);
                if (elPath && elPath[1].replace(/\\'/g, "'") === path) {
                    el.classList.add('selected');
                }
            });
            selectedDocPath = path;
            
            const viewer = document.getElementById('docViewer');
            viewer.innerHTML = '<div class="empty-state">加载中...</div>';
            
            try {
                const response = await fetch('/api/docs/' + encodeURIComponent(path));
                const content = await response.text();
                
                if (response.ok) {
                    viewer.innerHTML = '<div class="markdown-content">' + renderMarkdown(content) + '</div>';
                } else {
                    viewer.innerHTML = '<div class="empty-state">加载失败</div>';
                }
            } catch (e) {
                viewer.innerHTML = '<div class="empty-state">加载失败</div>';
            }
        }
        
        // 简单的 Markdown 渲染
        function renderMarkdown(text) {
            let html = escapeHtml(text);
            const bt = String.fromCharCode(96); // backtick character
            
            // 代码块
            const codeBlockRe = new RegExp(bt + bt + bt + '(\\\\w*)\\n([\\\\s\\\\S]*?)' + bt + bt + bt, 'g');
            html = html.replace(codeBlockRe, '<pre><code>$2</code></pre>');
            
            // 行内代码
            const inlineCodeRe = new RegExp(bt + '([^' + bt + ']+)' + bt, 'g');
            html = html.replace(inlineCodeRe, '<code>$1</code>');
            
            // 标题
            html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');
            html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>');
            html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>');
            
            // 粗体和斜体
            html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
            html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
            
            // 列表
            html = html.replace(/^- (.+)$/gm, '<li>$1</li>');
            html = html.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');
            
            // 引用
            html = html.replace(/^&gt; (.+)$/gm, '<blockquote>$1</blockquote>');
            
            // 段落
            html = html.replace(/\n\n/g, '</p><p>');
            html = '<p>' + html + '</p>';
            
            // 清理
            html = html.replace(/<p><\/p>/g, '');
            html = html.replace(/<p>(<h[123]>)/g, '$1');
            html = html.replace(/(<\/h[123]>)<\/p>/g, '$1');
            html = html.replace(/<p>(<ul>)/g, '$1');
            html = html.replace(/(<\/ul>)<\/p>/g, '$1');
            html = html.replace(/<p>(<pre>)/g, '$1');
            html = html.replace(/(<\/pre>)<\/p>/g, '$1');
            html = html.replace(/<p>(<blockquote>)/g, '$1');
            html = html.replace(/(<\/blockquote>)<\/p>/g, '$1');
            
            return html;
        }
        
        connect();
    </script>
</body>
</html>`
