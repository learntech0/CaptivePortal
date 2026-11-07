package backend

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

type MpesaCallback struct {
	Body struct {
		StkCallback struct {
			ResultCode       int    `json:"ResultCode"`
			AccountReference string `json:"AccountReference"`
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

	if cb.Body.StkCallback.ResultCode == 0 {
		refData := cb.Body.StkCallback.AccountReference
		parts := strings.Split(refData, "|")
		
		if len(parts) == 3 {
			macAddress := parts[0]
			routerIp := parts[1]
			durationMinutes := parts[2]

			secretKey := os.Getenv("LOCAL_CONTROLLER_SECRET")
			if secretKey == "" {
				secretKey = "dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw"
			}
			
			unlockPayload := map[string]interface{}{
				"macAddress":      macAddress,
				"routerIp":        routerIp,
				"durationMinutes": durationMinutes,
				"secret":          secretKey,
			}
			pBytes, _ := json.Marshal(unlockPayload)
			
			// Fire down to the local network controller edge point
			http.Post("https://ngrok-free.app", "application/json", bytes.NewBuffer(pBytes))
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Result":"Success"}`))
}
