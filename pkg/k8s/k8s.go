// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientSet is an interface for k8s API server.
type ClientSet struct {
	kubernetes.Interface

	// InPod indicates whether the current runtime environment is a pod.
	InPod bool
}

// NewKubernetesInterface creates a new interface for k8s API server.
// The current runtime environment is first assumed to be a pod and its identity is used to create the interface.
// If a pod is not detected, the given kubeConfigPath is used to create the interface.
func NewKubernetesInterface(kubeConfigPath string) (*ClientSet, error) {
	k := &ClientSet{}

	config, err := rest.InClusterConfig() // Assume run in a Pod or an environment with appropriate env variables set.
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, err
		}
		k.InPod = false
	} else {
		k.InPod = true
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k.Interface = clientset

	return k, nil
}
