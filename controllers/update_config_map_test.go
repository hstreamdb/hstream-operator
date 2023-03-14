package controllers

import (
	"context"
	"fmt"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"strconv"
)

var _ = Describe("UpdateConfigMap", func() {
	var hdb *hapi.HStreamDB
	var requeue *requeue
	updateConfigMap := updateConfigMap{}
	ctx := context.TODO()

	BeforeEach(func() {
		hdb = mock.CreateDefaultCR()
		err := k8sClient.Create(ctx, hdb)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = k8sClient.Delete(ctx, hdb)
	})

	Context("with a reconciled cluster", func() {
		BeforeEach(func() {
			requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
		})

		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var logDevice, nShard *corev1.ConfigMap
		It("should successfully get config map", func() {
			var err error
			logDevice, nShard, err = getConfigMaps(hdb)
			Expect(err).To(BeNil())
		})

		When("config maps have been deploy", func() {
			Context("reconcile though nothing change", func() {
				BeforeEach(func() {
					requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
				})

				It("should not requeue", func() {
					Expect(requeue).To(BeNil())
				})

				It("should get same uid", func() {
					newLogDevice, newNShards, err := getConfigMaps(hdb)
					Expect(err).To(BeNil())
					Expect(logDevice.UID).To(Equal(newLogDevice.UID))
					Expect(nShard.UID).To(Equal(newNShards.UID))
				})
			})

			Context("update config", func() {
				var oldNShard int32
				BeforeEach(func() {
					oldNShard = hdb.Spec.Config.NShards
					hdb.Spec.Config.NShards = 2
					hdb.Spec.Config.LogDeviceConfig = runtime.RawExtension{
						Raw: []byte(`
					{
						"server_settings": {
						  "enable-nodes-configuration-manager": "false",
						  "use-nodes-configuration-manager-nodes-configuration": "false",
						  "enable-node-self-registration": "false",
						  "enable-cluster-maintenance-state-machine": "false"
						}
					}
					`)}
					requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
					Expect(requeue).To(BeNil())
				})

				var newLogDevice, newNShard *corev1.ConfigMap
				It("should get new config map", func() {
					var err error
					newLogDevice, newNShard, err = getConfigMaps(hdb)
					Expect(err).To(BeNil())
				})

				It("should get new server_setting", func() {
					cm, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
					Expect(newLogDevice.Data).To(HaveKey(cm.MapKey))
					file := newLogDevice.Data[cm.MapKey]
					m := make(map[string]any)
					err := json.UnmarshalFromString(file, &m)
					Expect(err).To(BeNil())
					Expect(m).To(HaveKeyWithValue("server_settings", map[string]any{
						"enable-nodes-configuration-manager":                  "false",
						"use-nodes-configuration-manager-nodes-configuration": "false",
						"enable-node-self-registration":                       "false",
						"enable-cluster-maintenance-state-machine":            "false",
					}))
				})

				It("should get old nShard", func() {
					cm, _ := internal.ConfigMaps.Get(internal.NShardsConfig)
					Expect(newNShard.Data).To(HaveKey(cm.MapKey))
					Expect(newNShard.Data).To(HaveKeyWithValue("NSHARDS", strconv.Itoa(int(oldNShard))))
				})
			})

			Context("use external hmeta cluster", func() {
				BeforeEach(func() {
					hdb.Spec.ExternalHMeta = &hapi.ExternalHMeta{
						Host:      "rqlite-svc",
						Port:      4001,
						Namespace: "default",
					}

					requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
					Expect(requeue).To(BeNil())
				})

				var newLogDevice *corev1.ConfigMap
				It("should get new config map", func() {
					var err error
					newLogDevice, _, err = getConfigMaps(hdb)
					Expect(err).To(BeNil())
				})

				It("should get new server_setting", func() {
					cm, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
					Expect(newLogDevice.Data).To(HaveKey(cm.MapKey))
					file := newLogDevice.Data[cm.MapKey]
					m := make(map[string]any)
					err := json.UnmarshalFromString(file, &m)
					Expect(err).To(BeNil())
					Expect(m).To(HaveKeyWithValue("rqlite", map[string]any{
						"rqlite_uri": "ip://rqlite-svc.default:4001",
					}))
				})
			})
		})
	})

	Context("with a invalid config", func() {
		BeforeEach(func() {
			hdb.Spec.Config.LogDeviceConfig = runtime.RawExtension{
				Raw: []byte("invalid config"),
			}
			requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
		})

		It("should requeue", func() {
			Expect(requeue).NotTo(BeNil())
		})

		It("get error 'invalid raw'", func() {
			Expect(requeue.curError).NotTo(BeNil())
			Expect(requeue.curError.Error()).To(ContainSubstring("parse log device config failed: invalid json format"))
		})
	})

	Context("with default nshard", func() {
		BeforeEach(func() {
			hdb.Spec.Config.NShards = 0
			requeue = updateConfigMap.reconcile(ctx, clusterReconciler, hdb)
		})
		It("should not requeue", func() {
			Expect(requeue).To(BeNil())
		})

		var nShard *corev1.ConfigMap
		It("should successfully get config map", func() {
			var err error
			_, nShard, err = getConfigMaps(hdb)
			Expect(err).To(BeNil())
		})

		It("NSHARDS should be '1'", func() {
			Expect(nShard.Data["NSHARDS"]).To(Equal("1"))
		})
	})
})

func getConfigMaps(hdb *hapi.HStreamDB) (logDevice, nShards *corev1.ConfigMap, err error) {
	config, _ := internal.ConfigMaps.Get(internal.LogDeviceConfig)
	keyObj := types.NamespacedName{
		Namespace: hdb.Namespace,
		Name:      internal.GetResNameOnPanic(hdb, config.MapNameSuffix),
	}
	logDevice = &corev1.ConfigMap{}
	if err = k8sClient.Get(context.TODO(), keyObj, logDevice); err != nil {
		err = fmt.Errorf("get log device config failed: %w", err)
		return
	}

	config, _ = internal.ConfigMaps.Get(internal.NShardsConfig)
	keyObj.Name = internal.GetResNameOnPanic(hdb, config.MapNameSuffix)
	nShards = &corev1.ConfigMap{}
	if err = k8sClient.Get(context.TODO(), keyObj, nShards); err != nil {
		err = fmt.Errorf("get nshard config failed: %w", err)
		return
	}
	return
}
