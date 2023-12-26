package connectorgen

import (
	"github.com/hstreamdb/hstream-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func GenConnectorContainer(connector *v1beta1.Connector, name, stream, configMapName string) *corev1.Container {
	switch connector.Spec.Type {
	case v1beta1.SinkElaticsearch:
		return DefaultSinkElasticsearchContainer(connector, name, configMapName)
	case v1beta1.ExternalSource:
		return DefaultExternalSourceContainer(connector, name, stream)
	default:
		return nil
	}
}
