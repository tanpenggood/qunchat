const http = require('http')
const fs = require('fs')
const path = require('path')
const { WebSocketServer } = require('ws')

const PORT = process.env.PORT || 8080
const ROOT = path.resolve(__dirname, '..')

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css',
  '.js': 'application/javascript',
  '.json': 'application/json',
  '.png': 'image/png',
  '.ico': 'image/x-icon',
}

const clients = new Set()

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
  data.online = clients.size
  const msg = JSON.stringify(data)
  for (const ws of clients) {
    if (ws.readyState === 1) ws.send(msg)
  }
}

wss.on('connection', (ws) => {
  clients.add(ws)
  let name = ''

  ws.on('message', (raw) => {
    let msg
    try { msg = JSON.parse(raw) } catch { return }
    if (!name) {
      if (!msg.name) return
      name = msg.name
      broadcast({ name: '系统', text: name + ' 加入了群聊', time: Date.now() })
      return
    }
    if (msg.cmd === 'rename') {
      if (!msg.name) return
      const oldName = name
      name = msg.name
      broadcast({ name: '系统', text: oldName + ' 改名为 ' + name, time: Date.now() })
      return
    }
    if (!msg.text) return
    msg.name = name
    msg.time = Date.now()
    broadcast(msg)
  })

  ws.on('close', () => {
    clients.delete(ws)
    if (name) broadcast({ name: '系统', text: name + ' 离开了群聊', time: Date.now() })
  })

  ws.on('error', () => clients.delete(ws))
})

server.listen(PORT, () => {
  console.log('Dev server at http://localhost:' + PORT)
  console.log('For production, build Go: cd qunchat && go build -o server ./server && ./server')
})
