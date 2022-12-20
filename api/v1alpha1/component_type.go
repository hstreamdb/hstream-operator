package v1alpha1

import "fmt"

type ComponentType string

const (
	ComponentTypeHServer     ComponentType = "hserver"
	ComponentTypeHStore      ComponentType = "hstore"
	ComponentTypeAdminServer ComponentType = "admin-server"
)

func (ct ComponentType) GetResName(instance string) string {
	return fmt.Sprintf("%s-%s", instance, ct)
}
