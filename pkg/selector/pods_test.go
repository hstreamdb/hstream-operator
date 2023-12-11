package selector

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("selector/pods", func() {
	clientset := fake.NewSimpleClientset()
	selector := NewSelector(clientset)

	Context("GetPods", Ordered, func() {
		BeforeAll(func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx",
					Labels: map[string]string{
						"env": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}

			_, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})

		It("should get pods with label env=test", func() {
			labelMap := map[string]string{
				"env": "test",
			}
			pods, err := selector.GetPods("default", &labelMap, nil)

			Expect(err).To(BeNil())
			Expect(len(pods)).To(Equal(1))
			Expect(pods[0].Labels).To(Equal(labelMap))
		})

		It("should get pods with field status.phase=Running", func() {
			fieldMap := map[string]string{
				"status.phase": "Running",
			}
			pods, err := selector.GetPods("default", nil, &fieldMap)

			Expect(err).To(BeNil())
			Expect(len(pods)).To(Equal(1))
			Expect(pods[0].Status.Phase).To(Equal(corev1.PodRunning))
		})

		It("should get pods with both label and field selectors", func() {
			labelMap := map[string]string{
				"env": "test",
			}
			fieldMap := map[string]string{
				"status.phase": "Running",
			}
			pods, err := selector.GetPods("default", &labelMap, &fieldMap)

			Expect(err).To(BeNil())
			Expect(len(pods)).To(Equal(1))
			Expect(pods[0].Labels).To(Equal(labelMap))
			Expect(pods[0].Status.Phase).To(Equal(corev1.PodRunning))
		})
	})
})
