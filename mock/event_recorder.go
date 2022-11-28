package mock

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

type mockEventRecorder struct {
}

func (m *mockEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {

}

// Eventf is just like Event, but with Sprintf for the message field.
func (m *mockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {

}

// AnnotatedEventf is just like eventf, but with annotations attached
func (m *mockEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {

}

func GetEventRecorderFor(name string) record.EventRecorder {
	return &mockEventRecorder{}
}
