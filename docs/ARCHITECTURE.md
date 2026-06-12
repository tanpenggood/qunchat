# 暗号 架构文档

## 系统架构

```
┌──────────────┐     WebSocket      ┌──────────────┐
│   Browser A   │ ◄───────────────► │              │
│  ┌──────────┐ │                   │  Go Server   │
│  │localStrge│ │                   │  (production) │
│  │Broadcast │ │                   │  ┌──────────┐ │
│  │ Channel  │─┼── 同源内跨标签 ──► │  │ Hub      │ │
│  └──────────┘ │                   │  │ clients[] │ │
├──────────────┤ │                   │  │ broadcast│ │
│   Browser B   │ ◄───────────────► │  │ register  │ │
│  ┌──────────┐ │                   │  │ unregister│ │
│  │localStrge│ │                   │  └──────────┘ │
│  └──────────┘ │                   └──────────────┘
└──────────────┘
```

### 组件说明

- **Browser**: 原生 HTML/CSS/JS 前端，单页应用
- **Go Server**: HTTP 静态文件服务 + WebSocket 端点 `/ws`
- **Hub**: 连接管理器，维护在线客户端集合，消息广播
- **Client**: 每个 WebSocket 连接对应一个 goroutine readPump + writePump
- **localStorage**: 前端持久化层，存放消息记录和用户配置
- **BroadcastChannel**: 浏览器同源跨标签页通信

---

## 数据流

### 用户发送消息

```
用户输入 → sendMessage() → ws.send(JSON) → Go Server readPump
  → Hub.broadcast ← Message ← Hub.Run() 设置 online 后 Marshal
  → 所有 Client.send channel → writePump → ws.WriteMessage
  → 各浏览器 onmessage → renderMsg() + pushMsg(localStorage)
```

### 用户加入

```
ws.onopen → ws.send({name}) → readPump → Hub.register
  → Hub.Run() 记录 client → broadcastSystem("加入了群聊")
  → 所有客户端收到系统消息 + online 更新
```

### 用户离开

```
连接断开 → Hub.unregister → delete(h.clients)
  → broadcastSystem("离开了群聊")
  → 所有客户端收到系统消息 + online 更新
```

---

## 服务端核心逻辑

### Hub

Hub 是服务端的心脏，三个 channel 驱动事件循环：

```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan Message    // 消息广播
    register   chan *Client    // 新连接注册
    unregister chan *Client    // 断开清理
    mu         sync.RWMutex
}
```

`Run()` 是一个 `select` 多路复用循环：

1. **register**: 将 client 加入 map，计数日志
2. **unregister**: 从 map 删除，关闭 send channel，广播离开消息
3. **broadcast**: 设置 `Online = len(clients)`，Marshal，遍历所有 client 发送

### Client

每个连接创建两个 goroutine：

| Goroutine | 职责 |
|-----------|------|
| `readPump` | 读取 WebSocket 消息 → 解析 Message → 处理命令/转发到 Hub.broadcast |
| `writePump` | 从 Hub 分发的 send channel 读取 → WebSocket 写出；30 秒 ping 保活 |

readPump 状态机：

```
首次消息 → 设置 c.name → broadcastSystem("加入了群聊")
rename 命令 → c.name = msg.Name → broadcastSystem("改名为")
普通消息 → 填充 name/time → Hub.broadcast
```

### Channel 类型变更

v1.0 中 `broadcast` channel 从 `chan []byte` 改为 `chan Message`，使 Hub 能在广播时注入 `Online` 字段：

- 旧: readPump 自行 Marshal → `broadcast <- []byte`
- 新: readPump 发送 `Message` 结构体 → Hub.Run() 中 Marshal

---

## 前端架构

### 模块划分

