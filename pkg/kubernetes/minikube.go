package kubernetes

// This file contains functions of use when using minikube
import (
	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsMinikube  will test the operating environment of the present process to determine
// if it is running within a minikube provisioned cluster
//
func IsMinikube() (isMinikube bool, err kv.Error) {
	selector := "kubernetes.io/hostname=minikube"
	nodes, errGo := Client().CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector})
	if errGo != nil {
		return false, kv.Wrap(errGo).With("selector", selector, "stack", stack.Trace().TrimRuntime())
	}
	if len(nodes.Items) > 0 {
		return true, nil
	}
	return false, nil
}
