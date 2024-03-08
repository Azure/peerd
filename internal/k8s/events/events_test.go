package events

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

func TestExpectedEvents(t *testing.T) {
	er := &eventRecorder{
		recorder: &testRecorder{t},
		nodeRef: &v1.ObjectReference{
			Kind:       "Node",
			Name:       "node-name",
			UID:        "node.UID",
			APIVersion: "node.APIVersion",
		},
	}

	er.Active()
	er.Connected()
	er.Disconnected()
	er.Initializing()
	er.Failed()
}

func TestFromContext(t *testing.T) {
	er := &eventRecorder{
		recorder: &testRecorder{t},
		nodeRef: &v1.ObjectReference{
			Kind:       "Node",
			Name:       "node-name",
			UID:        "node.UID",
			APIVersion: "node.APIVersion",
		},
	}

	ctx := context.WithValue(context.Background(), eventsRecorderCtxKey, er)

	er2 := FromContext(ctx)
	if er != er2 {
		t.Errorf("expected event recorders to match")
	}
}

type testRecorder struct {
	t *testing.T
}

// AnnotatedEventf implements record.EventRecorder.
func (*testRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{}) {
	panic("unimplemented")
}

// Event implements record.EventRecorder.
func (*testRecorder) Event(object runtime.Object, eventtype string, reason string, message string) {
	panic("unimplemented")
}

// Eventf implements record.EventRecorder.
func (t *testRecorder) Eventf(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{}) {
	if reason != "P2PActive" && reason != "P2PConnected" && reason != "P2PDisconnected" && reason != "P2PInitializing" && reason != "P2PFailed" {
		t.t.Errorf("unexpected reason: %s", reason)
	}
}

var _ record.EventRecorder = &testRecorder{}
