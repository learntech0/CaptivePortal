package backend

import (
	"encoding/json"
	"net/http"
	"strconv" // ADDED: needed to convert duration string to int
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

	// Do the work FIRST, respond LAST (Vercel may freeze after WriteHeader)
	if cb.Body.StkCallback.ResultCode == 0 {
		refData := cb.Body.StkCallback.AccountReference
		parts := strings.Split(refData, "|")
		// Expected: "<hexmac>|<duration>"
		if len(parts) != 2 {
			logf("callback: malformed AccountReference=%q checkout=%s", refData, cb.Body.StkCallback.CheckoutRequestID)
		} else if mac := macFromHex(parts[0]); mac != "" {
			
			// FIXED: Convert the duration string to an integer
			durationInt, err := strconv.Atoi(parts[1])
			if err != nil {
				logf("callback: bad duration=%q checkout=%s", parts[1], cb.Body.StkCallback.CheckoutRequestID)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"Result":"Success"}`))
				return
			}

			InitRedis()
			key := "processed:" + cb.Body.StkCallback.CheckoutRequestID
			wasSet, _ := rdb.SetNX(ctx, key, "1", 24*time.Hour).Result()
			if wasSet {
				// FIXED: GrantAccess now correctly receives the integer
				err := GrantAccess(mac, durationInt)
				if err != nil {
					logf("callback: GrantAccess failed mac=%s err=%v", mac, err)
				} else {
					logf("callback: granted mac=%s duration=%d min", mac, durationInt)
				}
			} else {
				logf("callback: duplicate checkout=%s", cb.Body.StkCallback.CheckoutRequestID)
			}
		} else {
			logf("callback: bad mac hex=%q checkout=%s", parts[0], cb.Body.StkCallback.CheckoutRequestID)
		}
	} else {
		logf("callback: non-zero ResultCode=%d checkout=%s", cb.Body.StkCallback.ResultCode, cb.Body.StkCallback.CheckoutRequestID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Result":"Success"}`))
}

// "AABBCCDDEEFF" -> "AA:BB:CC:DD:EE:FF"
func macFromHex(s string) string {
	if len(s) != 12 {
		return ""
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return ""
		}
	}
	s = strings.ToUpper(s)
	return s[0:2] + ":" + s[2:4] + ":" + s[4:6] + ":" + s[6:8] + ":" + s[8:10] + ":" + s[10:12]
}