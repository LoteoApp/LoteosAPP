package handler

import "net/http"

// Live reports whether the process is up. It never fails, so it stays a
// plain http.HandlerFunc instead of implementing HTTPHandler, and it is
// registered without auth so orchestrators (Dokploy/Docker) can probe it.
func Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
