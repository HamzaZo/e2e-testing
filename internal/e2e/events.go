package e2e

import (
	"context"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	typedv1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"time"
)

const (
	message   = "Successfully running e2e testing against cluster"
	reason    = "E2E testing"
	eventType = coreV1.EventTypeNormal
)

func UpdateEventStatus(api *ApiClient, namespace, podName string) error {

	p, err := api.ClientSet.CoreV1().Pods(namespace).Get(context.Background(), podName, metaV1.GetOptions{})
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	_ = coreV1.AddToScheme(scheme)

	klog.Info("Updating e2e pod event..")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(4)
	eventBroadcaster.StartRecordingToSink(&typedv1core.EventSinkImpl{Interface: api.ClientSet.CoreV1().Events("")})
	eventRecorder := eventBroadcaster.NewRecorder(scheme, coreV1.EventSource{})
	eventRecorder.Event(p, eventType, reason, message)
	eventBroadcaster.Shutdown()
	time.Sleep(2 * time.Second)

	return nil
}
