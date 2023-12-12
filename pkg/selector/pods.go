/*
Copyright 2023 HStream Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
