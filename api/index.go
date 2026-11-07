package handler

import (
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/register") {
		RegisterHandler(w, r)
		return
	}
	if strings.HasSuffix(path, "/callback") {
		CallbackHandler(w, r)
		return
	}
	http.NotFound(w, r)
}
