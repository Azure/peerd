package election

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func TestLeader(t *testing.T) {
	cs := &kubernetes.Clientset{}
	le := newLeaderElection("test", "test", cs)
	if le == nil {
		t.Fatal("expected leader election")
	}

	// Ensure no leader yet.
	if le.id != "" {
		t.Fatal("expected no leader")
	}

	// Ensure no leader is returned as long as we haven't started the leader election.
	timeout, cancelFunc := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancelFunc()
	leaderElected := make(chan bool)

	go func() {
		defer close(leaderElected)
		_, err := le.Leader()
		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
		leaderElected <- true
	}()

	select {
	case <-leaderElected:
		t.Fatal("expected no leader")
	case <-timeout.Done():
		// No leader found as expected.
	}

	leaderAddr := "/ip4/127.0.0.1/tcp/5000"

	// Simulate leader election.
	le.onNewLeader(leaderAddr)

	got, err := le.Leader()
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != leaderAddr {
		t.Fatalf("expected leader %s, got %s", leaderAddr, got.String())
	}

	// Simulate leader unknown.
	le.onNewLeader(resourcelock.UnknownLeader)

	got, err = le.Leader()
	if err != nil {
		t.Fatal(err)
	}

	// We still use the previously available leader.
	if got.String() != leaderAddr {
		t.Fatalf("expected leader %s, got %s", leaderAddr, got.String())
	}

	leader2Addr := "/ip4/127.0.0.1/tcp/5001"

	// Simulate new leader elected.
	le.onNewLeader(leader2Addr)

	got, err = le.Leader()
	if err != nil {
		t.Fatal(err)
	}

	if got.String() != leader2Addr {
		t.Fatalf("expected leader %s, got %s", leader2Addr, got.String())
	}
}

func TestLeaderElectionConfig(t *testing.T) {
	cs := &kubernetes.Clientset{}
	le := newLeaderElection("test", "test", cs)
	if le == nil {
		t.Fatal("expected leader election")
	}

	c := le.leaderElectionConfig(nil)
	if c.Callbacks.OnNewLeader == nil {
		t.Fatal("expected onNewLeader callback")
	}

	if !c.ReleaseOnCancel {
		t.Fatal("expected release on cancel to be true")
	}
}

func TestUnexpectedLeaderId(t *testing.T) {
	cs := &kubernetes.Clientset{}
	le := newLeaderElection("test", "test", cs)
	if le == nil {
		t.Fatal("expected leader election")
	}

	le.id = "not-a-multi-addr"
	close(le.initChan)

	_, err := le.Leader()
	if err == nil {
		t.Fatal("expected error")
	}
}
