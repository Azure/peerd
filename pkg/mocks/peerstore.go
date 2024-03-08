// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package mocks

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	multiaddr "github.com/multiformats/go-multiaddr"
)

// MockPeerstore provides a mock implementation of peerstore.Peerstore for unit testing.
type MockPeerstore struct{}

var _ peerstore.Peerstore = &MockPeerstore{}

func (*MockPeerstore) AddAddr(p peer.ID, addr multiaddr.Multiaddr, ttl time.Duration) {
	panic("unimplemented")
}

func (*MockPeerstore) AddAddrs(p peer.ID, addrs []multiaddr.Multiaddr, ttl time.Duration) {
	panic("unimplemented")
}

func (*MockPeerstore) AddPrivKey(peer.ID, crypto.PrivKey) error {
	panic("unimplemented")
}

func (*MockPeerstore) AddProtocols(peer.ID, ...protocol.ID) error {
	panic("unimplemented")
}

func (*MockPeerstore) AddPubKey(peer.ID, crypto.PubKey) error {
	panic("unimplemented")
}

func (*MockPeerstore) AddrStream(context.Context, peer.ID) <-chan multiaddr.Multiaddr {
	panic("unimplemented")
}

func (*MockPeerstore) Addrs(p peer.ID) []multiaddr.Multiaddr {
	panic("unimplemented")
}

func (*MockPeerstore) ClearAddrs(p peer.ID) {
	panic("unimplemented")
}

func (*MockPeerstore) Close() error {
	panic("unimplemented")
}

func (*MockPeerstore) FirstSupportedProtocol(peer.ID, ...protocol.ID) (protocol.ID, error) {
	panic("unimplemented")
}

func (*MockPeerstore) Get(p peer.ID, key string) (interface{}, error) {
	panic("unimplemented")
}

func (*MockPeerstore) GetProtocols(peer.ID) ([]protocol.ID, error) {
	panic("unimplemented")
}

func (*MockPeerstore) LatencyEWMA(peer.ID) time.Duration {
	panic("unimplemented")
}

func (*MockPeerstore) PeerInfo(peer.ID) peer.AddrInfo {
	panic("unimplemented")
}

func (*MockPeerstore) Peers() peer.IDSlice {
	panic("unimplemented")
}

func (*MockPeerstore) PeersWithAddrs() peer.IDSlice {
	panic("unimplemented")
}

func (*MockPeerstore) PeersWithKeys() peer.IDSlice {
	panic("unimplemented")
}

// PrivKey generates a new private key for the given peer.
func (*MockPeerstore) PrivKey(peer.ID) crypto.PrivKey {
	// Generate a new key for each peer.
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		panic(err)
	}
	return priv
}

func (*MockPeerstore) PubKey(peer.ID) crypto.PubKey {
	panic("unimplemented")
}

func (*MockPeerstore) Put(p peer.ID, key string, val interface{}) error {
	panic("unimplemented")
}

func (*MockPeerstore) RecordLatency(peer.ID, time.Duration) {
	panic("unimplemented")
}

func (*MockPeerstore) RemovePeer(peer.ID) {
	panic("unimplemented")
}

func (*MockPeerstore) RemoveProtocols(peer.ID, ...protocol.ID) error {
	panic("unimplemented")
}

func (*MockPeerstore) SetAddr(p peer.ID, addr multiaddr.Multiaddr, ttl time.Duration) {
	panic("unimplemented")
}

func (*MockPeerstore) SetAddrs(p peer.ID, addrs []multiaddr.Multiaddr, ttl time.Duration) {
	panic("unimplemented")
}

func (*MockPeerstore) SetProtocols(peer.ID, ...protocol.ID) error {
	panic("unimplemented")
}

func (*MockPeerstore) SupportsProtocols(peer.ID, ...protocol.ID) ([]protocol.ID, error) {
	panic("unimplemented")
}

func (*MockPeerstore) UpdateAddrs(p peer.ID, oldTTL time.Duration, newTTL time.Duration) {
	panic("unimplemented")
}
