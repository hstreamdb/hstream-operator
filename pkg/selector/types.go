package selector

import "k8s.io/client-go/kubernetes"

type Selector struct {
	clientset *kubernetes.Clientset
}

func NewSelector(clientset *kubernetes.Clientset) *Selector {
	return &Selector{
		clientset: clientset,
	}
}
