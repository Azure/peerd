package content

import (
	"net/http"

	"github.com/azure/peerd/internal/context"
)

// SetOutboundHeaders sets the mandatory headers for all outbound requests.
func SetOutboundHeaders(r *http.Request, correlationId string) {
	r.Header.Set(P2PHeaderKey, "true")
	r.Header.Set(CorrelationHeaderKey, correlationId)
	r.Header.Set(NodeHeaderKey, context.NodeName)
}
