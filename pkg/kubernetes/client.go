package kubernetes

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jjeffery/kv"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/karlmutch/stack"
)

var (
	clientNamespace = ""
	clientSet       *kubernetes.Clientset // Thread safe Kubernetes API https://github.com/kubernetes/client-go/issues/36

	initFailure = kv.NewError("uninitialized")
)

func initInCluster() (err kv.Error) {
	cfg, errGo := rest.InClusterConfig()
	if errGo != nil {
		return kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

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
	kubeConfig = os.Getenv("KUBE_CONFIG")

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

func SetNamespace(namespace string) (err kv.Error) {
	if initFailure != nil {
		return initFailure
	}
	clientNamespace = namespace

	// TODO Check the namespace exists
	return nil
}

func Namespace() (namespace string, err kv.Error) {
	if initFailure != nil {
		return "", initFailure
	}
	return clientNamespace, nil
}

func Client() (client *kubernetes.Clientset) {
	return clientSet
}
