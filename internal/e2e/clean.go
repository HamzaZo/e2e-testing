package e2e

import (
	"context"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func Cleaning(api *ApiClient) error {
	klog.Info("Cleaning deployment..")

	if err := api.ClientSet.AppsV1().Deployments(DemoNamespace).Delete(context.Background(), "demo-k8s",
		metaV1.DeleteOptions{}); err != nil {
		return err
	}
	klog.Info("Cleaning service..")

	if err := api.ClientSet.CoreV1().Services(DemoNamespace).Delete(context.Background(), "demo-k8s",
		metaV1.DeleteOptions{}); err != nil {
		return err
	}
	klog.Info("Cleaning ingress..")

	if err := api.ClientSet.NetworkingV1().Ingresses(DemoNamespace).Delete(context.Background(), "demo-k8s-ingress",
		metaV1.DeleteOptions{}); err != nil {
		return err
	}
	klog.Info("Cleaning flow tester pod..")
	if err := api.ClientSet.CoreV1().Pods(NetNamespace).Delete(context.Background(), "flowtester",
		metaV1.DeleteOptions{}); err != nil {
		return err
	}

	klog.Info("Cleaning namespaces..")
	if err := api.ClientSet.CoreV1().Namespaces().Delete(context.Background(), DemoNamespace,
		metaV1.DeleteOptions{}); err != nil {
		return err
	}
	if err := api.ClientSet.CoreV1().Namespaces().Delete(context.Background(), NetNamespace,
		metaV1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}
