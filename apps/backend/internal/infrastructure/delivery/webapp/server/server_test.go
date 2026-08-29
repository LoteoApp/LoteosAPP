package server_test

import (
	"net/http"
	"testing"
	"time"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/route"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/server"
)

func TestNewOutlastsTheSlowestRoute(t *testing.T) {
	t.Parallel()

	httpServer := server.New("8080", http.NewServeMux(), route.MaxHandlerTimeout)

	if httpServer.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", httpServer.Addr)
	}
	// A route that runs to its own timeout still has to be able to read the
	// request and write the response: deadlines shorter than that would cut
	// the connection after the handler already committed its work.
	if httpServer.ReadTimeout <= route.MaxHandlerTimeout {
		t.Errorf("ReadTimeout = %v, want more than the slowest route (%v)", httpServer.ReadTimeout, route.MaxHandlerTimeout)
	}
	if httpServer.WriteTimeout <= httpServer.ReadTimeout {
		t.Errorf("WriteTimeout = %v, want more than ReadTimeout (%v)", httpServer.WriteTimeout, httpServer.ReadTimeout)
	}
	if httpServer.ReadHeaderTimeout <= 0 || httpServer.ReadHeaderTimeout >= httpServer.ReadTimeout {
		t.Errorf("ReadHeaderTimeout = %v, want a short deadline before ReadTimeout", httpServer.ReadHeaderTimeout)
	}
	if httpServer.IdleTimeout <= time.Second {
		t.Errorf("IdleTimeout = %v, want an idle connection to be closed eventually", httpServer.IdleTimeout)
	}
}
