package v1alpha2

const (
	// LastSpecKey is used to store the hash of the object spec.
	LastSpecKey = "hstream.io/last-applied-spec"

	// OldReplicas is used to store the old replicas of statefulsets/deployments.
	OldReplicas = "hstream.io/old-replicas"

	// NewReplicas is used to store the new replicas of statefulsets/deployments.
	NewReplicas = "hstream.io/new-replicas"
)
