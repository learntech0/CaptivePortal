package handler

import (
	"net/http"

	"captive-portal/backend"
)

func Handler(w http.ResponseWriter, r *http.Request) {
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