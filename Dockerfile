# syntax=docker/dockerfile:1

# 1. Frontend: web/dist, which the Go binary embeds.
FROM node:24-slim AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# 2. Server: single static binary with the UI and migrations embedded.
FROM golang:1.26-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /src/web/dist web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# 3. Runtime.
FROM alpine:3.22
RUN adduser -D -H -u 10001 app && mkdir -p /app/data/uploads && chown -R app /app/data
WORKDIR /app
COPY --from=server /out/server /app/server
USER app
ENV ADDR=:8080 UPLOAD_DIR=/app/data/uploads
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
ENTRYPOINT ["/app/server"]
