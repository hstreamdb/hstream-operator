package v1alpha2

type Gateway struct {
	Component `json:",inline"`
	//+kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
	//+kubebuilder:default:=14789
	Port int32 `json:"port"`
}