| 模块 | 位置 | 职责 |
|------|------|------|
| 登录 | `login-overlay` | 昵称输入，joinChat() |
| 聊天 | `#msg-area` | 消息渲染，滚动 |
| 输入 | `#input-bar` | 文本输入，发送 |
| 持久化 | `pushMsg/loadMsgs` | localStorage 读写 |
| 同步 | `BroadcastChannel` | 跨标签昵称/清空 |
| 连接 | `connect()` | WebSocket 生命周期管理 |

### 关键设计

**self/other 区分**：使用持久化 `clientId`（localStorage 生成）判断消息归属，而非对比昵称。

```js
const isSelf = data.senderId === clientId
```

**多标签同步**：BroadcastChannel 仅同步昵称变更和清空操作，不转发消息。所有标签共享同一 WebSocket 连接（同源），消息已通过服务端广播到达。

**消息存储**：环形缓冲区策略，超出 500 条时丢弃最旧消息；`QuotaExceededError` 时丢弃最旧 50%。

---

## 开发 vs 生产

| | 开发服务器 | 生产服务器 |
|--|-----------|-----------|
| 语言 | Node.js + ws | Go 1.22 |
| 启动 | `node scripts/dev.js` | Docker / 二进制 |
| WebSocket | `ws` 库 | `gorilla/websocket` |
| 静态文件 | 按需读取 | `http.FileServer` |
| 在线人数 | `clients.size` | `len(h.clients)` |
| 内存 | ~30-50 MB/空载 | ~5-10 MB/空载 |

---

## WebSocket 协议

### 端点

`GET /ws` — 升级为 WebSocket 连接

### 消息格式

所有消息均为 JSON，UTF-8 编码。

**客户端 → 服务端**

```json
{"name":"alice","text":"hello","senderId":"abc123"}
{"cmd":"rename","name":"newname","senderId":"abc123"}
```

**服务端 → 客户端**

```json
{"name":"alice","text":"hello","time":1718000000000,"senderId":"abc123","online":5}
{"name":"系统","text":"alice 加入了群聊","time":1718000000000,"online":5}
```

### 字段说明

| 字段 | 类型 | 出现于 | 说明 |
|------|------|--------|------|
| `cmd` | string | C→S | 命令类型（rename） |
| `name` | string | 双向 | 发送者昵称或 "系统" |
| `text` | string | 双向 | 消息文本 |
| `time` | number | S→C | Unix 毫秒时间戳 |
| `senderId` | string | 双向 | 持久化客户端标识 |
| `online` | number | S→C | 当前在线人数 |

### 心跳

- 服务端每 30 秒发送 Ping 帧
- 客户端返回 Pong 帧
- readPump 设置 60 秒读超时，依赖 PongHandler 刷新

---

## 部署

### Docker

多阶段构建：

1. `golang:1.22-alpine` 编译 Go 二进制
2. `alpine:3.20` 运行（仅 ~15MB 镜像）
3. 默认端口 8080，可通过 `PORT` 环境变量修改

### 裸机

```bash
cd qunchat
CGO_ENABLED=0 go build -o server ./server
./server  # 监听 :8080
```

---

## ADR 摘要

| 日期 | 决策 | 后果 |
|------|------|------|
| v1.0 | 选 Go 非 Node.js | 内存 ~5-10MB vs ~50MB+ |
| v1.0 | clientId 替代 name 做归属 | 改名不破气泡，但 ID 不可读 |
| v1.0 | localStorage 而非 IndexedDB | 简单够用，但 5MB 上限 |
| v1.0 | BroadcastChannel 有损降级 | 不支持的浏览器失去多标签同步 |
| v1.0 | broadcast channel 承载 Message 而非 []byte | Hub 可在 Marshal 前注入 Online |

---

## 安全注意事项

- `CheckOrigin` 放通所有来源 —— 仅适用于内网/受控环境
- 单消息 8KB 限制 —— 防内存溢出
- 无认证/鉴权 —— 信任模型
- 无消息服务端存储 —— 天然隐私友好
- WebSocket 路径固定 `/ws` —— 可通过反向代理路径隔离
