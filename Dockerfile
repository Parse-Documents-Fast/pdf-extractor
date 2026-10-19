FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pdf-extractor ./cmd/pdf-extractor

FROM alpine:3.18
RUN apk --no-cache add ca-certificates curl

RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

COPY --from=builder /build/pdf-extractor /usr/local/bin/

RUN chown appuser:appgroup /usr/local/bin/pdf-extractor

USER appuser

EXPOSE 8000

HEALTHCHECK --interval=10s --timeout=5s --retries=3 --start-period=5s \
    CMD curl -f http://localhost:8000/health || exit 1

ENTRYPOINT ["pdf-extractor"]
