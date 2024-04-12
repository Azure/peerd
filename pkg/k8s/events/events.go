// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package events

import (
	"context"

	"github.com/azure/peerd/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	typedv1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

type eventsRecorderCtxKeyType struct{}

var (
	eventsRecorderCtxKey = eventsRecorderCtxKeyType{}
)

// NewRecorder creates a new event recorder.
func NewRecorder(ctx context.Context, k *k8s.ClientSet) (EventRecorder, error) {
	clientset := k.Interface
	inPod := k.InPod

	var objRef *v1.ObjectReference
	if !inPod {
		node, err := clientset.CoreV1().Nodes().Get(ctx, k.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		objRef = &v1.ObjectReference{
			Kind:       "Node",
			Name:       node.Name,
			UID:        node.UID,
			APIVersion: node.APIVersion,
		}
	} else {
		pod, err := clientset.CoreV1().Pods(k.Namespace).Get(ctx, k.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		objRef = &v1.ObjectReference{
			Kind:       "Pod",
			Name:       pod.Name,
			Namespace:  pod.Namespace,
			UID:        pod.UID,
			APIVersion: pod.APIVersion,
		}
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartStructuredLogging(4)
	broadcaster.StartRecordingToSink(&typedv1core.EventSinkImpl{Interface: clientset.CoreV1().Events(k.Namespace)})

	return &eventRecorder{
		recorder: broadcaster.NewRecorder(
			scheme.Scheme,
			v1.EventSource{},
		),
		objRef: objRef,
	}, nil
}

// WithContext returns a new context with an event recorder.
func WithContext(ctx context.Context, clientset *k8s.ClientSet) (context.Context, error) {
	er, err := NewRecorder(ctx, clientset)
	if err != nil {
		return nil, err
	}

	return context.WithValue(ctx, eventsRecorderCtxKey, er), nil
}

// FromContext returns the event recorder from the context.
func FromContext(ctx context.Context) EventRecorder {
	return ctx.Value(eventsRecorderCtxKey).(EventRecorder)
}

type eventRecorder struct {
	recorder record.EventRecorder
	objRef   *v1.ObjectReference
}

// Active should be called to indicate that the instance is active in the cluster.
func (er *eventRecorder) Active() {
	er.recorder.Eventf(er.objRef, v1.EventTypeNormal, "P2PActive", "P2P proxy is active on instance %s", er.objRef.Name)
}

// Connected should be called to indicate that the instance is connected to the cluster.
func (er *eventRecorder) Connected() {
	er.recorder.Eventf(er.objRef, v1.EventTypeNormal, "P2PConnected", "P2P proxy is connected to cluster on instance %s", er.objRef.Name)
}

// Disconnected should be called to indicate that the instance is disconnected from the cluster.
func (er *eventRecorder) Disconnected() {
	er.recorder.Eventf(er.objRef, v1.EventTypeWarning, "P2PDisconnected", "P2P proxy is disconnected from cluster on instance %s", er.objRef.Name)
}

// Failed should be called to indicate that the instance has failed.
func (er *eventRecorder) Failed() {
	er.recorder.Eventf(er.objRef, v1.EventTypeWarning, "P2PFailed", "P2P proxy failed on instance %s", er.objRef.Name)
}

// Initializing should be called to indicate that the instance is initializing.
func (er *eventRecorder) Initializing() {
	er.recorder.Eventf(er.objRef, v1.EventTypeNormal, "P2PInitializing", "P2P proxy is initializing on instance %s", er.objRef.Name)
}

var _ EventRecorder = &eventRecorder{}
