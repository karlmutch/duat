package kubernetes

import (
	"context"

	"github.com/jjeffery/kv"
	"github.com/karlmutch/stack"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (job *Job) startMinimalPod(ctx context.Context, name string, volume string, logger chan *Status) (err kv.Error) {

	api := Client().CoreV1().Pods(job.start.Namespace)

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

	// Create Deployment
	_, errGo := api.Create(podSpec)
	if errGo != nil {
		job.failed = kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		return job.failed
	}
	return nil
}
