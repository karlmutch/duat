package kubernetes

// This file contains implementations of functions and recievers that
// are relevant to using microk8s deployments

import (
	"context"

	"github.com/go-stack/stack"
	"github.com/jjeffery/kv"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MicroK8s is used to encapsulate receivers for working with microk8s clusters
//
type MicroK8s struct{}

// GetRegistryPod is used to obtain pod information for the microk8s provisioned image registry.
//
func (*MicroK8s) GetRegistryPod(ctx context.Context) (pod *apiv1.Pod, err kv.Error) {

	namespace := "container-registry"
	api := Client().CoreV1().Pods(namespace)

	label := "app=registry"
	pods, errGo := api.List(ctx, metav1.ListOptions{LabelSelector: label})
	if errGo != nil {
		return nil, kv.Wrap(errGo).With("namespace", namespace, "stack", stack.Trace().TrimRuntime())
	}
	selectedPods := []*apiv1.Pod{}
	for _, aPod := range pods.Items {
		if aPod.Status.Phase != apiv1.PodRunning {
			continue
		}
		selectedPods = append(selectedPods, aPod.DeepCopy())
	}

	if len(selectedPods) > 1 {
		return nil, kv.NewError("too many unexpected pods inside the microk8s container-registry namespace").With("namespace", namespace, "stack", stack.Trace().TrimRuntime())
	}
	if len(selectedPods) < 1 {
		return nil, kv.NewError("microk8s container-registry namespace missing expected pod").With("namespace", namespace, "stack", stack.Trace().TrimRuntime())
	}

	return selectedPods[0], nil
}
