// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package routing

import (
	"context"

	"github.com/azure/peerd/pkg/peernet"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Router provides a content routing interface to the network.
type Router interface {
	// Net returns the network interface.
	Net() peernet.Network

	// Resolve resolves the given key to a peer address.
	Resolve(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, error)

	// ResolveWithNegativeCacheCallback is like Resolve but it also returns a function callback that can be used to cache that a key could not be resolved.
	ResolveWithNegativeCacheCallback(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, func(), error)

	// Provide provides the given keys to the network.
	// This lets the k-closest peers to the key know that we are providing it.
	Provide(ctx context.Context, keys []string) error

	// Close closes the router.
	Close() error
}

// PeerInfo describes a peer.
type PeerInfo struct {
	peer.ID

	// HttpHost is the HTTP host of the peer.
	HttpHost string
}
