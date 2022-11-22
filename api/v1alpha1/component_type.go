package v1alpha1

import "fmt"

type ComponentType string

const (
	ComponentTypeHServer     ComponentType = "hserver"
	ComponentTypeHStore      ComponentType = "hstore"
	ComponentTypeAdminServer ComponentType = "admin-server"
)

// IsStateful determines whether a component type should store data.
func (ct ComponentType) IsStateful() bool {
	// TODO: remove ComponentTypeHServer
	return ct == ComponentTypeHServer || ct == ComponentTypeHStore
}

func (ct ComponentType) GetResName(instance string) string {
	return fmt.Sprintf("%s-%s", instance, ct)
}
