# 🔍 Deep Knowledge Search

知识深度搜索命令行工具 - 基于 LLM 的智能任务分解与执行系统

## ✨ 功能特性

### 🤖 智能任务分解
- 自动将复杂任务拆解为可执行的子任务
- 支持 **并行/串行** 执行模式
- 递归分解深度可配置（默认 3 层）

### ✅ 任务验证
- 执行后自动验证结果是否符合目标
- 迭代验证直到通过（最多 5 次）
- 验证不通过自动改进

### 📊 Web Dashboard
- 实时显示任务执行状态
- WebSocket 推送，无需刷新
- 任务树状结构可视化

### 📁 文件管理
- 每个任务独立文件夹存储
- 自动生成 `INDEX.md` 文章索引
- 详细执行日志 `execution.json`

---

## 🚀 快速开始

### 1. 配置
编辑 `config.json`:
```json
{
    "api_key": "your-api-key",
    "base_url": "https://api.deepseek.com/v1/chat/completions",
    "model": "deepseek-chat",
    "temperature": 0.3,
    "web_port": 8083,
    "web_enabled": true
}
```

### 2. 运行
```bash
# 命令行模式
./dks.exe "研究 Go 语言的并发模型"

# 交互模式
./dks.exe
```

### 3. 查看 Dashboard
打开浏览器访问: `http://localhost:8083`

---

## 📦 项目结构

```
deepknowledgesearch/
├── main.go              # 程序入口
├── config.json          # 配置文件
├── agent/               # Agent 模块
│   ├── agent.go         # 初始化
│   ├── task_node.go     # 任务节点
│   ├── executor.go      # 任务执行器
│   ├── planner.go       # 任务规划器
│   ├── prompts.go       # 提示词模板
│   ├── display.go       # 控制台显示
│   └── log_storage.go   # 日志存储
├── llm/                 # LLM 模块
│   ├── config.go        # LLM 配置
│   ├── client.go        # API 客户端
│   └── message.go       # 消息类型
├── mcp/                 # MCP 工具模块
│   ├── mcp.go           # 工具注册
│   └── tools.go         # 工具实现
├── web/                 # Web Dashboard
│   ├── server.go        # HTTP 服务器
│   └── hub.go           # WebSocket 中心
├── config/              # 配置模块
│   └── config.go        # 统一配置
├── output/              # 任务输出目录
└── logs/                # 执行日志目录
```

---

## 🔧 内置工具

| 工具 | 描述 |
|------|------|
| `saveToDisk` | 保存内容到文件 |

---

## 📋 输出示例

每次任务执行后生成:
```
output/任务名称_时间戳/
├── 子任务1_时间.md
├── 子任务2_时间.md
└── ...

logs/任务名称_时间戳/
├── execution.json    # 完整执行日志
├── summary.txt       # 任务摘要
└── INDEX.md          # 文章索引
```

---

## 🛠️ 配置说明

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `api_key` | LLM API 密钥 | 必填 |
| `base_url` | API 地址 | DeepSeek |
| `model` | 模型名称 | deepseek-chat |
| `temperature` | 生成温度 | 0.3 |
| `web_port` | Dashboard 端口 | 8080 |
| `web_enabled` | 启用 Web | true |

---

## 📄 License

MIT
