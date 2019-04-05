package kubernetes

import (
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

func (task *Task) deleteNamespace(ns string, logger chan *Status) (err kv.Error) {
	if errGo := Client().CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{}); errGo != nil {
		task.failed = kv.Wrap(errGo).With("namespace", ns, "stack", stack.Trace().TrimRuntime())
		return task.failed
	}
	return nil
}
