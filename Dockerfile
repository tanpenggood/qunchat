# ============================================================
# Qunchat — 多后端 Dockerfile
# 用法:
#   docker build --target go     -t qunchat:go     .
#   docker build --target node   -t qunchat:node   .
#   docker build --target python -t qunchat:python .
# ============================================================

# ---------- Go 后端 ----------
FROM m.daocloud.io/docker.io/library/golang:1.22-alpine AS go
WORKDIR /app
COPY go.mod go.sum* ./
COPY backends/go/ ./backends/go/
COPY frontend/index.html ./frontend/
RUN GOPROXY=https://goproxy.cn,direct go mod tidy && go build -o server ./backends/go
EXPOSE 8080
CMD ["./server"]


# ---------- Node 后端 ----------
FROM m.daocloud.io/docker.io/library/node:18-alpine AS node
WORKDIR /app
COPY backends/node/package*.json ./backends/node/
RUN cd backends/node && npm install
COPY backends/node/dev.js ./backends/node/
COPY frontend/index.html ./frontend/
EXPOSE 8080
CMD ["node", "backends/node/dev.js"]


# ---------- Python 后端 ----------
FROM m.daocloud.io/docker.io/library/python:3.11-slim AS python
WORKDIR /app
COPY backends/python/requirements.txt ./backends/python/
RUN pip install -r backends/python/requirements.txt --no-cache-dir
COPY backends/python/server.py ./backends/python/
COPY frontend/index.html ./frontend/
EXPOSE 8080
CMD ["python", "backends/python/server.py"]
