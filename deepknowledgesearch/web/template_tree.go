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
            max-height: 180px;
            overflow-y: auto;
            color: #a6adc8;
        }
        .code-block.response { border-left: 2px solid #4ade80; }
        .code-block.request { border-left: 2px solid #89b4fa; }
        .sub-label { color: #888; font-size: 0.75em; margin: 6px 0 3px; }
        .empty-state { text-align: center; color: #666; padding: 30px; }
    </style>
</head>
<body>
    <div class="app-container">
        <div class="main-content" id="mainContent">
            <header>
                <h1>🔍 Deep Knowledge Search</h1>
                <span id="status" class="status-badge disconnected">连接中...</span>
            </header>
            
            <div class="tree-container" id="treeContainer">
                <div class="empty-state">等待任务开始...</div>
            </div>
            
            <div class="log-panel" id="logs">
                <div class="empty-state" style="padding:15px">暂无日志</div>
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
        
        connect();
    </script>
</body>
</html>`
