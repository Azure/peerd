package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubernetesInterface creates a new interface for k8s API server.
// The current runtime environment is first assumed to be a pod and its identity is used to create the interface.
// If a pod is not detected, the given kubeConfigPath is used to create the interface.
func NewKubernetesInterface(kubeConfigPath string) (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig() // Assume run in a Pod or an environment with appropriate env variables set.
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
