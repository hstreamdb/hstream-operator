package selector

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

func (e *Selector) GetPods(namespace string, labelMap *map[string]string, fieldMap *map[string]string) ([]v1.Pod, error) {
	listOptions := metav1.ListOptions{}

	if labelMap != nil {
		listOptions.LabelSelector = labels.FormatLabels(*labelMap)
	}

	if fieldMap != nil {
		listOptions.FieldSelector = fields.SelectorFromSet(*fieldMap).String()
	}

	podList, err := e.clientset.CoreV1().Pods(namespace).List(
		context.TODO(),
		listOptions,
	)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}
