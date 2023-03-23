package e2e

import (
	"context"
	"e2e-k8s/internal/utils"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"net/http"
	"strings"
	"time"
)

type FlowTest struct {
	Registry string
	Host     string
}

const (
	flowOpen        = "Flow is open"
	flowClose       = "Flow is not open"
	flowTesterImage = "nicolaka/netshoot:v0.9"
	demoK8sImage    = "paulbouwer/hello-kubernetes:1.10"
	serviceName     = "demo-k8s"
	DemoNamespace   = "eph-demo-app"
	NetNamespace    = "eph-demo-net"
)

var (
	port int32 = 8080
)

func init() {
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "true")
}

func (f FlowTest) CreateResources(api *ApiClient) error {
	klog.Info("Creating demo namespaces..")
	err := createNamespace(api)
	if err != nil {
		klog.Errorf("failed to create namespace, error: %v", err)
	}
	klog.Info("Creating and expose k8s-demo..")
	err = createDeployment(DemoNamespace, f.Registry, api)
	if err != nil {
		klog.Errorf("failed to create deployment error: %v", err)
	}

	klog.Info("Creating ingress for k8s-demo..")
	err = createIngress(DemoNamespace, f.Host, api)
	if err != nil {
		klog.Errorf("failed to create ingress error: %v", err)
	}
	klog.Info("Creating network pod and run flow test..")
	err = createPodAndRunFlow(NetNamespace, f.Registry, api)
	if err != nil {
		klog.Errorf("failed to create flow tester pod error: %v", err)
	}

	return nil
}

func createNamespace(api *ApiClient) error {
	ns := []string{DemoNamespace, NetNamespace}
	for _, item := range ns {
		n := coreV1.Namespace{
			ObjectMeta: metaV1.ObjectMeta{
				Name: item,
				Labels: map[string]string{
					"e2e": "true",
				},
			},
		}

		_, err := api.ClientSet.CoreV1().Namespaces().Create(context.TODO(), &n, metaV1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func createDeployment(namespace, registry string, api *ApiClient) error {
	d := appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "demo-k8s",
			Labels: map[string]string{
				"e2e": "demo-k8s",
			},
			Namespace: namespace,
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: utils.Int32Ptr(1),
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					"e2e": "demo-k8s",
				},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"e2e": "demo-k8s",
					},
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:            "demo-k8s",
							Image:           fmt.Sprintf("%v/%v", registry, demoK8sImage),
							ImagePullPolicy: coreV1.PullIfNotPresent,
							Ports: []coreV1.ContainerPort{
								{
									ContainerPort: port,
									Protocol:      "TCP",
								},
							},
							Resources: coreV1.ResourceRequirements{
								Limits: map[coreV1.ResourceName]resource.Quantity{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("128Mi"),
								},
								Requests: map[coreV1.ResourceName]resource.Quantity{
									"cpu":    resource.MustParse("60m"),
									"memory": resource.MustParse("64Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := api.ClientSet.AppsV1().Deployments(namespace).Create(context.TODO(), &d, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	err = waitForPodReady(api, DemoNamespace, "demo-k8s")
	if err != nil {
		return err
	}

	err = createSvc(DemoNamespace, api)
	if err != nil {
		klog.Errorf("failed to create service error: %v", err)
	}

	return nil
}

func waitForPodReady(api *ApiClient, namespace, label string) error {
	for {
		pod, err := getPods(api, namespace, label)
		if err != nil {
			return errors.Wrap(err, "Error listing pods")
		}

		if len(pod.Items) == 0 {
			// No pods found, retry
			time.Sleep(20 * time.Second) // wait for 5 seconds before retrying
			continue
		}

		for _, pod := range pod.Items {
			if pod.Status.Phase == coreV1.PodRunning {
				// Pod is ready, no need to retry
				klog.Infof("pod %q is running", pod.Name)
				return nil
			}
		}

		// Pods exist but none are ready, retry
		time.Sleep(5 * time.Second) // wait for 5 seconds before retrying
	}
}

func createSvc(namespace string, api *ApiClient) error {
	s := coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name: serviceName,
			Labels: map[string]string{
				"e2e": "demo-k8s",
			},
			Namespace: namespace,
		},
		Spec: coreV1.ServiceSpec{
			Ports: []coreV1.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       port,
					TargetPort: intstr.IntOrString{IntVal: port},
				},
			},
			Selector: map[string]string{
				"e2e": "demo-k8s",
			},
		},
	}

	_, err := api.ClientSet.CoreV1().Services(namespace).Create(context.TODO(), &s, metaV1.CreateOptions{})
	if err != nil {
		klog.Errorf("failed to create service %q", serviceName)
		return err
	}
	return nil
}

