FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY . .

RUN sed -i '/replace github.com\/music-streaming\/proto/d' go.mod 2>/dev/null || true

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o music-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates ffmpeg

WORKDIR /root/

COPY --from=builder /app/music-service .

RUN mkdir -p /uploads

EXPOSE 50052

CMD ["./music-service"]
