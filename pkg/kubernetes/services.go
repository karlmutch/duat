package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-stack/stack"
	"github.com/jjeffery/kv"
)

func (task *Task) initServices(ctx context.Context, logger chan *Status) (err kv.Error) {
	api := Client().CoreV1()
	for _, service := range task.start.ServiceSpecs {
		if _, errGo := api.Services(task.start.Namespace).Create(ctx, service, metav1.CreateOptions{}); errGo != nil {
			task.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
			return task.failed
		}
	}
	return nil
}
