package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type PayRequest struct {
	Phone    string  `json:"phone"`
	Amount   float64 `json:"amount"`
	Mac      string  `json:"mac"`
	RouterIp string  `json:"router_ip"`
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid payload"})
		return
	}

	var durationMinutes int
	switch req.Amount {
	case 10:
		durationMinutes = 60
	case 20:
		durationMinutes = 180
	case 50:
		durationMinutes = 1440
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unsupported subscription package tier selected."})
		return
	}

	accountRef := fmt.Sprintf("%s|%s|%d", req.Mac, req.RouterIp, durationMinutes)
	fmt.Printf("Triggering STK push to %s (Tracking: %s)\n", req.Phone, accountRef)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "STK prompt successfully dispatched."})
}
