FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata nodejs npm && \
    npm install -g pnpm

COPY go.mod go.sum ./
RUN go mod download

COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY web/ web/
RUN pnpm install --frozen-lockfile && pnpm build

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o subhub ./cmd/subhub

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/subhub .
RUN mkdir -p /data && chown -R appuser:appgroup /app /data && chmod +x subhub

USER appuser

EXPOSE 9000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9000/healthz || exit 1

CMD ["./subhub", "serve"]
