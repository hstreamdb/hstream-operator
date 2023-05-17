package v1alpha2

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HStoreReady  string = "HStoreReady"
	HServerReady string = "HServerReady"
	GatewayReady string = "GatewayReady"
	Ready        string = "Ready"
)

func (hdb *HStreamDB) IsConditionTrue(conditionType string) bool {
	_, condition := hdb.GetCondition(conditionType)
	if condition == nil {
		return false
	}
	return condition.Status == metav1.ConditionTrue
}

func (hdb *HStreamDB) GetCondition(conditionType string) (int, *metav1.Condition) {
	for i := range hdb.Status.Conditions {
		c := hdb.Status.Conditions[i]
		if c.Type == conditionType {
			return i, &c
		}
	}
	return -1, nil
}

func (hdb *HStreamDB) SetCondition(condition metav1.Condition) {
	now := metav1.Now()
	condition.LastTransitionTime = now
	condition.ObservedGeneration = hdb.Generation

	index, storeCondition := hdb.GetCondition(condition.Type)
	if index != -1 {
		if storeCondition.Status == condition.Status && !storeCondition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = storeCondition.LastTransitionTime
		}
		hdb.Status.Conditions[index] = condition
	} else {
		hdb.Status.Conditions = append(hdb.Status.Conditions, condition)
	}

	sort.Slice(hdb.Status.Conditions, func(i, j int) bool {
		return hdb.Status.Conditions[j].LastTransitionTime.Before(&hdb.Status.Conditions[i].LastTransitionTime)
	})
}
