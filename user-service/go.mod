module github.com/music-streaming/user-service

go 1.23

require (
    github.com/go-redis/redis/v8 v8.11.5
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/google/uuid v1.6.0
    github.com/nats-io/nats.go v1.33.0
    github.com/stretchr/testify v1.8.1
    golang.org/x/crypto v0.18.0
    google.golang.org/grpc v1.59.0
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
)

// Remove the replace directive since proto is now local
// replace github.com/music-streaming/proto => ../proto
