package events

// EventRecorder can be used to record various event.
type EventRecorder interface {
	// Initializing should be called to indicate that the node is initializing.
	Initializing()

	// Connected should be called to indicate that the node is connected to the cluster.
	Connected()

	// Active should be called to indicate that the node is active in the cluster.
	Active()

	// Disconnected should be called to indicate that the node is disconnected from the cluster.
	Disconnected()

	// Failed should be called to indicate that the node has failed.
	Failed()
}
