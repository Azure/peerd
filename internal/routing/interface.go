package routing

import (
	"context"

	"github.com/azure/peerd/pkg/peernet"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Router provides an interface to a peered network.
type Router interface {
	// Net returns the network interface.
	Net() peernet.Network

	// Resolve resolves the given key to a peer address.
	Resolve(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, error)

	// ResolveWithCache is like Resolve but it also returns a function callback that can be used to cache that a key could not be resolved.
	ResolveWithCache(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, func(), error)

	// Advertise advertises the given keys to the network.
	Advertise(ctx context.Context, keys []string) error

	// Close closes the router.
	Close() error
}

// PeerInfo describes a peer.
type PeerInfo struct {
	peer.ID
	Addr string
}
