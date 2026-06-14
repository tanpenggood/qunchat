const http = require('http')
const fs = require('fs')
const path = require('path')
const { WebSocketServer } = require('ws')

const PORT = process.env.PORT || 8080
const ROOT = path.resolve(__dirname, '..', '..', 'frontend')

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css',
  '.js': 'application/javascript',
  '.json': 'application/json',
  '.png': 'image/png',
  '.ico': 'image/x-icon',
}

const clients = new Map()
const userSessions = new Map()

const mime = (p) => MIME[path.extname(p)] || 'application/octet-stream'

const server = http.createServer((req, res) => {
  let file = req.url === '/' ? '/index.html' : req.url
  file = path.join(ROOT, file)
  if (!file.startsWith(ROOT)) { res.writeHead(403); res.end(); return }
  fs.readFile(file, (err, data) => {
    if (err) { res.writeHead(404); res.end('Not Found'); return }
    res.writeHead(200, { 'Content-Type': mime(file) })
    res.end(data)
  })
})

const wss = new WebSocketServer({ server })

function broadcast(data) {
  data.online = userSessions.size
  const msg = JSON.stringify(data)
  for (const ws of clients.keys()) {
    if (ws.readyState === 1) ws.send(msg)
  }
}

wss.on('connection', (ws) => {
  clients.set(ws, { name: '', senderId: '' })

  ws.on('message', (raw) => {
    let msg
    try { msg = JSON.parse(raw) } catch { return }
    const info = clients.get(ws)
    if (!info) return

    if (!info.name) {
      if (!msg.name || !msg.senderId) return
      info.name = msg.name
      info.senderId = msg.senderId
      const session = userSessions.get(msg.senderId)
      if (session) {
        session.count++
        ws.send(JSON.stringify({ name: '系统', text: '已连接', online: userSessions.size }))
      } else {
        userSessions.set(msg.senderId, { name: msg.name, count: 1 })
        broadcast({ name: '系统', text: msg.name + ' 加入了群聊', time: Date.now() })
      }
      return
    }

    if (msg.cmd === 'rename') {
      if (!msg.name) return
      const oldName = info.name
      info.name = msg.name
      const session = userSessions.get(info.senderId)
      if (session) session.name = msg.name
      broadcast({ name: '系统', text: oldName + ' 改名为 ' + msg.name, time: Date.now() })
      return
    }

    if (msg.cmd === 'whoisonline') {
      const users = []
      for (const s of userSessions.values()) {
        if (s.name) users.push(s.name)
      }
      ws.send(JSON.stringify({ type: 'online_list', users }))
      return
    }

    if (!msg.text) return
    msg.name = info.name
    msg.time = Date.now()
    broadcast(msg)
  })

  ws.on('close', () => {
    const info = clients.get(ws)
    clients.delete(ws)
    if (info && info.name && info.senderId) {
      const session = userSessions.get(info.senderId)
      if (session) {
        session.count--
        if (session.count <= 0) {
          userSessions.delete(info.senderId)
          broadcast({ name: '系统', text: info.name + ' 离开了群聊', time: Date.now() })
        }
      }
    }
  })

  ws.on('error', () => clients.delete(ws))
})

server.listen(PORT, () => {
  console.log('Dev server at http://localhost:' + PORT)
  console.log('For production, build Go: go build -o server ./backends/go && ./server')
})
