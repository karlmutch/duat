package kubernetes

import (
	"context"
	"time"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (task *Task) createNamespace(ns string, overwrite bool, logger chan *Status) (err kv.Error) {
	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		}}

	if _, errGo := Client().CoreV1().Namespaces().Create(nsSpec); errGo != nil {
		if !errors.IsAlreadyExists(errGo) || !overwrite {
			task.failed = kv.Wrap(errGo).With("namespace", ns, "stack", stack.Trace().TrimRuntime())
			return task.failed
		}
	}
	return nil
}

func (task *Task) deleteNamespace(ctx context.Context, ns string, logger chan *Status) (err kv.Error) {

	deletePolicy := metav1.DeletePropagationForeground
	opts := &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	api := Client().CoreV1().Namespaces()
	errGo := api.Delete(ns, opts)
	if errGo != nil {
		task.failed = kv.Wrap(errGo).With("namespace", ns, "stack", stack.Trace().TrimRuntime())
		return task.failed
	}

	if _, ok := ctx.Deadline(); !ok {
		// No timeout or deadline was set so simply return with no error
		return nil
	}

	for {
		if _, errGo = api.Get(ns, metav1.GetOptions{}); errGo != nil {
			if errors.IsNotFound(errGo) {
				return nil
			}

			task.failed = kv.Wrap(errGo).With("namespace", ns, "stack", stack.Trace().TrimRuntime())
			return task.failed
		}

		select {
		case <-ctx.Done():
			return kv.NewError("could not verify the the namespace was deleted").With("namespace", ns, "stack", stack.Trace().TrimRuntime())
		case <-time.After(time.Second):
		}
	}
}
