module github.com/music-streaming/music-service

go 1.19

require (
    github.com/go-redis/redis/v8 v8.11.5
    github.com/google/uuid v1.6.0
    github.com/nats-io/nats.go v1.33.0
    google.golang.org/grpc v1.59.0
    google.golang.org/protobuf v1.33.0
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)

replace github.com/music-streaming/proto => ../proto
