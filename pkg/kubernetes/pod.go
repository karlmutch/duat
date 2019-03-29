package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"

	apiv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func (job *Job) startMinimalPod(ctx context.Context, name string, volume string, logger chan *Status) (err kv.Error) {

	api := Client().CoreV1()

	podSpec := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:  "alpine",
					Image: "alpine",
					SecurityContext: &apiv1.SecurityContext{
						Privileged: &[]bool{false}[0],
					},
					ImagePullPolicy: apiv1.PullPolicy(apiv1.PullIfNotPresent),
					Env:             []apiv1.EnvVar{},
					VolumeMounts: []apiv1.VolumeMount{
						apiv1.VolumeMount{
							Name:      volume,
							MountPath: "/data",
						},
					},
					Command: []string{"/bin/sleep"},
					Args:    []string{"1d"},
				},
			},
			RestartPolicy:    apiv1.RestartPolicyNever,
			ImagePullSecrets: []apiv1.LocalObjectReference{},
			Volumes: []apiv1.Volume{
				apiv1.Volume{
					Name: volume,
					VolumeSource: apiv1.VolumeSource{
						HostPath: &apiv1.HostPathVolumeSource{
							Path: "/data",
						},
					},
				},
			},
		},
	}

	_, errGo := api.Pods(job.start.Namespace).Create(podSpec)
	if errGo != nil {
		job.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		return job.failed
	}

	watchOpts := metav1.ListOptions{
		Watch:         true,
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	}

	w, errGo := api.Pods(job.start.Namespace).Watch(watchOpts)
	if errGo != nil {
		job.failed = kv.Wrap(errGo).With("namespace", job.start.Namespace, "name", name, "stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	defer w.Stop()

	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()

	err = func() (err kv.Error) {
		lastPhase := apiv1.PodUnknown
		for {
			select {
			case events, ok := <-w.ResultChan():
				if !ok {
					return
				}
				resp, ok := events.Object.(*apiv1.Pod)
				if ok && resp.Status.Phase == apiv1.PodRunning {
					return nil
				}
				lastPhase = resp.Status.Phase
			case <-waitCtx.Done():
				job.failed = kv.NewError("pod not started").With("namespace", job.start.Namespace, "pod", name, "phase", spew.Sdump(lastPhase), "stack", stack.Trace().TrimRuntime())
				return job.failed
			}
		}
	}()

	return err
}

// filePod is used to copy data between a local machine and a remote pod.  Kubernetes does not support this as a direct
// call instead software attached to the pod using a shell and then streams file contents across.
//
// Further examples and information include:
// https://github.com/AOEpeople/kube-container-exec/blob/master/main.go
// https://github.com/maorfr/skbn/blob/master/pkg/skbn/kube.go
// https://medium.com/nuvo-group-tech/copy-files-and-directories-between-kubernetes-and-s3-d290ded9a5e0
// https://gist.github.com/kyroy/8453a0c4e075e91809db9749e0adcff2
//
func (job *Job) filePod(ctx context.Context, name string, retrieve bool, localFile string, remoteFile string, logger chan *Status) (err kv.Error) {

	restConfig := &rest.Config{}
	restClient := Client().CoreV1().RESTClient()

	req := restClient.Post().
		Namespace(job.start.Namespace).
		Resource("pods").
		Name(name).
		SubResource("exec").
		Param("container", "alpine").
		Param("stdout", "true").
		Param("stderr", "true")

	localF := &os.File{}
	if retrieve {
		out, errGo := os.OpenFile(localFile, os.O_RDWR|os.O_CREATE, 0600)
		if errGo != nil {
			job.failed = kv.Wrap(errGo).With("fn", localFile, "namespace", job.start.Namespace, "pod", name, "stack", stack.Trace().TrimRuntime())
			return job.failed
		}
		localF = out
		for _, item := range []string{"/bin/cp", remoteFile, "/dev/stdout"} {
			req.Param("command", item)
		}
		req.Param("stdin", "false")
	} else {
		in, errGo := os.Open(localFile)
		if errGo != nil {
			job.failed = kv.Wrap(errGo).With("fn", localFile, "namespace", job.start.Namespace, "pod", name, "stack", stack.Trace().TrimRuntime())
			return job.failed
		}
		localF = in
		for _, item := range []string{"/bin/cp", "/dev/stdin", remoteFile} {
			req.Param("command", item)
		}
		req.Param("stdin", "true")
	}

	defer localF.Close()

	executor, errGo := remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, req.URL())
	if errGo != nil {
		job.failed = kv.Wrap(errGo).With("namespace", job.start.Namespace, "pod", name, "stack", stack.Trace().TrimRuntime())
		return job.failed
	}

	localF.Sync()
	os.Stdout.Sync()
	os.Stderr.Sync()

	streamOpts := remotecommand.StreamOptions{
		Stdin:             nil,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		Tty:               false,
		TerminalSizeQueue: nil,
	}

	if retrieve {
		streamOpts.Stdout = localF
	} else {
		streamOpts.Stdin = localF
	}
	if errGo = executor.Stream(streamOpts); errGo != nil {
		job.failed = kv.Wrap(errGo).With("namespace", job.start.Namespace, "pod", name, "stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	return nil
}

func (job *Job) stopPod(ctx context.Context, name string, logger chan *Status) (err kv.Error) {

	api := Client().CoreV1()

	deleteOpts := &metav1.DeleteOptions{
		GracePeriodSeconds: &[]int64{0}[0],
	}

	deadline, ok := ctx.Deadline()
	if ok && !deadline.Before(time.Now().Add(time.Second)) {
		*deleteOpts.GracePeriodSeconds = int64(deadline.Sub(time.Now().Add(time.Second)).Truncate(time.Second).Seconds())
	}

	if errGo := api.Pods(job.start.Namespace).Delete(name, deleteOpts); errGo != nil {
		job.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	return nil
}
