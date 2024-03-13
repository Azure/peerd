// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package mocks

import (
	"context"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	multiaddr "github.com/multiformats/go-multiaddr"
)

// MockHost provides a mock implementation of host.Host for unit testing.
type MockHost struct {
	PeerStore peerstore.Peerstore
}

var _ host.Host = &MockHost{}

func (*MockHost) Addrs() []multiaddr.Multiaddr {
	panic("unimplemented")
}

func (*MockHost) Close() error {
	panic("unimplemented")
}

func (*MockHost) ConnManager() connmgr.ConnManager {
	panic("unimplemented")
}

func (*MockHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
	panic("unimplemented")
}

func (*MockHost) EventBus() event.Bus {
	panic("unimplemented")
}

// ID returns the peer ID of this host.
func (*MockHost) ID() peer.ID {
	return "localhost-peer-for-unit-testing"
}

func (*MockHost) Mux() protocol.Switch {
	panic("unimplemented")
}

func (*MockHost) Network() network.Network {
	panic("unimplemented")
}

func (*MockHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	panic("unimplemented")
}

// Peerstore returns the mocked peerstore of this host.
func (m *MockHost) Peerstore() peerstore.Peerstore {
	return m.PeerStore
}

func (*MockHost) RemoveStreamHandler(pid protocol.ID) {
	panic("unimplemented")
}

func (*MockHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {
	panic("unimplemented")
}

func (*MockHost) SetStreamHandlerMatch(protocol.ID, func(protocol.ID) bool, network.StreamHandler) {
	panic("unimplemented")
}
