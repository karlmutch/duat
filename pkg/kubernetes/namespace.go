package kubernetes

import (
	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (job *Task) Namespaces() (namespaces map[string]string, err kv.Error) {
	namespaces = map[string]string{}
	return namespaces, nil
}

func (job *Task) createNamespace(ns string, overwrite bool, logger chan *Status) (err kv.Error) {
	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}

	if _, errGo := Client().CoreV1().Namespaces().Create(nsSpec); errGo != nil {
		if !errors.IsAlreadyExists(errGo) || !overwrite {
			job.failed = kv.Wrap(errGo).With("namespace", ns, "stack", stack.Trace().TrimRuntime())
			return job.failed
		}
	}
	return nil
}

func (job *Task) deleteNamespace(ns string, logger chan *Status) (err kv.Error) {
	if errGo := Client().CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{}); errGo != nil {
		job.failed = kv.Wrap(errGo).With("namespace", ns, "stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	return nil
}
