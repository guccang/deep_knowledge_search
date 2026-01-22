let ws;
const statusEl = document.getElementById('status');
const logsEl = document.getElementById('logs');
const canvas = document.getElementById('graphCanvas');
const ctx = canvas.getContext('2d');
const mainContent = document.getElementById('mainContent');
const detailPanel = document.getElementById('detailPanel');
const panelContent = document.getElementById('panelContent');
const panelTitle = document.getElementById('panelTitle');

let logCount = 0;
let taskData = null;
let nodes = [];
let selectedNode = null;

// 颜色配置
const colors = {
    pending: '#6b7280',
    running: '#3b82f6',
    done: '#22c55e',
    failed: '#ef4444',
    canceled: '#f59e0b',
    line: '#4a5568',
    text: '#e8e8e8',
    bg: '#1e1e2e'
};

// 初始化 Canvas
function resizeCanvas() {
    const container = canvas.parentElement;
    canvas.width = container.clientWidth;
    canvas.height = container.clientHeight;
    renderGraph();
}

window.addEventListener('resize', resizeCanvas);
setTimeout(resizeCanvas, 100);

// 连接 WebSocket
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
            renderGraph();
            break;
        case 'task_complete':
            if (taskData) taskData.status = 'done';
            addLog('info', '✅ 任务完成', msg.time);
            renderGraph();
            break;
        case 'task_failed':
            if (taskData) taskData.status = 'failed';
            addLog('error', '❌ 任务失败: ' + msg.data.error, msg.time);
            renderGraph();
            break;
        case 'node_start':
        case 'node_complete':
        case 'node_failed':
            updateTaskData(msg.data);
            addLog(msg.type === 'node_failed' ? 'error' : 'info',
                (msg.type === 'node_start' ? '▶ ' : msg.type === 'node_complete' ? '✓ ' : '✗ ') + msg.data.title,
                msg.time);
            renderGraph();
            break;
        case 'node_data':
            updateTaskData(msg.data);
            renderGraph();
            break;
        case 'tree_update':
            // 完整树更新，直接替换
            taskData = msg.data;
            renderGraph();
            break;
        case 'log':
            addLog(msg.data.level, msg.data.message, msg.time);
            break;
    }
}

function updateTaskData(nodeData) {
    if (!taskData) {
        taskData = nodeData;
    } else if (!taskData.id) {
        // task_start 创建的临时数据没有 id，用第一个完整节点替换
        if (nodeData.depth === 0) {
            taskData = nodeData;
        } else {
            // 如果是子节点，需要等待根节点
            mergeNodeData(taskData, nodeData);
        }
    } else {
        mergeNodeData(taskData, nodeData);
    }
}

