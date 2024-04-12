// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package election

import (
	"context"
	"sync"
	"time"

	"github.com/azure/peerd/pkg/k8s"
	"github.com/multiformats/go-multiaddr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// LeaderElection provides an interface to elect a leader in a kubernetes cluster.
type LeaderElection interface {
	// RunOrDie runs the leader election.
	RunOrDie(ctx context.Context, id string) error

	// Leader gets the address of the elected leader.
	Leader() (multiaddr.Multiaddr, error)
}

// leaderElection provides an implementation of LeaderElection.
type leaderElection struct {
	// ns is the namespace in which to run the leader election.
	ns string

	// name is the name of the leader election resource in the cluster.
	name string

	// id is the id of the elected leader.
	id string

	cs       kubernetes.Interface
	initChan chan interface{}
	mx       sync.RWMutex
}

var _ LeaderElection = &leaderElection{}

// Leader gets the address of the elected leader.
func (le *leaderElection) Leader() (multiaddr.Multiaddr, error) {
	<-le.initChan
	le.mx.RLock()
	defer le.mx.RUnlock()

	addr, err := multiaddr.NewMultiaddr(le.id)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// RunOrDie runs the leader election.
func (le *leaderElection) RunOrDie(ctx context.Context, id string) error {
	lockCfg := resourcelock.ResourceLockConfig{
		Identity: id,
	}

	rl, err := resourcelock.New(resourcelock.LeasesResourceLock, le.ns, le.name, le.cs.CoreV1(), le.cs.CoordinationV1(), lockCfg)
	if err != nil {
		return err
	}

	go leaderelection.RunOrDie(ctx, le.leaderElectionConfig(rl))
	return nil
}

// leaderElectionConfig creates a new configuration for the leader election.
func (le *leaderElection) leaderElectionConfig(rl resourcelock.Interface) leaderelection.LeaderElectionConfig {
	return leaderelection.LeaderElectionConfig{
		Lock:            rl,
		ReleaseOnCancel: true,
		LeaseDuration:   10 * time.Second,
		RenewDeadline:   5 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {},
			OnStoppedLeading: func() {},
			OnNewLeader:      le.onNewLeader,
		},
	}
}

// onNewLeader is called when a new leader is elected.
// It updates the leaderElection instance with the new leader's identity.
func (le *leaderElection) onNewLeader(identity string) {
	if identity == resourcelock.UnknownLeader {
		return
	}

	select {
	case <-le.initChan:
		break
	default:
		// A leader has been elected.
		close(le.initChan)
	}

	le.mx.Lock()
	defer le.mx.Unlock()
	le.id = identity
}

// New build a new LeaderElection instance with the given name.
func New(name string, cs *k8s.ClientSet) LeaderElection {
	return &leaderElection{
		ns:       cs.Namespace,
		name:     name,
		cs:       cs,
		initChan: make(chan interface{}),
	}
}
