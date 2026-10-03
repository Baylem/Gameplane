package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/ValgulNecron/gameplane/api/internal/gatewayprotocol"
)

const operationTimeout = 30 * time.Second

// requestContext keeps ordinary operations bounded without imposing their
// deadline on healthy streams or transfers. A positive MaxRequestDuration is
// an optional total lifetime for every operation. Peer certificate expiry and
// parent cancellation always apply, including to upgraded WebSockets.
func (h *handler) requestContext(req *http.Request, longRunning bool) (context.Context, context.CancelFunc) {
	deadline := req.TLS.PeerCertificates[0].NotAfter
	limit := h.cfg.MaxRequestDuration
	if !longRunning && (limit == 0 || limit > operationTimeout) {
		limit = operationTimeout
	}
	if limit > 0 {
		if maximum := time.Now().Add(limit); maximum.Before(deadline) {
			deadline = maximum
		}
	}
	return context.WithDeadline(req.Context(), deadline)
}

func longRunningOperation(method, path string) bool {
	if method == http.MethodGet {
		return gatewayprotocol.Streaming(path) || path == "/files/download" || path == "/logs/download"
	}
	return method == http.MethodPost && (path == "/files/upload" || path == "/mods/upload" || path == "/mods/install")
}
