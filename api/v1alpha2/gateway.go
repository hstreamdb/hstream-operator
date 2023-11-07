package v1alpha2

import corev1 "k8s.io/api/core/v1"

type Gateway struct {
	Component `json:",inline"`
	//+kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
	//+kubebuilder:default:=14789
	Port int32 `json:"port,omitempty"`
	// Must 'kubernetes.io/tls' secret, and the tls.key must the PKCS8 format
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}
