# ── build ────────────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/penbun-api .

# ── runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.21

# ใบรับรองสำหรับต่อฐานข้อมูลแบบเข้ารหัส และโซนเวลาสำหรับแปลงวันที่
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 penbun

COPY --from=build /out/penbun-api /usr/local/bin/penbun-api

USER penbun
EXPOSE 8089

# ตรวจ readyz ไม่ใช่ healthz เพราะ readyz ยืนยันว่าต่อฐานข้อมูลได้จริง
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8089/readyz || exit 1

ENTRYPOINT ["/usr/local/bin/penbun-api"]
