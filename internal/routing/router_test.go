// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package routing

import (
	"context"
	"errors"
	"testing"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/pkg/k8s"
	"github.com/dgraph-io/ristretto"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	corerouting "github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	multiaddr "github.com/multiformats/go-multiaddr"
	"k8s.io/client-go/kubernetes/fake"
)

var fakeClientset = k8s.ClientSet{Interface: fake.NewSimpleClientset(), InPod: true}

func TestResolveWithCache(t *testing.T) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatal(err)
	}

	h := &testHost{"host-id"}
	key := "some-key"

	tcr := &testCr{
		m: map[string][]string{},
	}

	r := &router{
		clientset:   &fakeClientset,
		host:        h,
		port:        "5000",
		lookupCache: c,
		rd:          routing.NewRoutingDiscovery(tcr),
	}

	ctx := context.Background()
	_, negCacheCallback, err := r.ResolveWithCache(ctx, key, false, 2)
	if err != nil {
		t.Fatal(err)
	}

	negCacheCallback()
	time.Sleep(250 * time.Millisecond) // allow cache to flush

	if val, ok := r.lookupCache.Get(key); !ok || val != p2pcontext.P2pLookupNotFoundValue {
		t.Errorf("expected key to be %s, got %s", p2pcontext.P2pLookupNotFoundValue, val)
	}
}

func TestResolve(t *testing.T) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatal(err)
	}

	h := &testHost{"host-id"}
	key := "some-key"
	contentId, err := createCid(key)
	if err != nil {
		t.Fatal(err)
	}

	r := &router{
		clientset:   &fakeClientset,
		host:        h,
		port:        "5000",
		lookupCache: c,
		rd: routing.NewRoutingDiscovery(&testCr{
			m: map[string][]string{
				contentId.String(): {"10.0.0.1", "10.0.0.2"},
			},
		}),
	}

	ctx := context.Background()
	got, err := r.Resolve(ctx, key, false, 2)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for info := range got {
		if info.Addr == "https://10.0.0.1:5000" || info.Addr == "https://10.0.0.2:5000" {
			count++
		} else {
			t.Errorf("expected peer1 or peer2, got %s", info)
		}

		if count == 2 {
			break
		}
	}

	if count != 2 {
		t.Errorf("expected 2 addresses, got %d", count)
	}
}

func TestProvide(t *testing.T) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatal(err)
	}

	h := &testHost{"host-id"}
	key := "some-key"
	contentId, err := createCid(key)
	if err != nil {
		t.Fatal(err)
	}
	tcr := &testCr{
		m: map[string][]string{},
	}

	r := &router{
		clientset:   &fakeClientset,
		host:        h,
		port:        "5000",
		lookupCache: c,
		rd:          routing.NewRoutingDiscovery(tcr),
	}

	ctx := context.Background()
	err = r.Advertise(ctx, []string{key})
	if err != nil {
		t.Fatal(err)
	}

	if len(tcr.provided) != 1 {
		t.Errorf("expected 1 cid to be provided, got %d", len(tcr.provided))
	} else if tcr.provided[0] != contentId {
		t.Errorf("expected cid %s to be provided, got %s", contentId, tcr.provided[0])
	}
}

type testCr struct {
	m        map[string][]string
	provided []cid.Cid
}

// FindProvidersAsync implements routing.ContentRouting.
func (t *testCr) FindProvidersAsync(ctx context.Context, c cid.Cid, count int) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo, count)
	if val, ok := t.m[c.String()]; ok {
		for _, addr := range val {
			ch <- peer.AddrInfo{ID: peer.ID(addr), Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/" + addr + "/tcp/5005")}}
		}
	}
	return ch
}

// Provide implements routing.ContentRouting.
func (t *testCr) Provide(ctx context.Context, c cid.Cid, advertise bool) error {
	if !advertise {
		return errors.New("advertise must be true")
	}
	t.provided = append(t.provided, c)
	return nil
}

var _ corerouting.ContentRouting = &testCr{}

type testHost struct {
	id peer.ID
}

// Addrs implements host.Host.
func (*testHost) Addrs() []multiaddr.Multiaddr {
	panic("unimplemented")
}

// Close implements host.Host.
func (*testHost) Close() error {
	panic("unimplemented")
}

// ConnManager implements host.Host.
func (*testHost) ConnManager() connmgr.ConnManager {
	panic("unimplemented")
}

// Connect implements host.Host.
func (*testHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
	panic("unimplemented")
}

// EventBus implements host.Host.
func (*testHost) EventBus() event.Bus {
	panic("unimplemented")
}

// ID implements host.Host.
func (th *testHost) ID() peer.ID {
	return th.id
}

// Mux implements host.Host.
func (*testHost) Mux() protocol.Switch {
	panic("unimplemented")
}

// Network implements host.Host.
func (*testHost) Network() network.Network {
	panic("unimplemented")
}

// NewStream implements host.Host.
func (*testHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	panic("unimplemented")
}

// Peerstore implements host.Host.
func (*testHost) Peerstore() peerstore.Peerstore {
	panic("unimplemented")
}

// RemoveStreamHandler implements host.Host.
func (*testHost) RemoveStreamHandler(pid protocol.ID) {
	panic("unimplemented")
}

// SetStreamHandler implements host.Host.
func (*testHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {
	panic("unimplemented")
}

// SetStreamHandlerMatch implements host.Host.
func (*testHost) SetStreamHandlerMatch(protocol.ID, func(protocol.ID) bool, network.StreamHandler) {
	panic("unimplemented")
}

var _ host.Host = &testHost{}
