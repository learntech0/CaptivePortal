package backend

import (
	"encoding/json"
	"net/http"
	"os"
)

func HandleCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	secret := os.Getenv("LOCAL_CONTROLLER_SECRET")
	if secret == "" {
		secret = "dKadWiF6iF5LRQiRBTiSefVwrOZwKw1MojQsWvuwi4q8iym3URREM7U4MiC8ulYw" // FIXED: Matches shell
	}
	if r.Header.Get("X-Router-Token") != secret {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	devices := GetPaidDevices()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"unlock": devices,
	})
}