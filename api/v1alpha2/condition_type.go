package v1alpha2

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HMetaReady         string = "HMetaReady"
	HStoreReady        string = "HStoreReady"
	HStoreUpdating     string = "HStoreUpdating"
	HStoreScalingUp    string = "HStoreScalingUp"
	HStoreScalingDown  string = "HStoreScalingDown"
	HStoreDraining     string = "HStoreDraining"
	HStoreDrained      string = "HStoreDrained"
	HServerReady       string = "HServerReady"
	HServerScalingUp   string = "HServerScalingUp"
	GatewayReady       string = "GatewayReady"
	ConsoleReady       string = "ConsoleReady"
	AllComponentsReady string = "AllComponentsReady"
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

	index, current := hdb.GetCondition(condition.Type)
	if index > -1 {
		if current.Status == condition.Status && !current.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = current.LastTransitionTime
		}

		hdb.Status.Conditions[index] = condition
	} else {
		hdb.Status.Conditions = append(hdb.Status.Conditions, condition)
	}

	sort.Slice(hdb.Status.Conditions, func(i, j int) bool {
		return hdb.Status.Conditions[j].LastTransitionTime.Before(&hdb.Status.Conditions[i].LastTransitionTime)
	})
}
