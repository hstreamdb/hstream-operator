package v1alpha2

import "fmt"

type ComponentType string

const (
	ComponentTypeHServer     ComponentType = "hserver"
	ComponentTypeHStore      ComponentType = "hstore"
	ComponentTypeAdminServer ComponentType = "admin-server"
	ComponentTypeHMeta       ComponentType = "hmeta"
)

func (ct ComponentType) GetResName(instance string) string {
	return fmt.Sprintf("%s-%s", instance, ct)
}
