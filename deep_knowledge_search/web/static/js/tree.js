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
    switch (msg.type) {
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

    // 显示耗时
    if (node.started_at) {
        const start = new Date(node.started_at);
        const end = node.finished_at ? new Date(node.finished_at) : new Date();
        // 如果任务未完成且未运行（如暂停或失败），使用最后更新时间或保持当前时间
        // 这里简化处理：如果是 running，计算动态耗时；如果是 done/failed，计算固定耗时

        let duration = 0;
        if (node.finished_at) {
            duration = new Date(node.finished_at) - start;
        } else if (node.status === 'running') {
            duration = new Date() - start;
        }

        if (duration > 0) {
            let durationStr = '';
            if (duration < 1000) durationStr = duration + 'ms';
            else if (duration < 60000) durationStr = (duration / 1000).toFixed(1) + 's';
            else durationStr = (duration / 60000).toFixed(1) + 'm';

            if (!node.finished_at && node.status === 'running') {
                durationStr += '...';
            }

            // 只有当耗时有意义时才显示
            html += '<span class="node-badge" style="background:rgba(107,114,128,0.1);color:#6b7280" title="开始: ' + new Date(node.started_at).toLocaleTimeString() + '">⏱️ ' + durationStr + '</span>';
        }
    }

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

    // 时间信息
    if (node.created_at) {
        html += '<div class="info-row"><span class="info-label">创建时间:</span><span class="info-value">' + new Date(node.created_at).toLocaleString() + '</span></div>';
    }
    if (node.started_at) {
        html += '<div class="info-row"><span class="info-label">开始时间:</span><span class="info-value">' + new Date(node.started_at).toLocaleString() + '</span></div>';
    }
    if (node.finished_at) {
        html += '<div class="info-row"><span class="info-label">结束时间:</span><span class="info-value">' + new Date(node.finished_at).toLocaleString() + '</span></div>';

        // 计算总耗时
        if (node.started_at) {
            const duration = new Date(node.finished_at) - new Date(node.started_at);
            let durationStr = '';
            if (duration < 1000) durationStr = duration + 'ms';
            else if (duration < 60000) durationStr = (duration / 1000).toFixed(2) + 's';
            else durationStr = (duration / 60000).toFixed(2) + 'm';
            html += '<div class="info-row"><span class="info-label">总耗时:</span><span class="info-value">' + durationStr + '</span></div>';
        }
    }
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
let docsOrderIndex = {};  // 排序索引
let selectedDocPath = null;

async function loadDocs() {
    try {
        const response = await fetch('/api/docs');
        const data = await response.json();
        docsData = data.docs || [];
        docsOrderIndex = data.order_index || {};
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

    // 构建树结构（同时支持正斜杠和反斜杠作为分隔符）
    const tree = {};
    docsData.forEach(doc => {
        const parts = doc.path.split(/[\/\\]/);
        let current = tree;
        parts.forEach((part, idx) => {
            if (!part) return; // 跳过空字符串
            if (!current[part]) {
                current[part] = { _info: idx === parts.length - 1 ? doc : { is_dir: true }, _children: {} };
            }
            current = current[part]._children;
        });
    });

    container.innerHTML = renderDocFolder(tree, '', null);
}

// 获取排序顺序
function getOrderForPath(pathParts) {
    if (!pathParts || pathParts.length === 0) return null;

    // 第一级是任务文件夹
    const taskFolder = pathParts[0];
    if (!docsOrderIndex[taskFolder]) return null;

    // 如果是 doc 目录，获取其下的排序
    let orderData = docsOrderIndex[taskFolder];

    // 跳过任务文件夹，从 doc 开始查找
    for (let i = 1; i < pathParts.length; i++) {
        const part = pathParts[i];
        if (part === 'doc') continue;
        if (orderData.children && orderData.children[part]) {
            orderData = orderData.children[part];
        } else {
            break;
        }
    }

    return orderData ? orderData.order : null;
}

function renderDocFolder(folder, prefix, parentParts) {
    let html = '';
    let entries = Object.entries(folder);

    // 获取当前路径的排序顺序
    const pathParts = prefix ? prefix.split(/[\/\\]/) : [];
    const order = getOrderForPath(pathParts);

    // 如果有排序索引，按索引排序
    if (order && order.length > 0) {
        entries.sort((a, b) => {
            const aIsDir = Object.keys(a[1]._children).length > 0 || (a[1]._info && a[1]._info.is_dir);
            const bIsDir = Object.keys(b[1]._children).length > 0 || (b[1]._info && b[1]._info.is_dir);

            // 目录优先
            if (aIsDir !== bIsDir) return bIsDir - aIsDir;

            // 按排序索引排序
            const aIdx = order.indexOf(a[0]);
            const bIdx = order.indexOf(b[0]);

            // 都在索引中，按索引顺序
            if (aIdx !== -1 && bIdx !== -1) return aIdx - bIdx;
            // 只有 a 在索引中，a 排前面
            if (aIdx !== -1) return -1;
            // 只有 b 在索引中，b 排前面
            if (bIdx !== -1) return 1;
            // 都不在索引中，按字母顺序
            return a[0].localeCompare(b[0]);
        });
    } else {
        // 没有排序索引，按默认规则（目录优先，字母顺序）
        entries.sort((a, b) => {
            const aIsDir = Object.keys(a[1]._children).length > 0 || (a[1]._info && a[1]._info.is_dir);
            const bIsDir = Object.keys(b[1]._children).length > 0 || (b[1]._info && b[1]._info.is_dir);
            if (aIsDir !== bIsDir) return bIsDir - aIsDir;
            return a[0].localeCompare(b[0]);
        });
    }

    entries.forEach(([name, data]) => {
        // 跳过隐藏文件（如 .order.json）
        if (name.startsWith('.')) return;

        const path = prefix ? prefix + '/' + name : name;
        const hasChildren = Object.keys(data._children).length > 0;
        const isDir = data._info && data._info.is_dir;

        if (isDir || hasChildren) {
            html += '<div class="doc-item folder" onclick="toggleDocFolder(this)">📁 ' + escapeHtml(name) + '</div>';
            html += '<div class="doc-folder-items">';
            html += renderDocFolder(data._children, path, pathParts);
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
            // 配置 marked
            marked.setOptions({
                gfm: true,
                breaks: true,
                headerIds: true,
                mangle: false,
                sanitize: false // 信任后端返回的内容
            });
            viewer.innerHTML = '<div class="markdown-content">' + marked.parse(content) + '</div>';
        } else {
            viewer.innerHTML = '<div class="empty-state">加载失败</div>';
        }
    } catch (e) {
        viewer.innerHTML = '<div class="empty-state">加载失败</div>';
    }
}


// 移除手动 Markdown 解析函数


// =========================================
// 可恢复任务
// =========================================
let recoverableData = [];

async function loadRecoverableTasks() {
    try {
        const response = await fetch('/api/task/recoverable');
        const data = await response.json();
        if (data.success && data.tasks && data.tasks.length > 0) {
            recoverableData = data.tasks;
            renderRecoverableTasks();
            document.getElementById('recoverableTasks').style.display = 'block';
        } else {
            document.getElementById('recoverableTasks').style.display = 'none';
        }
    } catch (e) {
        console.error('加载可恢复任务失败', e);
        document.getElementById('recoverableTasks').style.display = 'none';
    }
}

function renderRecoverableTasks() {
    const container = document.getElementById('recoverableList');
    if (!recoverableData || recoverableData.length === 0) {
        container.innerHTML = '<div class="empty-state">暂无可恢复的任务</div>';
        return;
    }

    let html = '';
    recoverableData.forEach(task => {
        const statusClass = task.status === 'running' ? 'running' : 'paused';
        html += '<div class="recoverable-item">';
        html += '<div class="recoverable-info">';
        html += '<span class="recoverable-status ' + statusClass + '"></span>';
        html += '<span class="recoverable-title">' + escapeHtml(task.title) + '</span>';
        html += '<span class="recoverable-folder">' + escapeHtml(task.task_folder) + '</span>';
        html += '</div>';
        html += '<button class="btn btn-primary" onclick="recoverTask(\'' + escapeHtml(task.task_folder).replace(/'/g, "\\'") + '\')">恢复</button>';
        html += '</div>';
    });
    container.innerHTML = html;
}

async function recoverTask(taskFolder) {
    try {
        const response = await fetch('/api/task/recover/' + encodeURIComponent(taskFolder), {
            method: 'POST'
        });
        const data = await response.json();
        if (data.success) {
            alert('任务恢复已启动！');
            // 隐藏可恢复任务列表
            document.getElementById('recoverableTasks').style.display = 'none';
        } else {
            alert('恢复失败: ' + (data.error || data.message));
        }
    } catch (e) {
        alert('恢复失败: ' + e.message);
    }
}

// 页面加载时尝试加载可恢复任务
setTimeout(loadRecoverableTasks, 1000);

connect();
