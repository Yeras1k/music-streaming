module github.com/music-streaming/music-service

go 1.19

require (
    github.com/go-redis/redis/v8 v8.11.5
    github.com/google/uuid v1.3.0
    github.com/nats-io/nats.go v1.31.0
    google.golang.org/grpc v1.56.3
    google.golang.org/protobuf v1.30.0
    gorm.io/driver/postgres v1.5.0
    gorm.io/gorm v1.25.0
)

replace github.com/music-streaming/proto => ../proto
