package kubernetes

import (
	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
)

func (task *Task) initSecrets(logger chan *Status) (err kv.Error) {
	api := Client().CoreV1()
	for _, secret := range task.start.SecretSpecs {
		if _, errGo := api.Secrets(task.start.Namespace).Create(secret); errGo != nil {
			task.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
			return task.failed
		}
	}
	return nil
}
