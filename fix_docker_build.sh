#!/bin/bash

echo "=== Fixing Docker builds by copying proto to each service ==="

# Copy proto to each service
for service in user-service music-service payment-service api-gateway event-subscriber; do
    echo "Copying proto to $service..."
    cp -r proto $service/ 2>/dev/null || echo "Proto already in $service"
done

echo "=== Creating updated Dockerfiles ==="

# User service Dockerfile
cat > user-service/Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy everything including proto directory
COPY . .

# Remove the replace directive temporarily
RUN sed -i '/replace github.com\/music-streaming\/proto/d' go.mod

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1001 -S appuser && \
    adduser -u 1001 -S appuser -G appuser

WORKDIR /app

COPY --from=builder /app/user-service .
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 50051

ENTRYPOINT ["./user-service"]
EOF

# Music service Dockerfile
cat > music-service/Dockerfile << 'EOF'
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
EOF

# Payment service Dockerfile
cat > payment-service/Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY . .

RUN sed -i '/replace github.com\/music-streaming\/proto/d' go.mod 2>/dev/null || true

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o payment-service ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/payment-service .

EXPOSE 50053

CMD ["./payment-service"]
EOF

# API Gateway Dockerfile
cat > api-gateway/Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY . .

RUN sed -i '/replace github.com\/music-streaming\/proto/d' go.mod 2>/dev/null || true

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/api-gateway .

EXPOSE 8080

CMD ["./api-gateway"]
EOF

# Event subscriber Dockerfile
cat > event-subscriber/Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o event-subscriber ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/event-subscriber .

CMD ["./event-subscriber"]
EOF

echo "=== Building all services ==="
docker-compose build --no-cache

echo "=== Starting services ==="
docker-compose up -d

echo "=== Checking status ==="
sleep 10
docker-compose ps

echo "=== Done! Access frontend at http://localhost:3000 ==="
