package admin

import (
	"net/http"

	"github.com/hstreamdb/hstream-operator/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var ac *adminClient

	BeforeEach(func() {
		ac = &adminClient{
			remoteExec: &mockExecutor{},
			hdb:        mock.CreateDefaultCR(),
		}
	})

	Context("GetHMetaStatus", func() {
		It("should return expected status", func() {
			ac.remoteExec.(*mockExecutor).getAPIByService = func(namespace, serviceName, path string) ([]byte, int, error) {
				nodes := map[string]HMetaNode{
					"node1": {
						Reachable: true,
					},
					"node2": {
						Reachable: true,
					},
				}
				resp, _ := json.Marshal(nodes)

				return resp, http.StatusOK, nil
			}

			status, err := ac.GetHMetaStatus()

			Expect(err).NotTo(HaveOccurred())
			for _, node := range status.Nodes {
				Expect(node.Reachable).To(Equal(true))
			}
		})
	})
})
