FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o whatsapp-viewer main.go parser.go memory-limiter.go

FROM alpine:latest AS deploy
WORKDIR /app
COPY --from=builder /app/whatsapp-viewer .
COPY templates ./templates
COPY assets ./assets
RUN mkdir tmp
EXPOSE 80
CMD ["./whatsapp-viewer"]