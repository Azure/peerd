// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package routing

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/k8s/events"
	"github.com/azure/peerd/pkg/k8s"
	"github.com/azure/peerd/pkg/k8s/election"
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

type router struct {
	clientset   *k8s.ClientSet
	p2pnet      peernet.Network
	host        host.Host
	rd          *routing.RoutingDiscovery
	port        string
	lookupCache *ristretto.Cache

	active atomic.Bool
}

// PeerNotFoundError indicates that no peer could be found for the given key.
type PeerNotFoundError struct {
	error
	key string
}

// NewRouter creates a new router.
func NewRouter(ctx context.Context, clientset *k8s.ClientSet, routerAddr, serverPort string) (Router, error) {
	log := zerolog.Ctx(ctx).With().Str("component", "router").Logger()

	h, p, err := net.SplitHostPort(routerAddr)
	if err != nil {
		return nil, err
	}

	multiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", h, p))
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

	host, err := libp2p.New(libp2p.ListenAddrs(multiAddr), factory)
	if err != nil {
		return nil, fmt.Errorf("could not create host: %w", err)
	}

	self := fmt.Sprintf("%s/p2p/%s", host.Addrs()[0].String(), host.ID().String())
	log.Info().Str("id", self).Msg("starting p2p router")

	leaderElection := election.New(p2pcontext.Namespace, "peerd-leader-election", p2pcontext.KubeConfigPath)

	err = leaderElection.RunOrDie(ctx, self)
	if err != nil {
		return nil, err
	}

	// TODO avtakkar: reconsider the max record age for cached files. Or, ensure that the cached list is periodically advertised.
	dhtOpts := []dht.Option{dht.Mode(dht.ModeServer), dht.ProtocolPrefix("/microsoft"), dht.DisableValues(), dht.MaxRecordAge(p2pcontext.KeyTTL)}
	bootstrapPeerOpt := dht.BootstrapPeersFunc(func() []peer.AddrInfo {
		addr, err := leaderElection.Leader()
		if err != nil {
			events.FromContext(ctx).Disconnected()
			log.Error().Err(err).Msg("could not get leader")
			return nil
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Error().Err(err).Msg("could not get leader")
			return nil
		}

		defer func() {
			events.FromContext(ctx).Connected()
		}()

		if addrInfo.ID == host.ID() {
			log.Info().Msg("leader is self, skipping connection to bootstrap node")
			return nil
		}

		log.Info().Str("node", addrInfo.ID.String()).Msg("bootstrap node found")
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
		clientset:   clientset,
		p2pnet:      n,
		host:        host,
		rd:          rd,
		port:        serverPort,
		lookupCache: c,
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

// ResolveWithCache is like Resolve but it also returns a function callback that can be used to cache that a key could not be resolved.
func (r *router) ResolveWithCache(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, func(), error) {
	if val, ok := r.lookupCache.Get(key); ok && val.(string) == p2pcontext.P2pLookupNotFoundValue {
		// TODO avtakkar: currently only doing a negative cache, this could maybe become a positive cache as well.
		return nil, nil, PeerNotFoundError{key: key, error: fmt.Errorf("(cached) peer not found for key")}
	}
	peerCh, err := r.Resolve(ctx, key, allowSelf, count)
	return peerCh, func() {
		r.lookupCache.SetWithTTL(key, p2pcontext.P2pLookupNotFoundValue, 1, p2pcontext.P2pLookupCacheTtl)
	}, err
}

// Resolve resolves the given key to a peer address.
func (r *router) Resolve(ctx context.Context, key string, allowSelf bool, count int) (<-chan PeerInfo, error) {
	log := zerolog.Ctx(ctx).With().Str("host", r.host.ID().String()).Str("key", key).Logger()
	c, err := createCid(key)
	if err != nil {
		return nil, err
	}
	addrCh := r.rd.FindProvidersAsync(ctx, c, count)
	peerCh := make(chan PeerInfo, count)
	go func() {
		for info := range addrCh {
			if !allowSelf && info.ID == r.host.ID() {
				continue
			}
			if len(info.Addrs) != 1 {
				log.Info().Msg("expected address list to only contain a single item")
				continue
			}

			v, err := info.Addrs[0].ValueForProtocol(multiaddr.P_IP4)
			if err != nil {
				log.Error().Err(err).Msg("could not get IPV4 address")
				continue
			}

			// Combine peer with registry port to create mirror endpoint.
			peerCh <- PeerInfo{info.ID, fmt.Sprintf("https://%s:%s", v, r.port)}

			if r.active.CompareAndSwap(false, true) {
				er, err := events.NewRecorder(ctx, r.clientset)
				if err != nil {
					log.Error().Err(err).Msg("could not create event recorder")
				} else {
					er.Active() // Report that p2p is active.
				}
			}
		}
	}()
	return peerCh, nil
}

// Advertise advertises the given keys to the network.
func (r *router) Advertise(ctx context.Context, keys []string) error {
	zerolog.Ctx(ctx).Trace().Str("host", r.host.ID().String()).Strs("keys", keys).Msg("advertising keys")
	for _, key := range keys {
		c, err := createCid(key)
		if err != nil {
			return err
		}
		err = r.rd.Provide(ctx, c, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func createCid(key string) (cid.Cid, error) {
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
