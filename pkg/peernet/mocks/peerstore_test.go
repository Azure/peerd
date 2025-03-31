package mocks

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto/pb"
)

// Add unit tests for all functions of peerstore.go.

func TestPrivKey(t *testing.T) {
	mps := MockPeerstore{}
	privKey := mps.PrivKey("peerID")
	if privKey == nil {
		t.Errorf("Expected non-nil private key, got nil")
	}
	if privKey.Type() != pb.KeyType_RSA {
		t.Errorf("Expected RSA private key, got %s", privKey.Type())
	}
}
