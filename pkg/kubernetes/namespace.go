package kubernetes

import (
	"context"
	"strings"
	"time"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TrimNamespace can be used to trim strings that consist of words with dash '-' separators
// and a semver style format consisting of trailing branch names potentially and a unique ID
// on the very end to 64 characters, of significant information.  Examples of names might
// include 0.9.23-feature-265-zero-length-metadata-reinstated-aaaagmwypak
//
func TrimNamespace(ns string) (trimmed string) {
	return wordTrim(ns, "-", 64)
}

func wordTrim(input string, delimiter string, max int) (result string) {

	words := strings.Split(input, delimiter)

	if len(words) == 1 {
		if len(words[0]) > max {
			return words[0][:max]
		}
		return words[0]
	}

	// Make sure our first word plus the delimiter size does not flood the string, if so then only use that word
	if len(words[0])+len(delimiter)+len(words[1]) > max || len(words) < 2 {
		return words[0][:max]
	}

	output := []string{words[0], words[len(words)-1]}
	words = words[1 : len(words)-1]
	gatheredLen := len(output[0]) + len(delimiter) + len(output[1])

	for _, aWord := range words {
		gatheredLen += len(delimiter) + len(aWord)

		if gatheredLen > max {
			break
		}

		output = append(output[:len(output)-1], append([]string{aWord}, output[len(output)-1])...)

	}
	return strings.Join(output, delimiter)
}

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
