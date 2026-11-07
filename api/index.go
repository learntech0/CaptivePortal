package handler

import (
	"net/http"
	"strings"
	"captive-portal/backend"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/register") {
		backend.HandleRegister(w, r)
		return
	}
	if strings.HasSuffix(path, "/callback") {
		backend.HandleCallback(w, r)
		return
	}
	http.NotFound(w, r)
}
