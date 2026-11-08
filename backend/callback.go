package backend

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type MpesaCallback struct {
	Body struct {
		StkCallback struct {
			ResultCode        int    `json:"ResultCode"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			AccountReference  string `json:"AccountReference"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

func HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var cb MpesaCallback
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Safaricom requires a 200 OK immediately to stop retries.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Result":"Success"}`))

	if cb.Body.StkCallback.ResultCode == 0 {
		refData := cb.Body.StkCallback.AccountReference
		parts := strings.Split(refData, "|")
		if len(parts) == 3 {
			macAddress := parts[0]
			durationMinutes := parts[2]

			// Idempotency check: If this CheckoutRequestID was already processed, skip.
			InitRedis()
			key := "processed:" + cb.Body.StkCallback.CheckoutRequestID
			wasSet, _ := rdb.SetNX(ctx, key, "1", 24*time.Hour).Result()
			if !wasSet {
				// Already processed this callback, do nothing.
				return
			}

			AddPaidDevice(macAddress, durationMinutes)
		}
	}
}