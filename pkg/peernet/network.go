// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package peernet

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
)

const (
	// defaultTimeout is the total HTTP timeout that should work with most peers to download 1 Mb of data.
	defaultTimeout = 90 * time.Second
)

var (
	// defaultHttpClient is the default HTTP client that does not authenticate peers.
	defaultHttpClient = &http.Client{
		Timeout: defaultTimeout,
	}
)

// Network provides the transport and HTTP clients for communicating with peers.
type Network interface {
	// DefaultTLSConfig creates a default TLS config.
	// This config should not require client certificate verification.
	DefaultTLSConfig() *tls.Config

	// RoundTripperFor returns an HTTP round tripper which authenticates the given peer.
	// If pid is empty, the round tripper should work for any peer.
	RoundTripperFor(pid peer.ID) http.RoundTripper

	// HTTPClientFor returns an HTTP client which authenticates the given peer.
	// If pid is empty, the client should work for any peer.
	HTTPClientFor(pid peer.ID) *http.Client
}

type network struct {
	id               *libp2ptls.Identity
	defaultTLSConfig *tls.Config
	defaultTransport *http.Transport
}

var _ Network = &network{}

// DefaultTLSConfig creates a TLS config to use for this server.
// This config does not require client certificate verification and is reusable.
func (n *network) DefaultTLSConfig() *tls.Config {
	return n.defaultTLSConfig
}

// HTTPClientFor returns a single use HTTP client for the given peer.
// The client does not verify the peer's certificate.
func (n *network) HTTPClientFor(pid peer.ID) *http.Client {
	if pid == "" {
		return defaultHttpClient
	}

	return &http.Client{
		Transport: n.transportFor(pid),
		Timeout:   defaultTimeout,
	}
}

// RoundTripperFor returns a single use round tripper for the given peer.
// The peer is expected to provide a valid certificate.
// If pid is empty, the round tripper will work for any peer.
func (n *network) RoundTripperFor(pid peer.ID) http.RoundTripper {
	if pid == "" {
		return n.defaultTransport
	}

	return n.transportFor(pid)
}

// transportFor returns a single use transport for outbound connection to the given peer.
// The peer is expected to provide a valid certificate.
// If pid is empty, the transport will work for any peer.
func (n *network) transportFor(pid peer.ID) *http.Transport {
	if pid == "" {
		return n.defaultTransport
	}

	p2pTlsConfigForPeer, _ := n.id.ConfigForPeer(pid)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:          p2pTlsConfigForPeer.Certificates,
			ClientAuth:            tls.RequireAndVerifyClientCert,
			VerifyPeerCertificate: p2pTlsConfigForPeer.VerifyPeerCertificate,
			InsecureSkipVerify:    true,
		},
		MaxConnsPerHost: 100,
	}

	return transport
}

// New creates a new network interface for communicating with peers.
func New(h host.Host) (Network, error) {
	privKey := h.Peerstore().PrivKey(h.ID())

	id, err := libp2ptls.NewIdentity(privKey)
	if err != nil {
		return nil, err
	}

	tlsConfig, _ := id.ConfigForPeer(peer.ID(""))
	defaultTLSConfig := &tls.Config{
		Certificates: tlsConfig.Certificates,
	}

	defaultTransport := &http.Transport{
		TLSClientConfig: defaultTLSConfig,
		MaxConnsPerHost: 100,
	}

	return &network{
		id:               id,
		defaultTLSConfig: defaultTLSConfig,
		defaultTransport: defaultTransport,
	}, nil
}
