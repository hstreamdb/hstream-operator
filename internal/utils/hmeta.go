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

package utils

import (
	"fmt"
	"strings"

	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/pkg/args"
	"github.com/hstreamdb/hstream-operator/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GetHMetaAddr(hdb *hapi.HStreamDB) (string, error) {
	hmetaAddr := ""

	if hdb.Spec.ExternalHMeta != nil {
		hmetaAddr = hdb.Spec.ExternalHMeta.GetAddr()
	} else {
		svc := hapi.ComponentTypeHMeta.GetHeadlessService(hdb, nil)
		port, err := GetHMetaContainerPort(hdb.Spec.HMeta.Container.Args)
		if err != nil {
			return "", err
		}

		hmetaAddr = fmt.Sprintf("%s.%s:%d", svc.Name, svc.Namespace, port.ContainerPort)
	}

	return hmetaAddr, nil
}

func GetHMetaContainerPort(hmetaArgs []string) (corev1.ContainerPort, error) {
	parsedArgs := args.ParseArgs(hmetaArgs)

	if addr, ok := parsedArgs["--http-addr"]; ok {
		if slice := strings.Split(addr, ":"); len(slice) == 2 {
			return corev1.ContainerPort{
				Name:          "port",
				ContainerPort: intstr.Parse(slice[1]).IntVal,
			}, nil
		}
	}

	return constants.DefaultHMetaPort, nil
}
