package kubernetes

import (
	"github.com/jjeffery/kv"
	"github.com/go-stack/stack"
)

// initSecrets will try and create all of the secrets that are available in the task structure.
// Any errors are reported however the create will continue regardless to try and create them all
//
func (task *Task) initSecrets(logger chan *Status) (errs []kv.Error) {
	errs = []kv.Error{}

	api := Client().CoreV1()
	for _, secret := range task.start.SecretSpecs {
		if _, errGo := api.Secrets(task.start.Namespace).Create(secret); errGo != nil {

			errs = append(errs, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime()))
		}
	}
	return errs
}
