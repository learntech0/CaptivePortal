package backend

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client

func InitRedis() {
	if rdb != nil {
		return
	}
	
	// Vercel/Upstash injects this variable when you link the database
	redisURL := os.Getenv("UPSTASH_REDIS_URL") 
	if redisURL == "" {
		redisURL = os.Getenv("REDIS_URL")
	}
	if redisURL == "" {
		// Fallback for local testing
		redisURL = "localhost:6379" 
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic("Invalid Redis URL: " + err.Error())
	}
	rdb = redis.NewClient(opt)
}

type PaidDevice struct {
	Mac      string `json:"mac"`
	Duration string `json:"duration"`
}

func AddPaidDevice(mac string, durationMinutes string) {
	InitRedis()
	// Store duration (expires in 24h just to keep Redis clean)
	rdb.Set(ctx, "duration:"+mac, durationMinutes, 24*time.Hour)
	// Add to pending queue
	rdb.SAdd(ctx, "pending_queue", mac)
}

// GetPaidDevices ONLY READS. It does NOT delete.
func GetPaidDevices() []PaidDevice {
	InitRedis()
	macs, err := rdb.SMembers(ctx, "pending_queue").Result()
	if err != nil || len(macs) == 0 {
		return []PaidDevice{}
	}

	out := make([]PaidDevice, 0, len(macs))
	for _, mac := range macs {
		dur, _ := rdb.Get(ctx, "duration:"+mac).Result()
		if dur == "" {
			dur = "60" // Safe fallback
		}
		out = append(out, PaidDevice{Mac: mac, Duration: dur})
	}
	return out
}

// AcknowledgeDevices is called ONLY after the router confirms the firewall rule is active.
func AcknowledgeDevices(macs []string) {
	InitRedis()
	if len(macs) == 0 {
		return
	}
	for _, mac := range macs {
		// Now it is safe to remove them
		rdb.SRem(ctx, "pending_queue", mac)
	}
}