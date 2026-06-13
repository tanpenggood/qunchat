# ============================================================
# Qunchat — 多后端 Dockerfile
# 用法:
#   docker build --target go     -t qunchat:go     .
#   docker build --target node   -t qunchat:node   .
#   docker build --target python -t qunchat:python .
# ============================================================

# ---------- Go 后端 ----------
FROM golang:1.22-alpine AS go
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY server/ ./server/
COPY index.html ./
RUN go build -o server ./server
EXPOSE 8080
CMD ["./server"]


# ---------- Node 后端 ----------
FROM node:18-alpine AS node
WORKDIR /app
COPY scripts/package*.json ./scripts/
RUN cd scripts && npm install
COPY scripts/dev.js ./scripts/
COPY index.html ./
EXPOSE 8080
CMD ["node", "scripts/dev.js"]


# ---------- Python 后端 ----------
FROM python:3.11-slim AS python
WORKDIR /app
COPY scripts/requirements.txt ./scripts/
RUN pip install -r scripts/requirements.txt --no-cache-dir
COPY scripts/server.py ./scripts/
COPY index.html ./
EXPOSE 8080
CMD ["python", "scripts/server.py"]
