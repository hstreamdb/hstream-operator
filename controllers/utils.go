package controllers

import (
	"fmt"
	hapi "github.com/hstreamdb/hstream-operator/api/v1alpha2"
	"github.com/hstreamdb/hstream-operator/internal"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"strconv"
	"strings"
)

// structAssign copy the value of struct from src to dist
func structAssign(dist interface{}, src interface{}) {
	dVal := reflect.ValueOf(dist).Elem()
	sVal := reflect.ValueOf(src).Elem()
	sType := sVal.Type()
	for i := 0; i < sVal.NumField(); i++ {
		if sVal.Field(i).IsZero() {
			continue
		}

		// we need to check if the dist struct has the same field
		name := sType.Field(i).Name
		dvField := dVal.FieldByName(name)
		if ok := dvField.IsValid(); !ok {
			continue
		}

		dvField.Set(reflect.ValueOf(sVal.Field(i).Interface()))
	}
}

// mergeLabels merges the labels specified by the operator into
// on object's metadata.
//
// This will return whether the target's labels have changed.
func mergeLabelsInMetadata(target *metav1.ObjectMeta, desired metav1.ObjectMeta) bool {
	return mergeMap(target.Labels, desired.Labels)
}

// mergeAnnotations merges the annotations specified by the operator into
// on object's metadata.
//
// This will return whether the target's annotations have changed.
func mergeAnnotations(target *metav1.ObjectMeta, desired metav1.ObjectMeta) bool {
	return mergeMap(target.Annotations, desired.Annotations)
}

// mergeMap merges a map into another map.
//
// This will return whether the target's values have changed.
func mergeMap(target map[string]string, desired map[string]string) bool {
	changed := false
	for key, value := range desired {
		if target[key] != value {
			target[key] = value
			changed = true
		}
	}
	return changed
}

// extendEnv adds environment variables to an existing environment, unless
// environment variables with the same name are already present.
func extendEnv(container *corev1.Container, env []corev1.EnvVar) {
	existingVars := make(map[string]bool, len(container.Env))

	for _, envVar := range container.Env {
		existingVars[envVar.Name] = true
	}

	for _, envVar := range env {
		if !existingVars[envVar.Name] {
			container.Env = append(container.Env, envVar)
		}
	}
}

func extendArg(container *corev1.Container, defaultArgs map[string]string) (args map[string]string, err error) {
	flags := internal.FlagSet{}
	if err = flags.Parse(container.Args); err != nil {
		return
	}

	// flag in the args doesn't contain prefix '-' or '--'
	args = flags.Flags()
	for flag, value := range defaultArgs {
		// we need to cut the prefix '-' or '--' before comparing with existingVars
		flag = strings.TrimLeft(flag, "-")
		if _, ok := args[flag]; !ok {
			args[flag] = value
		}
	}

	container.Args = make([]string, 0, len(args)*2)
	// sort the arg list
	flags.Visit(func(flag, value string) {
		container.Args = append(container.Args, "--"+flag)
		if value != "" {
			container.Args = append(container.Args, value)
		}
	})
	return
}

func extendPorts(args map[string]string, userDefinedPorts, defaultPorts []corev1.ContainerPort) []corev1.ContainerPort {
	// copy default ports and cover the containerPort with user-defined port arg
	required := coverPorts(args, defaultPorts)
	// merge user-defined ports to required
	return mergePorts(required, userDefinedPorts)
}

// coverPorts use the port in user-defined args to cover the default port
func coverPorts(args map[string]string, required []corev1.ContainerPort) []corev1.ContainerPort {
	ports := make([]corev1.ContainerPort, len(required))
	copy(ports, required)

	for i := range required {
		if port, ok := args[(&required[i]).Name]; ok {
			ports[i].ContainerPort = intstr.Parse(port).IntVal
		}
	}
	return ports
}

// mergePorts merge the same name of user defined port to required port
func mergePorts(required, userDefined []corev1.ContainerPort) []corev1.ContainerPort {
	ports := make([]corev1.ContainerPort, len(required))
	copy(ports, required)

	for i := range userDefined {
		found := false
		for j := range ports {
			if (&ports[j]).Name == (&userDefined[i]).Name {
				found = true
				break
			}
		}
		if !found {
			if (&userDefined[i]).Name == "" {
				(&userDefined[i]).Name = fmt.Sprintf("unset-%d", (&userDefined[i]).ContainerPort)
			}
			ports = append(ports, userDefined[i])
		}
	}
	return ports
}

func isHashChanged(obj1, obj2 *metav1.ObjectMeta) bool {
	return obj1.Annotations[hapi.LastSpecKey] != obj2.Annotations[hapi.LastSpecKey]
}

func getHMetaAddr(hdb *hapi.HStreamDB) (string, error) {
	hmetaAddr := ""
	if hdb.Spec.ExternalHMeta != nil {
		hmetaAddr = hdb.Spec.ExternalHMeta.GetAddr()
	} else {
		svc := internal.GetService(hdb, hapi.ComponentTypeHMeta)
		flags := internal.FlagSet{}
		if err := flags.Parse(hdb.Spec.HMeta.Container.Args); err != nil {
			err = fmt.Errorf("parse hmeta args failed. %w", err)
			return "", err
		}
		parsedArgs := flags.Flags()
		port, ok := parseHMetaPort(parsedArgs)
		if !ok {
			port = strconv.Itoa(int(hmetaPorts[0].ContainerPort))
		}
		hmetaAddr = svc.Name + "." + svc.Namespace + ":" + port
	}
	return hmetaAddr, nil
}

func getHMetaContainerPorts(container *hapi.Container, parsedArgs map[string]string) (ports []corev1.ContainerPort) {
	if hmetaPort, ok := parseHMetaPort(parsedArgs); ok {
		return extendPorts(map[string]string{
			"port": hmetaPort,
		}, container.Ports, hmetaPorts)
	}
	return mergePorts(hmetaPorts, container.Ports)
}

func parseHMetaPort(parsedArgs map[string]string) (port string, ok bool) {
	addr := parsedArgs["http-addr"]
	if addr == "" {
		return
	}

	if slice := strings.Split(addr, ":"); len(slice) == 2 {
		return slice[1], true
	}
	return
}