function mergeNodeData(target, source) {
    // 1. 如果 ID 匹配，更新节点
    if (target.id === source.id) {
        // 保留已有的 children
        const existingChildren = target.children || [];
        Object.assign(target, source);
        // 合并 children（source 可能有新的 children 信息）
        if (source.children && source.children.length > 0) {
            target.children = source.children;
        } else if (existingChildren.length > 0) {
            target.children = existingChildren;
        }
        return true;
    }

    // 2. 检查是否应该添加为 target 的直接子节点
    if (source.parent_id === target.id) {
        if (!target.children) target.children = [];
        const existing = target.children.find(c => c.id === source.id);
        if (existing) {
            // 保留已有的 children
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

    // 3. 递归搜索所有子节点
    if (target.children) {
        for (let i = 0; i < target.children.length; i++) {
            if (mergeNodeData(target.children[i], source)) return true;
        }
    }

    return false;
}

// 图形渲染
function renderGraph() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    nodes = [];

    if (!taskData) {
        ctx.fillStyle = '#666';
        ctx.font = '14px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText('等待任务开始...', canvas.width / 2, canvas.height / 2);
        return;
    }

    // 计算布局
    const layout = calculateLayout(taskData, canvas.width, canvas.height);

    // 绘制连线
    drawConnections(layout);

    // 绘制节点
    drawNodes(layout);
}

function calculateLayout(node, width, height) {
    const nodeWidth = 140;
    const nodeHeight = 50;
    const levelHeight = 90;
    const result = [];

    function countLeaves(n) {
        if (!n.children || n.children.length === 0) return 1;
        return n.children.reduce((sum, c) => sum + countLeaves(c), 0);
    }

    function layoutNode(n, level, xStart, xEnd) {
        const x = (xStart + xEnd) / 2;
        const y = 40 + level * levelHeight;

        result.push({
            node: n,
            x: x,
            y: y,
            width: nodeWidth,
            height: nodeHeight
        });

        if (n.children && n.children.length > 0) {
            const totalLeaves = countLeaves(n);
            let currentX = xStart;

            for (const child of n.children) {
                const childLeaves = countLeaves(child);
                const childWidth = (xEnd - xStart) * (childLeaves / totalLeaves);
                layoutNode(child, level + 1, currentX, currentX + childWidth);
                currentX += childWidth;
            }
        }
    }

    layoutNode(node, 0, 50, width - 50);
    return result;
}

function drawConnections(layout) {
    ctx.strokeStyle = colors.line;
    ctx.lineWidth = 2;

    for (const item of layout) {
        if (item.node.children) {
            for (const child of item.node.children) {
                const childItem = layout.find(l => l.node.id === child.id);
                if (childItem) {
                    ctx.beginPath();
                    ctx.moveTo(item.x, item.y + item.height / 2);
                    ctx.bezierCurveTo(
                        item.x, item.y + item.height / 2 + 30,
                        childItem.x, childItem.y - 30,
                        childItem.x, childItem.y - item.height / 2
                    );
                    ctx.stroke();
                }
            }
        }
    }
}

function drawNodes(layout) {
    nodes = layout;

    for (const item of layout) {
        const isSelected = selectedNode && selectedNode.id === item.node.id;
        const status = item.node.status || 'pending';

        // 节点背景
        ctx.beginPath();
        const radius = 8;
        const x = item.x - item.width / 2;
        const y = item.y - item.height / 2;
        ctx.roundRect(x, y, item.width, item.height, radius);

        ctx.fillStyle = isSelected ? colors[status] : colors.bg;
        ctx.fill();

        ctx.strokeStyle = colors[status];
        ctx.lineWidth = isSelected ? 3 : 2;
        ctx.stroke();

        // 状态指示点
        ctx.beginPath();
        ctx.arc(x + 12, y + 12, 5, 0, Math.PI * 2);
        ctx.fillStyle = colors[status];
        ctx.fill();

        // 节点标题
        ctx.fillStyle = colors.text;
        ctx.font = '12px sans-serif';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';

        const title = item.node.title || 'Task';
        const maxLen = 16;
        const displayTitle = title.length > maxLen ? title.slice(0, maxLen) + '...' : title;
        ctx.fillText(displayTitle, item.x, item.y);
    }
}

// 点击处理
canvas.addEventListener('click', (e) => {
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    for (const item of nodes) {
        if (x >= item.x - item.width / 2 && x <= item.x + item.width / 2 &&
            y >= item.y - item.height / 2 && y <= item.y + item.height / 2) {
            selectNode(item.node);
            return;
        }
    }
});

function selectNode(node) {
    selectedNode = node;
    renderGraph();
    showNodeDetail(node);
}

function showNodeDetail(node) {
    panelTitle.textContent = node.title || 'Task Node';

    let html = '';

    // 基本信息
    html += '<div class="panel-section">';
    html += '<div class="section-title">📋 基本信息</div>';
    html += '<div class="node-info">';
    html += '<div class="info-row"><span class="info-label">ID:</span><span class="info-value">' + node.id + '</span></div>';
    html += '<div class="info-row"><span class="info-label">状态:</span><span class="info-value"><span class="status-dot status-' + (node.status || 'pending') + '"></span>' + (node.status || 'pending') + '</span></div>';
    if (node.description) {
        html += '<div class="info-row"><span class="info-label">描述:</span><span class="info-value">' + node.description + '</span></div>';
    }
    if (node.goal) {
        html += '<div class="info-row"><span class="info-label">目标:</span><span class="info-value">' + node.goal + '</span></div>';
    }
    html += '</div></div>';

    // LLM 调用记录
    if (node.llm_calls && node.llm_calls.length > 0) {
        html += '<div class="panel-section">';
        html += '<div class="section-title">🤖 LLM 调用记录 (' + node.llm_calls.length + ')</div>';

        node.llm_calls.forEach((call, idx) => {
            const typeLabels = { plan: '规划', execute: '执行', synthesize: '整合', verify: '验证' };
            html += '<div class="llm-call">';
            html += '<div class="llm-call-header" onclick="toggleLLMCall(' + idx + ')">';
            html += '<span class="llm-type">' + (typeLabels[call.type] || call.type) + '</span>';
            html += '<span class="llm-duration">' + call.duration_ms + 'ms</span>';
            html += '</div>';
            html += '<div class="llm-call-body" id="llm-call-' + idx + '">';

            html += '<div class="sub-label">请求消息:</div>';
            html += '<div class="code-block request">' + escapeHtml(JSON.stringify(call.messages, null, 2)) + '</div>';

            html += '<div class="sub-label">响应内容:</div>';
            html += '<div class="code-block response">' + escapeHtml(call.response) + '</div>';

            html += '</div></div>';
        });

        html += '</div>';
    }

    // 执行结果
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
    selectedNode = null;
    renderGraph();
}

function escapeHtml(text) {
    if (!text) return '';
    return text.toString()
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
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
    if (logCount > 50) {
        logsEl.removeChild(logsEl.lastChild);
    }
}

function clearLogs() {
    logsEl.innerHTML = '';
    logCount = 0;
}

connect();
