package handler

import (
	"net/http"
	"strings"

	"captive-portal/backend"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	route := r.URL.Query().Get("route")
	if route == "" {
		route = r.URL.Path
	}

	if strings.HasSuffix(route, "register") {
		backend.HandleRegister(w, r)
		return
	}
	if strings.HasSuffix(route, "callback") {
		backend.HandleCallback(w, r)
		return
	}
	if strings.HasSuffix(route, "check") {
		backend.HandleCheck(w, r)
		return
	}
	if strings.HasSuffix(route, "ack") { // <-- NEW ROUTE
		backend.HandleAck(w, r)
		return
	}
	http.NotFound(w, r)
}