// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package k8s

import (
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	peerdDefaultNamespace = "peerd-ns"
)

// ClientSet is an interface for k8s API server.
type ClientSet struct {
	kubernetes.Interface

	// InPod indicates whether the current runtime environment is a pod.
	InPod bool

	// Namespace is the namespace in which to run the leader election.
	Namespace string

	// Name is the name of this pod or node.
	Name string
}

// NewKubernetesInterface creates a new interface for k8s API server.
// The current runtime environment is first assumed to be a pod and its identity is used to create the interface.
// If a pod is not detected, the given kubeConfigPath is used to create the interface.
func NewKubernetesInterface(kubeConfigPath, name string) (*ClientSet, error) {
	k := &ClientSet{Name: name}

	config, err := rest.InClusterConfig() // Assume run in a Pod or an environment with appropriate env variables set.
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, err
		}
		k.InPod = false
		k.Namespace = peerdDefaultNamespace
	} else {
		k.InPod = true
		k.Namespace = getPodNamespace()
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k.Interface = clientset

	return k, nil
}

// getPodNamespace returns the namespace in which the pod is running or the default namespace.
// Ref: https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/
func getPodNamespace() string {
	namespace := os.Getenv("NAMESPACE")
	if namespace != "" {
		return namespace
	}

	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		return string(namespaceBytes)
	}

	return peerdDefaultNamespace
}
