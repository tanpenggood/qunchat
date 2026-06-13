#!/usr/bin/env python3
import json
import os
import time
from pathlib import Path

from aiohttp import web

PORT = int(os.environ.get("PORT", "8080"))
ROOT = Path(__file__).resolve().parent.parent

clients: dict[web.WebSocketResponse, str] = {}


async def broadcast(data: dict) -> None:
    data["online"] = len(clients)
    msg = json.dumps(data, ensure_ascii=False)
    for ws in list(clients):
        try:
            await ws.send_str(msg)
        except (ConnectionResetError, ConnectionAbortedError):
            pass


async def websocket_handler(request: web.Request) -> web.WebSocketResponse:
    ws = web.WebSocketResponse()
    await ws.prepare(request)
    clients[ws] = ""
    name = ""

    async for msg in ws:
        try:
            data = json.loads(msg.data)
        except json.JSONDecodeError:
            continue
        if not isinstance(data, dict):
            continue

        if not name:
            n = data.get("name", "")
            if not n:
                continue
            name = n
            clients[ws] = name
            await broadcast({
                "name": "系统",
                "text": f"{name} 加入了群聊",
                "time": int(time.time() * 1000),
            })
            continue

        if data.get("cmd") == "rename":
            n = data.get("name", "")
            if not n:
                continue
            old_name = name
            name = n
            clients[ws] = name
            await broadcast({
                "name": "系统",
                "text": f"{old_name} 改名为 {name}",
                "time": int(time.time() * 1000),
            })
            continue

        text = data.get("text", "")
        if not text:
            continue
        data["name"] = name
        data["time"] = int(time.time() * 1000)
        await broadcast(data)

    clients.pop(ws, None)
    if name:
        await broadcast({
            "name": "系统",
            "text": f"{name} 离开了群聊",
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
