package kubernetes

import (
	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (job *Job) Namespaces() (namespaces map[string]string, err kv.Error) {
	namespaces = map[string]string{}
	return namespaces, nil
}

func (job *Job) createNamespace(ns string, overwrite bool, logger chan *Status) (err kv.Error) {
	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}

	if _, errGo := Client().CoreV1().Namespaces().Create(nsSpec); errGo != nil {
		if !errors.IsAlreadyExists(errGo) || !overwrite {
			job.failed = kv.Wrap(errGo).With("namespace", ns, "stack", stack.Trace().TrimRuntime())
			return job.failed
		}
	}
	return nil
}
