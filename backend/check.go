package backend

import (
	"encoding/json"
	"net/http"
	"os"
)

func HandleCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	secret := os.Getenv("LOCAL_CONTROLLER_SECRET")
	if secret == "" || r.Header.Get("X-Router-Token") != secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	devices := GetPaidDevices()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"unlock": devices,
	})
}