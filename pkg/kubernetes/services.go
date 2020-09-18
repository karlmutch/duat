package kubernetes

import (
	"github.com/jjeffery/kv"
	"github.com/go-stack/stack"
)

func (task *Task) initServices(logger chan *Status) (err kv.Error) {
	api := Client().CoreV1()
	for _, service := range task.start.ServiceSpecs {
		if _, errGo := api.Services(task.start.Namespace).Create(service); errGo != nil {
			task.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
			return task.failed
		}
	}
	return nil
}
