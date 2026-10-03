FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api cmd/api/main.go

FROM alpine:3.18

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /build/api .

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 --start-period=5s \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./api"]