func createIngress(namespace, host string, api *ApiClient) error {
	iPath := networkingv1.PathType("Exact")
	i := networkingv1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "demo-k8s-ingress",
			Labels: map[string]string{
				"e2e": "demo-k8s",
			},
			Namespace: namespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/demo-k8s",
									PathType: &iPath,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := api.ClientSet.NetworkingV1().Ingresses(namespace).Create(context.TODO(), &i, metaV1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func createPodAndRunFlow(namespace, registry string, api *ApiClient) error {
	podEndpoint, err := getPodIP(api, DemoNamespace, fmt.Sprintf("demo-k8s"))
	if err != nil {
		return err
	}
	ncCommand := fmt.Sprintf("nc -z %s %v", podEndpoint, port)
	p := coreV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "flowtester",
			Labels: map[string]string{
				"e2e": "flow-tester",
			},
			Namespace: namespace,
		},
		Spec: coreV1.PodSpec{
			Containers: []coreV1.Container{
				{
					Name:    "flowtester",
					Image:   fmt.Sprintf("%v/%v", registry, flowTesterImage),
					Command: []string{"/bin/sh"},
					Args: []string{
						"-c", fmt.Sprintf("trap 'exit 0' SIGTERM; while true; do exec timeout 1 %s & wait $! && echo '%s' || echo '%s'; done", ncCommand, flowOpen, flowClose),
					},
					Resources: coreV1.ResourceRequirements{
						Limits: map[coreV1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("128Mi"),
						},
						Requests: map[coreV1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("60m"),
							"memory": resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	_, err = api.ClientSet.CoreV1().Pods(namespace).Create(context.TODO(), &p, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	err = waitForPodReady(api, NetNamespace, "flow-tester")
	if err != nil {
		return err
	}

	return nil
}

func getPodIP(api *ApiClient, namespace, label string) (string, error) {
	var podIP string

	pod, err := getPods(api, namespace, label)
	if err != nil {
		return "", err
	}
	for _, k := range pod.Items {
		podIP = k.Status.PodIP
	}

	return podIP, nil
}

func getPods(api *ApiClient, namespace, label string) (*coreV1.PodList, error) {
	pods, err := api.ClientSet.CoreV1().Pods(namespace).List(context.Background(), metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("e2e=%s", label),
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func getPodName(api *ApiClient, namespace, label string) (string, error) {
	var podName string
	pod, err := getPods(api, namespace, label)
	if err != nil {
		return "", err
	}

	for _, k := range pod.Items {
		podName = k.Name
	}

	return podName, nil
}

func getPodLogs(api *ApiClient, namespace, label string) (string, error) {
	podName, err := getPodName(api, namespace, label)
	if err != nil {
		return "", err
	}
	req := api.ClientSet.CoreV1().Pods(namespace).GetLogs(podName, &coreV1.PodLogOptions{Follow: true})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	var logs strings.Builder
	// Wait for "flowopen" to appear in logs
	for {
		buf := make([]byte, 1024)
		numBytes, readErr := podLogs.Read(buf)
		if numBytes > 0 {
			logs.Write(buf[:numBytes])
			if strings.Contains(string(buf[:numBytes]), flowOpen) {
				return logs.String(), nil
			}
		}
		if readErr != nil {
			return "", err
		}
	}
}

func (f FlowTest) ValidateFlowE2e(api *ApiClient, host string, e2eNamespace, e2ePodName string) error {
	flow, err := CheckNetworkFlow(api)
	if err != nil {
		return err
	}
	if flow {
		klog.Infof("Flow is Open on port %v from namespace %s to namespace %s",
			port, NetNamespace, DemoNamespace)
		err = CheckIngressFlow(host)
		if err != nil {
			return err
		}
		err = UpdateEventStatus(api, e2eNamespace, e2ePodName)
		if err != nil {
			return err
		}
	}

	return nil
}

func CheckNetworkFlow(api *ApiClient) (bool, error) {
	p, err := getPodLogs(api, NetNamespace, fmt.Sprintf("flow-tester"))
	if err != nil {
		klog.Errorf("failed to retrieve logs: %v", err)
		return false, err
	}
	if strings.Contains(p, flowOpen) {
		return true, nil
	}

	return false, nil
}

func CheckIngressFlow(host string) error {
	resp, err := http.Get(fmt.Sprintf("%v/demo-k8s", host))
	if err != nil {
		klog.Errorf("failed to send HTTP request to ingress")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("expected HTTP status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	return nil
}
