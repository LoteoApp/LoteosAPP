package server

import (
	"net/http"
	"time"
)

const (
	// Wall clock a client gets to send the largest body the API accepts
	// (16 MiB) on top of the time its handler may take.
	bodyReadBudget = 30 * time.Second
	// Room left for writing the response once the handler already finished.
	responseWriteBudget = 10 * time.Second
)

// New builds the HTTP server. maxHandlerTimeout is the longest per-route
// timeout registered on handler: the connection deadlines are derived from it
// so a request that runs to its own limit is never cut off mid-response,
// which for a write would leave the client retrying work that was committed.
func New(port string, handler http.Handler, maxHandlerTimeout time.Duration) *http.Server {
	readTimeout := maxHandlerTimeout + bodyReadBudget

	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      readTimeout + responseWriteBudget,
		IdleTimeout:       60 * time.Second,
	}
}
