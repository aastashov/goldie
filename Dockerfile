# ===== Stage 1: Build =====
FROM golang:1.24.7-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata curl build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o goldie .

# ===== Stage 2: Runtime =====
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/goldie .
COPY locales/ ./locales/
COPY --from=builder /usr/share/zoneinfo/Asia/Bishkek /usr/share/zoneinfo/Asia/Bishkek

ENV TZ=Asia/Bishkek

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD curl -f http://localhost:8080/health || exit 1

CMD ["./goldie", "serve"]
