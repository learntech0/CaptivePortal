package handler

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

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	var cb MpesaCallback
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if cb.Body.StkCallback.ResultCode == 0 {
		refData := cb.Body.StkCallback.AccountReference
		parts := strings.Split(refData, "|")
		
		// Ensure all packed segments are preserved (MAC, RouterIP, Duration)
		if len(parts) == 3 {
			macAddress := parts[0]
			routerIp := parts[1]
			durationMinutes := parts[2]

			// Send authorization instructions along with the session timeout to the local controller node
			secretKey := os.Getenv("LOCAL_CONTROLLER_SECRET")
			
			unlockPayload := map[string]interface{}{
				"macAddress":      macAddress,
				"routerIp":        routerIp,
				"durationMinutes": durationMinutes,
				"secret":          secretKey,
			}
			pBytes, _ := json.Marshal(unlockPayload)
			
			// Fire down to the local network edge point
			http.Post("https://ngrok-free.app", "application/json", bytes.NewBuffer(pBytes))
		}
	}

	w.WriteHeader(http.StatusOK)
}
