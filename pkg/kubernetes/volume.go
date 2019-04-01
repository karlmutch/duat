package kubernetes

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jjeffery/kv"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/davecgh/go-spew/spew"
	"github.com/karlmutch/stack"
	"github.com/mgutz/logxi"
)

func (job *Task) initVolume(logger chan *Status) (err kv.Error) {

	volume, errGo := uuid.NewRandom()
	if errGo != nil {
		job.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	job.volume = volume.String()

	fs := v1.PersistentVolumeFilesystem
	createOpts := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.volume,
			Namespace: job.start.Namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse("10Gi"),
				},
			},
			VolumeMode: &fs,
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase:       v1.ClaimBound,
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): resource.MustParse("10Gi"),
			},
		},
	}

	api := Client().CoreV1()
	if _, errGo = api.PersistentVolumeClaims(job.start.Namespace).Create(createOpts); errGo != nil {
		job.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	return nil
}

func (job *Task) waitOnVolume(ctx context.Context, logger chan *Status) (err kv.Error) {
	api := Client().CoreV1()
	watcher, errGo := api.PersistentVolumeClaims(job.start.Namespace).Watch(metav1.ListOptions{})
	if errGo != nil {
		job.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			pvc, ok := event.Object.(*v1.PersistentVolumeClaim)
			if !ok {
				continue
			}

			if pvc.ObjectMeta.Namespace != job.start.Namespace || pvc.ObjectMeta.Name != job.volume {
				continue
			}

			if state, isPresent := pvc.ObjectMeta.Annotations["pv.kubernetes.io/bind-completed"]; isPresent == false || state != "yes" {
				continue
			}
			if pvc.Status.Phase != v1.ClaimBound {
				continue
			}

			statusCtx, statusCancel := context.WithTimeout(ctx, time.Duration(20*time.Millisecond))
			job.sendStatus(statusCtx, logger, logxi.LevelInfo, kv.NewError("volume update").With("namespace", job.start.Namespace, "volume", pvc.ObjectMeta.Name, "phase", spew.Sdump(pvc.Status.Phase)))
			statusCancel()

			return nil
		case <-ctx.Done():
			return kv.NewError("timeout waiting for state to become 'bound'").With("namespace", job.start.Namespace, "volume", job.volume, "stack", stack.Trace().TrimRuntime())
		}
	}
}
