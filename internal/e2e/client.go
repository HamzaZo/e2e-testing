package e2e

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ApiClient struct {
	ClientSet kubernetes.Interface
}

func NewKubeClient() (*ApiClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &ApiClient{
		ClientSet: clSet,
	}, nil

}
