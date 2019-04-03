package kubernetes

import (
	"context"
	"os"
	"time"

	"github.com/jjeffery/kv"
	"github.com/mgutz/logxi"

	"github.com/davecgh/go-spew/spew"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TaskSpec struct {
	Namespace  string
	ID         string
	Dir        string
	Dockerfile string
	Env        map[string]string
	JobSpec    *batchv1.Job
	SecretSpec *v1.Secret
}

type Task struct {
	start  TaskSpec
	failed kv.Error
	volume string
}

func (task *Task) initialize(ctx context.Context, logger chan *Status) (err kv.Error) {

	if initFailure != nil {
		task.sendStatus(ctx, logger, logxi.LevelFatal, initFailure)
		return initFailure
	}

	if err = task.createNamespace(task.start.Namespace, true, logger); err != nil {
		return err
	}

	// Populate secrets
	if err = task.initSecrets(logger); err != nil {
		return err
	}

	// Create a persistent volume claim
	if err = task.initVolume(logger); err != nil {
		return err
	}

	// Wait for Bound state ifor the volume we just created or ctx.Done()
	//
	if err = task.waitOnVolume(ctx, logger); err != nil {
		return err
	}

	// Create an archive containing the snapshot of the code to be ossified within a build
	// image
	archiveName := task.start.Dir + ".tar.gz"
	if err = task.createArchive(task.start.Dir, archiveName); err != nil {
		return err
	}
	defer os.Remove(archiveName)

	// Start a pod and mount the freshly created volume
	podName := "copy-pod"
	if err = task.startMinimalPod(ctx, podName, task.volume, logger); err != nil {
		return err
	}

	// Copy the cloned github repo into using a mount for the persistent volume
	if err = task.filePod(ctx, podName, "alpine", false, archiveName, "/data/tmp.gz", logger); err != nil {
		return err
	}

	os.Stdout.Sync()
	os.Stderr.Sync()

	if err = task.runInPod(ctx, podName, "alpine", []string{"tar", "-xf", "/data/tmp.gz", "-C", "/data"}, nil, os.Stdout, os.Stderr, logger); err != nil {
		return err
	}

	// Get rid of the temporary pod used for copying data
	if err = task.stopPod(ctx, podName, logger); err != nil {
		return err
	}

	// Start the templated deployment and allow it to create its own container
	if err = task.runJob(ctx, logger); err != nil {
		return err
	}

	if err = task.deleteNamespace(task.start.Namespace, logger); err != nil {
		return err
	}

	return nil
}

func (task *Task) runWatchedJob(ctx context.Context, statusC chan *Status) {
	statusCtx, statusCancel := context.WithTimeout(ctx, time.Duration(20*time.Millisecond))
	defer statusCancel()

	if initFailure != nil {
		task.sendStatus(statusCtx, statusC, logxi.LevelFatal, initFailure)
		return
	}

	task.sendStatus(statusCtx, statusC, logxi.LevelInfo, kv.NewError("running").With("id", task.start.ID, "namespace", task.start.Namespace, "dir", task.start.Dir))

	// List pods for validation
	api := Client().CoreV1()
	podList, errGo := api.Pods(task.start.Namespace).List(metav1.ListOptions{})
	if errGo != nil {
		task.sendStatus(statusCtx, statusC, logxi.LevelFatal, kv.Wrap(errGo).With("msg", spew.Sdump(task), "namespace", task.start.Namespace, "dir", task.start.Dir))
		return
	}

	volList, errGo := api.PersistentVolumeClaims(task.start.Namespace).List(metav1.ListOptions{})
	if errGo != nil {
		task.sendStatus(statusCtx, statusC, logxi.LevelFatal, kv.Wrap(errGo).With("msg", spew.Sdump(task), "namespace", task.start.Namespace, "dir", task.start.Dir))
		return
	}
	for _, v := range podList.Items {
		task.sendStatus(statusCtx, statusC, logxi.LevelInfo, kv.NewError("pod").With("namespace", task.start.Namespace, "node_name", v.Spec.NodeName, "pod_name", v.Name))
	}
	for _, v := range volList.Items {
		qty := v.Spec.Resources.Requests[v1.ResourceStorage]
		task.sendStatus(statusCtx, statusC, logxi.LevelInfo, kv.NewError("volume").With("namespace", task.start.Namespace, "volume_name", v.Name, "capacity", qty.String()))
	}

	task.sendStatus(statusCtx, statusC, logxi.LevelNotice, kv.NewError("success").With("msg", spew.Sdump(task), "namespace", task.start.Namespace, "dir", task.start.Dir))
}

func TasksRunner(ctx context.Context, triggerC chan *TaskSpec, statusC chan *Status) {
	defer close(statusC)
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-triggerC:
			task := &Task{
				start: *msg,
			}

			if err := task.initialize(ctx, statusC); err != nil {
				task.sendStatus(ctx, statusC, logxi.LevelFatal, err)
				continue
			}
			go task.runWatchedJob(ctx, statusC)
		}
	}
}

func TasksStart(ctx context.Context, triggerC chan *TaskSpec, statusC chan *Status) {
	go TasksRunner(ctx, triggerC, statusC)
}
