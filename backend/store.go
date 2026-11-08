package backend

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client

// InitRedis is called once at startup (or lazily) to connect to Upstash
func InitRedis() {
	if rdb != nil {
		return
	}
	
	// Upstash provides REDIS_URL (rediss://...) or REST credentials.
	// For go-redis, we need the standard Redis URL or host:port.
	redisURL := os.Getenv("UPSTASH_REDIS_URL") // or REDIS_URL
	if redisURL == "" {
		// Fallback for local testing (requires a local Redis or Docker)
		redisURL = "localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic("Invalid Redis URL: " + err.Error())
	}
	rdb = redis.NewClient(opt)
}

// AddPaidDevice stores the MAC and its duration in Redis, and adds it to the queue.
func AddPaidDevice(mac string, durationMinutes string) {
	InitRedis()
	// Store the duration for this MAC (expires from Redis after 24h to keep it clean)
	rdb.Set(ctx, "duration:"+mac, durationMinutes, 24*time.Hour)
	// Add to the pending queue set
	rdb.SAdd(ctx, "pending_queue", mac)
}

// GetPaidDevices retrieves all pending MACs and their durations, then removes them from the queue.
// It returns a slice of structs for easy JSON serialization.
type PaidDevice struct {
	Mac      string `json:"mac"`
	Duration string `json:"duration"`
}

func GetPaidDevices() []PaidDevice {
	InitRedis()
	// Get all members of the set
	macs, err := rdb.SMembers(ctx, "pending_queue").Result()
	if err != nil || len(macs) == 0 {
		return []PaidDevice{}
	}

	out := make([]PaidDevice, 0, len(macs))
	for _, mac := range macs {
		dur, _ := rdb.Get(ctx, "duration:"+mac).Result()
		out = append(out, PaidDevice{Mac: mac, Duration: dur})
		// Remove from queue after processing (or you can keep it and let the router ACK)
		rdb.SRem(ctx, "pending_queue", mac)
	}
	return out
}