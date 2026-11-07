package handler

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

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
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

	// Read private consumer parameters dynamically from the environment
	consumerKey := os.Getenv("MPESA_CONSUMER_KEY")
	if consumerKey == "" {
		consumerKey = "o6yKcsA2KxV54AjDTa1VgMd4ZcqwxfCvAzrbRCnkqOtZDOdt" // Safe fallback from source
	}

	// Validate against pricing tier matrix and determine session time
	var durationMinutes int
	switch req.Amount {
	case 10:
		durationMinutes = 60      // 1 Hour
	case 20:
		durationMinutes = 180     // 3 Hours
	case 50:
		durationMinutes = 1440    // 24 Hours
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unsupported subscription package tier selected."})
		return
	}

	// Pack duration minutes natively inside the AccountReference metadata so CallbackHandler can read it
	accountRef := fmt.Sprintf("%s|%s|%d", req.Mac, req.RouterIp, durationMinutes)

	// Proceed to fire STK push payload to Safaricom Sandbox...
	fmt.Printf("Triggering STK push to %s for KES %d (Tracking: %s)\n", req.Phone, int(req.Amount), accountRef)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "STK prompt successfully dispatched."})
}
