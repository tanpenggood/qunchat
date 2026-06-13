# 暗号

轻量级、移动优先的群聊 Web 应用。支持 Go / Node.js / Python 三种后端，WebSocket 实时通信，零服务端消息存储。

## 功能

- 实时群聊 —— WebSocket 全双工通信，消息即发即收
- 昵称修改 —— 随时改名，群内广播通知
- 在线人数 —— 标题栏实时显示当前在线人数
- 消息持久化 —— localStorage 保存最近 500 条消息，刷新不丢
- 多标签同步 —— BroadcastChannel 跨标签同步昵称与清空操作
- 自我/他人区分 —— 基于持久化 `clientId`，改名不改气泡位置
- 多行消息 —— Shift+Enter 换行，Enter 发送
- 移动适配 —— safe-area-inset 刘海屏适配，触屏优化
- 连接状态 —— 顶部实时显示连接状态，断线自动重连

## 快速开始

```bash
# Go 后端（推荐）
go run ./backends/go

# Node.js 后端
cd backends/node && npm install && node dev.js

# Python 后端
pip install -r backends/python/requirements.txt
python backends/python/server.py
```

打开 http://localhost:8080

启动脚本也提供了交互式选择：

```bash
# Windows
.\start.ps1

# Linux / macOS
./start.sh
```

## 三种后端对比

| 后端   | 语言      | 适用场景         | 性能        | 依赖                       |
|--------|-----------|------------------|-------------|----------------------------|
| Go     | Go 1.22   | 生产部署         | ⚡ 高       | 无（标准库 + gorilla/websocket） |
| Node   | Node.js   | 原型开发 / 调试  | 🚀 中      | ws                         |
| Python | Python 3  | 教学 / 快速实验  | 🐢 低      | aiohttp                    |

## 生产部署（Docker）

```bash
# Go 后端（默认）
docker build --target go -t qunchat:go .
docker run -d -p 8080:8080 --name qunchat qunchat:go

# Node.js 后端
docker build --target node -t qunchat:node .
docker run -d -p 8080:8080 --name qunchat qunchat:node

# Python 后端
docker build --target python -t qunchat:python .
docker run -d -p 8080:8080 --name qunchat qunchat:python
```

环境变量：

| 变量   | 默认值   | 说明           |
|--------|----------|----------------|
| `PORT` | `8080`   | HTTP 服务端口  |

## 目录结构

```
qunchat/
├── backends/            # 三种后端实现
│   ├── go/              # Go 生产服务器
│   │   └── main.go
│   ├── node/            # Node.js 开发服务器
│   │   ├── dev.js
│   │   └── package.json
│   └── python/          # Python 开发服务器
│       ├── server.py
│       └── requirements.txt
├── frontend/            # 前端单页应用
│   └── index.html
├── docs/
│   ├── ARCHITECTURE.md
│   └── PRD.md
├── Dockerfile           # 多阶段构建（三种 target）
├── start.ps1            # Windows 启动脚本
├── start.sh             # Linux/macOS 启动脚本
├── go.mod
└── README.md
```

## 技术栈

| 层       | 技术                                |
|----------|-------------------------------------|
| 前端     | 原生 HTML/CSS/JS，无框架            |
| 通信     | WebSocket (gorilla/websocket)       |
| 后端     | Go 1.22 / Node.js / Python 3 可选   |
| 存储     | 浏览器 localStorage                 |
| 跨标签   | BroadcastChannel API                |
| 容器化   | Docker 多阶段构建 (alpine)          |

## 架构要点

- Go 生产服务器约 5-10 MB 常驻内存，goroutine 约 4-8 KB/连接
- 服务端**不存储**任何消息，仅负责转发和广播在线人数
- 消息格式统一为 JSON，包含 `name`, `text`, `time`, `senderId`, `online` 字段
- 系统消息（加入/离开/改名）的 `name` 固定为 `"系统"`

## 协议

WebSocket 消息格式：

```json
{"name":"alice","text":"大家好","time":1718000000000,"senderId":"abc123","online":5}
{"name":"系统","text":"alice 加入了群聊","time":1718000000000,"online":5}
{"cmd":"rename","name":"newname","senderId":"abc123"}
```

## 许可

MIT
