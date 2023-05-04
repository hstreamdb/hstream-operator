package v1alpha2

const (
	// LastSpecKey provides the annotation name we use to store the hash of the
	// object spec.
	LastSpecKey = "hstream.io/last-applied-spec"
	// ComponentKey provide the label name we use to store the type fo the
	// component
	ComponentKey = "hstream.io/component"
	// InstanceKey provide the label name we use to store the instance name
	InstanceKey = "hstream.io/instance"
)
