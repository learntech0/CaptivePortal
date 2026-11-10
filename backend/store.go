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

	redisURL := os.Getenv("UPSTASH_REDIS_URL")
	if redisURL == "" {
		redisURL = os.Getenv("REDIS_URL")
	}
	if redisURL == "" {
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
	rdb.Set(ctx, "duration:"+mac, durationMinutes, 24*time.Hour)
	rdb.SAdd(ctx, "pending_queue", mac)
}

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
			dur = "60"
		}
		out = append(out, PaidDevice{Mac: mac, Duration: dur})
	}
	return out
}

// Called only after the router confirms ndsctl auth succeeded.
func AcknowledgeDevices(macs []string) {
	InitRedis()
	for _, mac := range macs {
		rdb.SRem(ctx, "pending_queue", mac)
		rdb.Del(ctx, "duration:"+mac)
	}
}