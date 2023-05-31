package v1alpha2

import "fmt"

type ComponentType string

const (
	ComponentTypeGateway     ComponentType = "gateway"
	ComponentTypeAdminServer ComponentType = "admin-server"
	ComponentTypeConsole     ComponentType = "console"
	ComponentTypeHServer     ComponentType = "hserver"
	ComponentTypeHStore      ComponentType = "hstore"
	ComponentTypeHMeta       ComponentType = "hmeta"
)

func (ct ComponentType) GetResName(instance string) string {
	return fmt.Sprintf("%s-%s", instance, ct)
}
