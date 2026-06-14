#!/usr/bin/env python3
import json
import os
import time
from pathlib import Path

from aiohttp import web

PORT = int(os.environ.get("PORT", "8080"))
ROOT = Path(__file__).resolve().parent.parent.parent / 'frontend'

clients: dict[web.WebSocketResponse, dict] = {}
user_sessions: dict[str, dict] = {}


async def broadcast(data: dict) -> None:
    data["online"] = len(user_sessions)
    msg = json.dumps(data, ensure_ascii=False)
    for ws in list(clients):
        try:
            await ws.send_str(msg)
        except (ConnectionResetError, ConnectionAbortedError):
            pass


async def websocket_handler(request: web.Request) -> web.WebSocketResponse:
    ws = web.WebSocketResponse()
    await ws.prepare(request)
    clients[ws] = {"name": "", "sender_id": ""}

    async for msg in ws:
        try:
            data = json.loads(msg.data)
        except json.JSONDecodeError:
            continue
        if not isinstance(data, dict):
            continue

        info = clients.get(ws)
        if info is None:
            continue

        if not info["name"]:
            n = data.get("name", "")
            sid = data.get("senderId", "")
            if not n or not sid:
                continue
            info["name"] = n
            info["sender_id"] = sid
            session = user_sessions.get(sid)
            if session:
                session["count"] += 1
                await ws.send_str(json.dumps({
                    "name": "系统",
                    "text": "已连接",
                    "online": len(user_sessions),
                }, ensure_ascii=False))
            else:
                user_sessions[sid] = {"name": n, "count": 1}
                await broadcast({
                    "name": "系统",
                    "text": f"{n} 加入了群聊",
                    "time": int(time.time() * 1000),
                })
            continue

        if data.get("cmd") == "rename":
            n = data.get("name", "")
            if not n:
                continue
            old_name = info["name"]
            info["name"] = n
            session = user_sessions.get(info["sender_id"])
            if session:
                session["name"] = n
            await broadcast({
                "name": "系统",
                "text": f"{old_name} 改名为 {n}",
                "time": int(time.time() * 1000),
            })
            continue

        if data.get("cmd") == "whoisonline":
            users = [s["name"] for s in user_sessions.values() if s["name"]]
            await ws.send_str(json.dumps({
                "type": "online_list",
                "users": users,
            }, ensure_ascii=False))
            continue

        text = data.get("text", "")
        if not text:
            continue
        data["name"] = info["name"]
        data["time"] = int(time.time() * 1000)
        await broadcast(data)

    info = clients.pop(ws, None)
    if info and info["name"] and info["sender_id"]:
        session = user_sessions.get(info["sender_id"])
        if session:
            session["count"] -= 1
            if session["count"] <= 0:
                del user_sessions[info["sender_id"]]
                await broadcast({
                    "name": "系统",
                    "text": f"{info['name']} 离开了群聊",
                    "time": int(time.time() * 1000),
                })
    return ws


async def static_handler(request: web.Request) -> web.FileResponse:
    path = request.path
    if path == "/":
        path = "/index.html"
    filepath = (ROOT / path.lstrip("/")).resolve()
    if not str(filepath).startswith(str(ROOT)):
        raise web.HTTPForbidden
    if filepath.is_file():
        return web.FileResponse(filepath)
    raise web.HTTPNotFound


def create_app() -> web.Application:
    app = web.Application()
    app.router.add_get("/ws", websocket_handler)
    app.router.add_get("/{tail:.*}", static_handler)
    return app


if __name__ == "__main__":
    app = create_app()
    print(f"Python server starting on :{PORT}")
    web.run_app(app, port=PORT)
