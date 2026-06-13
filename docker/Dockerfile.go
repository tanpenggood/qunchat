# ---------- Go 后端 ----------
FROM m.daocloud.io/docker.io/library/golang:1.22-alpine AS go
WORKDIR /app
COPY go.mod go.sum* ./
COPY backends/go/ ./backends/go/
COPY frontend/index.html ./frontend/
RUN GOPROXY=https://goproxy.cn,direct go mod tidy && go build -o server ./backends/go
EXPOSE 8080
CMD ["./server"]