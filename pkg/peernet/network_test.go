// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package peernet

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/azure/peerd/pkg/mocks"
)

func TestNew(t *testing.T) {
	h := &mocks.MockHost{PeerStore: &mocks.MockPeerstore{}}

	_, err := New(h)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDefaultTLSConfig(t *testing.T) {
	h := &mocks.MockHost{PeerStore: &mocks.MockPeerstore{}}

	n, err := New(h)
	if err != nil {
		t.Fatal(err)
	}

	config := n.DefaultTLSConfig()
	if config == nil {
		t.Fatal("expected non-nil TLS config")
	}

	if config.Certificates == nil {
		t.Fatal("expected non-nil certificates")
	}
}

func TestTransportFor(t *testing.T) {
	h := &mocks.MockHost{PeerStore: &mocks.MockPeerstore{}}

	n, err := New(h)
	if err != nil {
		t.Fatal(err)
	}

	testPeerTransport := n.(*network).transportFor("test-peer")
	if testPeerTransport == nil {
		t.Fatal("expected non-nil transport")
	}

	if testPeerTransport.TLSClientConfig == nil {
		t.Fatal("expected non-nil TLS config")
	}

	if testPeerTransport.TLSClientConfig.VerifyPeerCertificate == nil {
		t.Fatal("expected non-nil VerifyPeerCertificate")
	}

	if testPeerTransport.TLSClientConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatal("expected RequireAndVerifyClientCert")
	}

	if testPeerTransport.TLSClientConfig.InsecureSkipVerify != true {
		t.Fatal("expected InsecureSkipVerify")
	}

	testPeerTransport2 := n.(*network).transportFor("test-peer-2")
	if testPeerTransport2 == nil {
		t.Fatal("expected non-nil transport")
	} else if testPeerTransport2 == testPeerTransport {
		t.Fatal("expected different transport")
	}

	defaultTransport := n.(*network).transportFor("")
	if defaultTransport == nil {
		t.Fatal("expected non-nil transport")
	}

	if defaultTransport.TLSClientConfig == nil {
		t.Fatal("expected non-nil TLS config")
	}

	if defaultTransport.TLSClientConfig.ClientAuth != tls.NoClientCert {
		t.Fatal("expected NoClientCert")
	}
}

func TestHTTPClientFor(t *testing.T) {
	h := &mocks.MockHost{PeerStore: &mocks.MockPeerstore{}}

	n, err := New(h)
	if err != nil {
		t.Fatal(err)
	}

	c := n.HTTPClientFor("test-peer")
	if c == nil {
		t.Fatal("expected non-nil client")
	}

	if c.Transport == nil {
		t.Fatal("expected non-nil transport")
	}

	if c.Timeout == 0 {
		t.Fatal("expected non-zero timeout")
	}
}

func TestRoundTripperFor(t *testing.T) {
	h := &mocks.MockHost{PeerStore: &mocks.MockPeerstore{}}

	n, err := New(h)
	if err != nil {
		t.Fatal(err)
	}

	c := n.RoundTripperFor("test-peer")
	if c == nil {
		t.Fatal("expected non-nil client")
	}

	if c.(*http.Transport).TLSClientConfig == nil {
		t.Fatal("expected non-nil TLS config")
	}

	if c.(*http.Transport).TLSClientConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatal("expected RequireAndVerifyClientCert")
	}
}
