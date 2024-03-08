package events

import (
	"context"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/pkg/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	typedv1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

type eventsRecorderCtxKeyType struct{}

var (
	eventsRecorderCtxKey = eventsRecorderCtxKeyType{}
)

// NewRecorder creates a new event recorder.
func NewRecorder(ctx context.Context) (EventRecorder, error) {
	clientset, err := k8s.NewKubernetesInterface(p2pcontext.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	inPod := false
	_, err = rest.InClusterConfig() // Assume run in a Pod or an environment with appropriate env variables set
	if err == nil {
		inPod = true
	}

	var objRef *v1.ObjectReference
	if !inPod {
		node, err := clientset.CoreV1().Nodes().Get(ctx, p2pcontext.NodeName, metav1.GetOptions{})
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
		pod, err := clientset.CoreV1().Pods(p2pcontext.Namespace).Get(ctx, p2pcontext.NodeName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		objRef = &v1.ObjectReference{
			Kind:       "Pod",
			Name:       pod.Name,
			UID:        pod.UID,
			APIVersion: pod.APIVersion,
		}
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartStructuredLogging(4)
	broadcaster.StartRecordingToSink(&typedv1core.EventSinkImpl{Interface: clientset.CoreV1().Events("")})

	return &eventRecorder{
		recorder: broadcaster.NewRecorder(
			scheme.Scheme,
			v1.EventSource{},
		),
		nodeRef: objRef,
	}, nil
}

// WithContext returns a new context with an event recorder.
func WithContext(ctx context.Context) (context.Context, error) {
	er, err := NewRecorder(ctx)
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
	nodeRef  *v1.ObjectReference
}

// Active should be called to indicate that the node is active in the cluster.
func (er *eventRecorder) Active() {
	er.recorder.Eventf(er.nodeRef, v1.EventTypeNormal, "P2PActive", "P2P proxy is active on node %s", er.nodeRef.Name)
}

// Connected should be called to indicate that the node is connected to the cluster.
func (er *eventRecorder) Connected() {
	er.recorder.Eventf(er.nodeRef, v1.EventTypeNormal, "P2PConnected", "P2P proxy is connected to cluster on node %s", er.nodeRef.Name)
}

// Disconnected should be called to indicate that the node is disconnected from the cluster.
func (er *eventRecorder) Disconnected() {
	er.recorder.Eventf(er.nodeRef, v1.EventTypeWarning, "P2PDisconnected", "P2P proxy is disconnected from cluster on node %s", er.nodeRef.Name)
}

// Failed should be called to indicate that the node has failed.
func (er *eventRecorder) Failed() {
	er.recorder.Eventf(er.nodeRef, v1.EventTypeWarning, "P2PFailed", "P2P proxy failed on node %s", er.nodeRef.Name)
}

// Initializing should be called to indicate that the node is initializing.
func (er *eventRecorder) Initializing() {
	er.recorder.Eventf(er.nodeRef, v1.EventTypeNormal, "P2PInitializing", "P2P proxy is initializing on node %s", er.nodeRef.Name)
}

var _ EventRecorder = &eventRecorder{}
