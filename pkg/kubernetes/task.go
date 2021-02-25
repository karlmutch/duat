package kubernetes

import (
	"context"
	"time"

	"github.com/jjeffery/kv"
	logxi "github.com/karlmutch/logxi/v1"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
)

// TaskSpec encapsulates the information used when initating the bootstrapping
// pods and jobs involved in generation of images etc
//
type TaskSpec struct {
	Namespace    string // The Kubernetes namespace being used for running the CI bootstrapping
	ID           string
	Dir          string
	Dockerfile   string
	Env          map[string]string
	JobSpec      *batchv1.Job
	SecretSpecs  []*v1.Secret
	ServiceSpecs []*v1.Service
}

// Task encapsulates the entire context of a Kubernetes batch job/pod, including the
// persistent volume that is being used to transport the state related to the pipeline
// actions being undertaken, for example a git cloned repository.
//
type Task struct {
	start  TaskSpec
	failed kv.Error
	volume string
}

func (task *Task) initialize(ctx context.Context, debugMode bool, logger chan *Status) (err kv.Error) {

	if initFailure != nil {
		task.sendStatus(ctx, logger, logxi.LevelFatal, initFailure)
		return initFailure
	}

	func() {
		deleteCtx, deleteCancel := context.WithTimeout(ctx, 60*time.Second)
		defer deleteCancel()
		if err = task.deleteNamespace(deleteCtx, task.start.Namespace, logger); err != nil {
			if err != nil {
				task.sendStatus(ctx, logger, logxi.LevelInfo, err)
			}
		}
	}()

	if err = task.createNamespace(task.start.Namespace, true, logger); err != nil {
		return err
	}

	// Populate secrets, report only the first failure if any occur
	if errs := task.initSecrets(logger); len(errs) != 0 {
		return errs[0]
	}

	// Populate services, these are often portals outside of our namespace and allow
	// interaction with external caches etc
	if err = task.initServices(logger); err != nil {
		return err
	}

	// Start the templated deployment and allow it to create its own container
	if err = task.runJob(ctx, logger); err != nil {
		return err
	}

	return nil
}

// TasksRunner will listen for changes to a git repository and trigger downstream tasks that will process and consume
// the change
//
func TasksRunner(ctx context.Context, debugMode bool, triggerC chan *TaskSpec, statusC chan *Status) {
	defer close(statusC)
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-triggerC:
			task := &Task{
				start: *msg,
			}

			task.sendStatus(ctx, statusC, logxi.LevelInfo, kv.NewError("change detected").With("dir", msg.Dir))

			if err := task.initialize(ctx, debugMode, statusC); err != nil {
				task.sendStatus(ctx, statusC, logxi.LevelFatal, err)
				continue
			}
		}
	}
}

// TasksStart is used to initate a git change notification processing go routine
//
func TasksStart(ctx context.Context, debugMode bool, triggerC chan *TaskSpec, statusC chan *Status) {
	go TasksRunner(ctx, debugMode, triggerC, statusC)
}
