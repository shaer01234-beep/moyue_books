# syntax=docker/dockerfile:1

# ============================================================
# 多阶段构建：前端内嵌进 Go 单二进制，最终镜像仅含运行所需文件。
# 用法：
#   docker build -t moyue-books .
#   docker run -d -p 18080:18080 -v $(pwd)/data:/app/data moyue-books
# ============================================================

# ---------- 1. 构建前端 ----------
FROM node:18-alpine AS frontend
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# 默认 base='./'，直接以根路径提供前端；fpk 网关前缀由 fnos/build.sh 注入，此处无需设置
RUN npm run build \
  && mkdir -p /dist \
  && cp -r dist/. /dist

# ---------- 2. 构建后端（前端经 //go:embed 内嵌） ----------
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache git
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# go:embed all:dist 需要 html/dist 在编译期存在
COPY --from=frontend /dist ./html/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/moyue-server .

# ---------- 3. 运行镜像 ----------
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S moyue && adduser -S moyue -G moyue

COPY --from=backend /out/moyue-server /app/moyue-server

ENV MB_DATA_DIR=/app/data \
    PORT=18080
EXPOSE 18080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:18080/ >/dev/null 2>&1 || exit 1

USER moyue
ENTRYPOINT ["/app/moyue-server"]
