package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
	"github.com/mgutz/logxi"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file contains the implementation of functions used to start and run
// task and perform other various tasks within a Kubernetes cluster

type Status struct {
	ID    string
	Level int
	Msg   kv.Error
}

func (task *Task) sendStatus(ctx context.Context, statusC chan *Status, level int, msg kv.Error) {
	select {
	case statusC <- &Status{ID: task.start.ID, Level: level, Msg: msg}:
		return
	case <-time.After(20 * time.Millisecond):
	}
	fmt.Println("ID", task.start.ID, spew.Sdump(msg))
}

func (task *Task) runJob(ctx context.Context, logger chan *Status) (err kv.Error) {

	// Add a uuid field to the Job so that we can watch just this single job
	label := uuid.New().String()
	labelK := "duat.uuid"

	specLabels := task.start.JobSpec.GetLabels()
	if specLabels == nil {
		labels := map[string]string{"duat.uuid": label}
		task.start.JobSpec.SetLabels(labels)
	} else {
		specLabels[labelK] = label
	}

	tmpLabels := task.start.JobSpec.Spec.Template.GetLabels()
	if tmpLabels == nil {
		labels := map[string]string{labelK: label}
		task.start.JobSpec.Spec.Template.SetLabels(labels)
	} else {
		tmpLabels[labelK] = label
	}

	api := Client().BatchV1().Jobs(task.start.Namespace)

	_, errGo := api.Create(task.start.JobSpec)
	if errGo != nil {
		return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	watch, errGo := Client().CoreV1().Pods(task.start.Namespace).Watch(metav1.ListOptions{LabelSelector: labelK + "=" + label})
	if errGo != nil {
		return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	lastPhase := ""
	for event := range watch.ResultChan() {
		p, ok := event.Object.(*v1.Pod)
		if !ok {
			continue
		}
		if lastPhase != string(p.Status.Phase) {
			lastPhase = string(p.Status.Phase)
			task.sendStatus(ctx, logger, logxi.LevelInfo, kv.NewError("pod update").With("id", task.start.ID, "namespace", task.start.Namespace, "phase", lastPhase))
		}
		if p.Status.Phase == v1.PodSucceeded || p.Status.Phase == v1.PodFailed {
			break
		}
	}
	return nil
}
