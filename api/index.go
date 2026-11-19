package handler

import (
	"net/http"

	"captive-portal/backend"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if enableCORS(w, r) {
        return // If it's an OPTIONS request, it has been handled, return directly
    }
	route := r.URL.Query().Get("route")
	switch route {
		case "register":
			backend.HandleRegister(w, r)
		case "callback":
			backend.HandleCallback(w, r)
		case "check":
			backend.HandleCheck(w, r)
		case "ack":
			backend.HandleAck(w, r)
		default:
			http.NotFound(w, r)
	}
}

func enableCORS(w http.ResponseWriter, r *http.Request) bool {
    // Allow your portal's origin. You can also use * for testing, but it's not recommended for production
    w.Header().Set("Access-Control-Allow-Origin", "http://hotspot.wifi") 
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    // Key: If it's a preflight OPTIONS request, return 204 directly to end it
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return true
    }
    return false
}