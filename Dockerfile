# ---------- Builder Stage ----------
FROM golang:1.25.8-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN go build -ldflags="-s -w" -o bytevault ./cmd/api

# ---------- Runtime Stage ----------
FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates curl ffmpeg poppler-utils

COPY --from=builder /app/bytevault .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
CMD curl -fsS http://localhost:8080/api/v1/health > /dev/null || exit 1

CMD ["./bytevault"]