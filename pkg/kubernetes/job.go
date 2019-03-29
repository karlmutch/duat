package kubernetes

import (
	"context"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jjeffery/kv"
	"github.com/mgutz/logxi"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file contains the implementation of functions used to start and run
// jobs and perform other various tasks within a Kubernetes cluster

type StartJob struct {
	Namespace  string
	ID         string
	Dir        string
	Dockerfile string
	Env        map[string]string
}

type Status struct {
	ID    string
	Level int
	Msg   kv.Error
}

type Job struct {
	start  StartJob
	failed kv.Error
	volume string
}

func (job *Job) sendStatus(ctx context.Context, statusC chan *Status, level int, msg kv.Error) {
	select {
	case statusC <- &Status{ID: job.start.ID, Level: level, Msg: msg}:
		return
	case <-time.After(20 * time.Millisecond):
	}
	fmt.Println("ID", job.start.ID, spew.Sdump(msg))
}

func (job *Job) initialize(ctx context.Context, logger chan *Status) (err kv.Error) {

	if err = job.createNamespace(job.start.Namespace, true, logger); err != nil {
		return err
	}

	// Create a persistent volume claim
	if err = job.initVolume(logger); err != nil {
		return err
	}

	// Wait for Bound state ifor the volume we just created or ctx.Done()
	//
	if err = job.waitOnVolume(ctx, logger); err != nil {
		return err
	}

	// Start a pod and mount the freshly created volume
	podName := "copy-pod"
	if err = job.startMinimalPod(ctx, podName, job.volume, logger); err != nil {
		return err
	}

	// Copy the cloned github repo into using a mount for the persistent volume
	if err = job.filePod(ctx, podName, false, "/tmp/karl", "/data/karl", logger); err != nil {
		return err
	}

	// Get rid of the temporary pod
	//if err = job.stopPod(ctx, podName, logger); err != nil {
	//	return err
	//}

	// Start the templated deployment and allow it to create its own container
	return nil
}

func (job *Job) runWatchedJob(ctx context.Context, statusC chan *Status) {
	statusCtx, statusCancel := context.WithTimeout(ctx, time.Duration(20*time.Millisecond))
	defer statusCancel()

	if initFailure != nil {
		job.sendStatus(statusCtx, statusC, logxi.LevelFatal, initFailure)
		return
	}

	job.sendStatus(statusCtx, statusC, logxi.LevelInfo, kv.NewError("running").With("id", job.start.ID, "namespace", job.start.Namespace, "dir", job.start.Dir))

	// List pods for validation
	api := Client().CoreV1()
	podList, errGo := api.Pods(job.start.Namespace).List(metav1.ListOptions{})
	if errGo != nil {
		job.sendStatus(statusCtx, statusC, logxi.LevelFatal, kv.Wrap(errGo).With("msg", spew.Sdump(job), "namespace", job.start.Namespace, "dir", job.start.Dir))
		return
	}

	volList, errGo := api.PersistentVolumeClaims(job.start.Namespace).List(metav1.ListOptions{})
	if errGo != nil {
		job.sendStatus(statusCtx, statusC, logxi.LevelFatal, kv.Wrap(errGo).With("msg", spew.Sdump(job), "namespace", job.start.Namespace, "dir", job.start.Dir))
		return
	}
	for _, v := range podList.Items {
		job.sendStatus(statusCtx, statusC, logxi.LevelInfo, kv.NewError("pod").With("namespace", job.start.Namespace, "node_name", v.Spec.NodeName, "pod_name", v.Name))
	}
	for _, v := range volList.Items {
		qty := v.Spec.Resources.Requests[v1.ResourceStorage]
		job.sendStatus(statusCtx, statusC, logxi.LevelInfo, kv.NewError("volume").With("namespace", job.start.Namespace, "volume_name", v.Name, "capacity", qty.String()))
	}

	job.sendStatus(statusCtx, statusC, logxi.LevelNotice, kv.NewError("success").With("msg", spew.Sdump(job), "namespace", job.start.Namespace, "dir", job.start.Dir))

}

func RunStarter(ctx context.Context, triggerC chan StartJob, statusC chan *Status) {
	defer close(statusC)
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-triggerC:
			job := &Job{
				start: msg,
			}

			if err := job.initialize(ctx, statusC); err != nil {
				job.sendStatus(ctx, statusC, logxi.LevelFatal, err)
				continue
			}
			go job.runWatchedJob(ctx, statusC)
		}
	}
}

func JobStarter(ctx context.Context, triggerC chan StartJob, statusC chan *Status) {
	go RunStarter(ctx, triggerC, statusC)
}
