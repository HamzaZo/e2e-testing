package e2e

import (
	"context"
	bacth "k8s.io/api/batch/v1"
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

func UpdateEventStatus(api *ApiClient, namespace, jobName string) error {

	j, err := api.ClientSet.BatchV1().Jobs(namespace).Get(context.Background(), jobName, metaV1.GetOptions{})
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	_ = bacth.AddToScheme(scheme)

	klog.Info("Updating e2e pod event..")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(4)
	eventBroadcaster.StartRecordingToSink(&typedv1core.EventSinkImpl{Interface: api.ClientSet.CoreV1().Events("")})
	eventRecorder := eventBroadcaster.NewRecorder(scheme, coreV1.EventSource{})
	eventRecorder.Event(j, eventType, reason, message)
	eventBroadcaster.Shutdown()
	time.Sleep(2 * time.Second)

	return nil
}
