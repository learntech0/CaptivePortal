package backend

import (
	"encoding/json"
	"net/http"
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
			// routerIp and durationMinutes are carried through the reference but
			// the router enforces expiry locally, so we only need to queue the MAC.
			_ = parts[1]
			_ = parts[2]

			// Persist the paid MAC into the cloud queue instead of pushing directly
			// to a local controller. The router polls /api/check to fetch it.
			AddPaidDevice(macAddress)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Result":"Success"}`))
}
