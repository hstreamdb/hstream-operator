package controller

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal/admin"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	drainingInterval = time.Minute * 5
)

type HStoreMaintenanceReconciler struct{}

func (r HStoreMaintenanceReconciler) reconcile(ctx context.Context, hr *HStreamDBReconciler, hdb *hapi.HStreamDB) *requeue {
	logger := log.WithValues("namespace", hdb.Namespace, "instance", hdb.Name, "reconciler", "HStoreMaintenanceReconciler")

	if !hdb.IsConditionTrue(hapi.HStoreUpdating) {
		return nil
	}

	sts := appsv1.StatefulSet{}
	err := hr.Client.Get(ctx, types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      hapi.ComponentTypeHStore.GetResName(hdb.Name),
	}, &sts)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return &requeue{curError: err}
	}

	exist, condition := hdb.GetCondition(hapi.HStoreUpdating)
	if exist > -1 && condition.Reason == hapi.HStoreScalingDown {
		logger.Info("start to drain HStore")

		if _, ok := sts.Annotations[hapi.OldReplicas]; !ok {
			log.Info("HStore is not ready to drain because old replicas is not set, wait for next reconcile")

			return &requeue{delay: time.Second * 5}
		}

		oldReplicas, _ := strconv.ParseInt(sts.Annotations[hapi.OldReplicas], 10, 32)
		newReplicas, _ := strconv.ParseInt(sts.Annotations[hapi.NewReplicas], 10, 32)

		args := []string{
			"--shard_target_state", "drained",
			"--reason", "will shrink",
		}

		nodeNames := r.calcDrainedNodeNames(hdb, int(oldReplicas), int(newReplicas))
		for _, n := range nodeNames {
			args = append(args, "--node-names", n)
		}

		logger.Info("call HAdmin to drain HStore",
			"command", "hadmin store apply",
			"args", strings.Join(args, " "),
		)

		if _, err = hr.AdminClientProvider.GetHAdminClient(hdb).MaintenanceStore(
			admin.MaintenanceActionApply,
			args...,
		); err != nil {
			return &requeue{curError: err}
		}

		hdb.SetCondition(metav1.Condition{
			Type:   hapi.HStoreUpdating,
			Status: metav1.ConditionTrue,
			Reason: hapi.HStoreDraining,
		})

		if err = hr.Status().Update(ctx, hdb); err != nil {
			return &requeue{curError: fmt.Errorf("failed to update HStore status: %w", err)}
		}

		return &requeue{delay: time.Minute}
	}

	if exist > -1 && condition.Reason == hapi.HStoreDraining {
		oldReplicas, _ := strconv.ParseInt(sts.Annotations[hapi.OldReplicas], 10, 32)
		newReplicas, _ := strconv.ParseInt(sts.Annotations[hapi.NewReplicas], 10, 32)

		var statusOutput string
		if statusOutput, err = hr.AdminClientProvider.GetHAdminClient(hdb).CallStore("status"); err != nil {
			return &requeue{delay: time.Second}
		}

		nodeNames := r.calcDrainedNodeNames(hdb, int(oldReplicas), int(newReplicas))
		for _, n := range nodeNames {
			if r.isNodeDrained(statusOutput, n) {
				log.Info("a HStore node is drained successfully", "node", n)
			} else {
				log.Info("a HStore node is still draining, wait for next reconcile", "node", n)

				return &requeue{delay: drainingInterval}
			}
		}
	}

	return nil
}

func (r *HStoreMaintenanceReconciler) calcDrainedNodeNames(hdb *hapi.HStreamDB, oldReplicas int, newReplicas int) []string {
	var indexes []string

	for i := 0; i < oldReplicas; i++ {
		indexes = append(indexes, hapi.ComponentTypeHStore.GetResName(hdb.Name)+"-"+strconv.Itoa(i))
	}

	return indexes[newReplicas:]
}

func (r *HStoreMaintenanceReconciler) isNodeDrained(status, name string) bool {
	pattern, _ := regexp.Compile(fmt.Sprintf(`%s.+EMPTY\(1\)`, name))
	scanner := bufio.NewScanner(strings.NewReader(status))

	for scanner.Scan() {
		line := scanner.Text()

		if pattern.MatchString(line) {
			return true
		}
	}

	return false
}
