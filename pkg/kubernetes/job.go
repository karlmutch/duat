package kubernetes

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"
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

func (job *Job) createArchive(src string, dst string) (err kv.Error) {

	file, errGo := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if errGo != nil {
		return kv.Wrap(errGo).With("fn", dst, "stack", stack.Trace().TrimRuntime())
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	errGo = filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring, that is
		// a relative location
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// return on non-regular files which adds the header but expects no content
		if !fi.Mode().IsRegular() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})
	if errGo != nil {
		return kv.Wrap(errGo).With("destination", dst, "stack", stack.Trace().TrimRuntime())
	}
	return nil
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

	// Create an archive containing the snapshot of the code to be ossified within a build
	// image
	archiveName := job.start.Dir + ".tar.gz"
	if err = job.createArchive(job.start.Dir, archiveName); err != nil {
		return err
	}
	defer os.Remove(archiveName)

	// Start a pod and mount the freshly created volume
	podName := "copy-pod"
	if err = job.startMinimalPod(ctx, podName, job.volume, logger); err != nil {
		return err
	}

	// Copy the cloned github repo into using a mount for the persistent volume
	if err = job.filePod(ctx, podName, "alpine", false, archiveName, "/data/tmp.gz", logger); err != nil {
		return err
	}

	os.Stdout.Sync()
	os.Stderr.Sync()
	if err = job.runInPod(ctx, podName, "alpine", []string{"tar", "-xf", "/data/tmp.gz", "-C", "/data"}, nil, os.Stdout, os.Stderr, logger); err != nil {
		return err
	}

	// Get rid of the temporary pod used for copying data
	if err = job.stopPod(ctx, podName, logger); err != nil {
		return err
	}

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
