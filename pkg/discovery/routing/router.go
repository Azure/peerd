// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package routing

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/azure/peerd/pkg/k8s"
	"github.com/azure/peerd/pkg/k8s/election"
	"github.com/azure/peerd/pkg/k8s/events"
	"github.com/azure/peerd/pkg/peernet"
	"github.com/dgraph-io/ristretto"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
	"github.com/rs/zerolog"
)

const (
	MaxRecordAge = 30 * time.Minute

	negCacheTtl     = 500 * time.Millisecond
	strPeerNotFound = "PEER_NOT_FOUND"
)

type router struct {
	// host is this libp2p host.
	host host.Host

	// p2pnet provides clients for downloading content from peers.
	p2pnet peernet.Network

	// content is the content discovery service.
	content *routing.RoutingDiscovery

	// peerRegistryPort is the port used for the peer registry.
	peerRegistryPort string

	// lookupCache is a cache for storing the results of lookups, usually used to store negative results.
	lookupCache *ristretto.Cache

	// k8sClient is the k8s client.
	k8sClient *k8s.ClientSet

	// active is a flag that indicates if this host is actively discovering content on the network.
	active atomic.Bool
}

var _ Router = &router{}

// ContentNotFoundError indicates that the content for the given key was not found in the network.
type ContentNotFoundError struct {
	error

	// key is the key that could not be resolved.
	key string
}

// NewRouter creates a new Router.
func NewRouter(ctx context.Context, clientset *k8s.ClientSet, hostAddr, peerRegistryPort string) (Router, error) {
	log := zerolog.Ctx(ctx).With().Str("component", "router").Logger()

	host, err := newHost(hostAddr)
	if err != nil {
		return nil, fmt.Errorf("could not create host: %w", err)
	}

	self := fmt.Sprintf("%s/p2p/%s", host.Addrs()[0].String(), host.ID().String())
	log.Debug().Str("id", self).Msg("starting p2p router")

	leaderElection := election.New("peerd-leader-election", clientset)

	err = leaderElection.RunOrDie(ctx, self)
	if err != nil {
		return nil, err
	}

	// TODO avtakkar: reconsider the max record age for cached files. Or, ensure that the cached list is periodically advertised.
	dhtOpts := []dht.Option{dht.Mode(dht.ModeServer), dht.ProtocolPrefix("/peerd"), dht.DisableValues(), dht.MaxRecordAge(MaxRecordAge)}
	bootstrapPeerOpt := dht.BootstrapPeersFunc(func() []peer.AddrInfo {
		addr, err := leaderElection.Leader()
		if err != nil {
			events.FromContext(ctx).Disconnected()
			log.Error().Err(err).Msg("could not get leader")
			return nil
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Error().Err(err).Msg("could not get leader addr info")
			return nil
		}

		defer func() {
			events.FromContext(ctx).Connected()
		}()

		if addrInfo.ID == host.ID() {
			log.Debug().Msg("bootstrapped as leader")
			return nil
		}

		log.Debug().Str("leader", addrInfo.ID.String()).Msg("leader found")
		return []peer.AddrInfo{*addrInfo}
	})

	dhtOpts = append(dhtOpts, bootstrapPeerOpt)
	kdht, err := dht.New(ctx, host, dhtOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create distributed hash table: %w", err)
	}
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("could not boostrap distributed hash table: %w", err)
	}
	rd := routing.NewRoutingDiscovery(kdht)

	c, err := ristretto.NewCache(&ristretto.Config{NumCounters: 1e7, MaxCost: 1073741824, BufferItems: 64})
	if err != nil {
		return nil, err
	}

	n, err := peernet.New(host)
	if err != nil {
		return nil, err
	}

	return &router{
		k8sClient:        clientset,
		p2pnet:           n,
		host:             host,
		content:          rd,
		peerRegistryPort: peerRegistryPort,
		lookupCache:      c,
	}, nil
}

// Transport returns the transport.
func (r *router) Net() peernet.Network {
	return r.p2pnet
}

// Close closes the router.
func (r *router) Close() error {
	return r.host.Close()
}

// ResolveWithNegativeCacheCallback is like Resolve but it also returns a function callback that can be used to cache that a key could not be resolved.
func (r *router) ResolveWithNegativeCacheCallback(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, func(), error) {
	if val, ok := r.lookupCache.Get(key); ok && val.(string) == strPeerNotFound {
		// TODO avtakkar: currently only doing a negative cache, this could maybe become a positive cache as well.
		return nil, nil, ContentNotFoundError{key: key, error: fmt.Errorf("(cached) peer not found for key")}
	}

	peerCh, err := r.Resolve(ctx, key, allowSelf, count)
	return peerCh, func() {
		r.lookupCache.SetWithTTL(key, strPeerNotFound, 1, negCacheTtl)
	}, err
}

// Resolve resolves the given key to a peer address.
func (r *router) Resolve(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, error) {
	log := zerolog.Ctx(ctx).With().Str("selfId", r.host.ID().String()).Str("key", key).Logger()
	contentId, err := createContentId(key)
	if err != nil {
		return nil, err
	}

	providersCh := r.content.FindProvidersAsync(ctx, contentId, count)
	peersCh := make(chan PeerInfo, count)

	go func() {
		for info := range providersCh {
			if !allowSelf && info.ID == r.host.ID() {
				continue
			}

			if len(info.Addrs) != 1 {
				log.Debug().Msg("expected address list to only contain a single item")
				continue
			}

			v, err := info.Addrs[0].ValueForProtocol(multiaddr.P_IP4)
			if err != nil {
				log.Error().Err(err).Str("peer", info.Addrs[0].String()).Msg("could not get IPV4 address")
				continue
			}

			// Combine peer with registry port to create mirror endpoint.
			peersCh <- PeerInfo{info.ID, fmt.Sprintf("https://%s:%s", v, r.peerRegistryPort)}

			if r.active.CompareAndSwap(false, true) {
				er, err := events.NewRecorder(ctx, r.k8sClient)
				if err != nil {
					log.Error().Err(err).Msg("failed to create event recorder")
				} else {
					er.Active() // Report that p2p is active.
				}
			}
		}
	}()

	return peersCh, nil
}

// Provide advertises the given keys to the network.
func (r *router) Provide(ctx context.Context, keys []string) error {
	zerolog.Ctx(ctx).Trace().Str("host", r.host.ID().String()).Strs("keys", keys).Msg("providing keys")
	for _, key := range keys {

		contentId, err := createContentId(key)
		if err != nil {
			return err
		}

		err = r.content.Provide(ctx, contentId, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// createContentId creates a deterministic content id from the given key.
func createContentId(key string) (cid.Cid, error) {
	pref := cid.Prefix{
		Version:  1,
		Codec:    uint64(mc.Raw),
		MhType:   mh.SHA2_256,
		MhLength: -1,
	}
	c, err := pref.Sum([]byte(key))
	if err != nil {
		return cid.Cid{}, err
	}
	return c, nil
}

// newHost creates a new Host from the given address.
func newHost(addr string) (host.Host, error) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", h, p))
	if err != nil {
		return nil, fmt.Errorf("could not create host multi address: %w", err)
	}

	factory := libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
		for _, addr := range addrs {
			v, err := addr.ValueForProtocol(multiaddr.P_IP4)
			if err != nil {
				continue
			}
			if v == "" {
				continue
			}
			if v == "127.0.0.1" {
				continue
			}
			return []multiaddr.Multiaddr{addr}
		}
		return nil
	})

	return libp2p.New(libp2p.ListenAddrs(hostAddr), factory)
}
