package kubernetes

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jjeffery/kv"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/go-stack/stack"
)

var (
	clientNamespace = ""
	clientSet       *kubernetes.Clientset // Thread safe Kubernetes API https://github.com/kubernetes/client-go/issues/36
	clientCfg       *rest.Config

	initFailure = kv.NewError("uninitialized")
)

func initInCluster() (err kv.Error) {
	cfg, errGo := rest.InClusterConfig()
	if errGo != nil {
		return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	clientCfg = cfg
	clientSet, errGo = kubernetes.NewForConfig(cfg)
	if errGo != nil {
		return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	ns, errGo := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if errGo != nil {
		return kv.NewError("kubernetes not detected").With("stack", stack.Trace().TrimRuntime())
	}

	clientNamespace = string(ns)

	return nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func kubeConfigGet() (kubeConfig string) {
	kubeConfig = os.Getenv("KUBECONFIG")

	if len(kubeConfig) == 0 {
		if home := homeDir(); home != "" {
			kubeConfig = filepath.Join(home, ".kube", "config")
		}
	}
	return kubeConfig
}

func initOutOfCluster() (err kv.Error) {
	configFile := kubeConfigGet()
	clientConfig, errGo := clientcmd.BuildConfigFromFlags("", configFile)
	if errGo != nil {
		return kv.Wrap(errGo).With("config", configFile).With("stack", stack.Trace().TrimRuntime())
	}
	if errGo != nil {
		return kv.Wrap(errGo).With("config", configFile).With("stack", stack.Trace().TrimRuntime())
	}

	clientCfg = clientConfig
	clientSet, errGo = kubernetes.NewForConfig(clientConfig)
	if errGo != nil {
		return kv.Wrap(errGo).With("config", configFile).With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}

func init() {
	if initFailure = initInCluster(); initFailure != nil {
		if err := initOutOfCluster(); err == nil {
			initFailure = nil
		}
	}
}

// Client returns the clientset used to access the Kubernetes cluster that is being used to
// run our pods within the CI/CD pipeline
//
func Client() (client *kubernetes.Clientset) {
	return clientSet
}

// RestConfig returns the rest configuration used to access the Kubernetes cluster Pods that are being created
//
func RestConfig() (client *rest.Config) {
	return clientCfg
}
