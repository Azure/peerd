package content

// Log messages.
const (
	PeerResolutionStartLog     = "peer resolution start"
	PeerResolutionStopLog      = "peer resolution stop"
	PeerNotFoundLog            = "peer not found"
	PeerResolutionExhaustedLog = "peer resolution exhausted"
	PeerRequestErrorLog        = "peer request error"
)

// Request headers.
const (
	P2PHeaderKey         = "X-MS-Peerd-RequestFromPeer"
	CorrelationHeaderKey = "X-MS-Peerd-CorrelationId"
	NodeHeaderKey        = "X-MS-Peerd-Node"
)
