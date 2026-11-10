package backend

import (
	"encoding/json"
	"net/http"
	"os"
)

type AckRequest struct {
	Macs []string `json:"macs"`
}

func HandleAck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	secret := os.Getenv("LOCAL_CONTROLLER_SECRET")
	if secret == "" || r.Header.Get("X-Router-Token") != secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var req AckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	AcknowledgeDevices(req.Macs)
	logf("ack: acked %d macs", len(req.Macs))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"acked"}`))
}