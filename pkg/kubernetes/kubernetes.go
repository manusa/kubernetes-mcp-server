package kubernetes

import (
	"github.com/fsnotify/fsnotify"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

type CloseWatchKubeConfig func() error

type Kubernetes struct {
	// Kubeconfig path override
	Kubeconfig                  string
	cfg                         *rest.Config
	clientCmdConfig             clientcmd.ClientConfig
	CloseWatchKubeConfig        CloseWatchKubeConfig
	scheme                      *runtime.Scheme
	parameterCodec              runtime.ParameterCodec
	clientSet                   kubernetes.Interface
	discoveryClient             *discovery.DiscoveryClient
	deferredDiscoveryRESTMapper *restmapper.DeferredDiscoveryRESTMapper
	dynamicClient               *dynamic.DynamicClient
}

func NewKubernetes(kubeconfig string) (*Kubernetes, error) {
	k8s := &Kubernetes{
		Kubeconfig: kubeconfig,
	}
	if err := resolveKubernetesConfigurations(k8s); err != nil {
		return nil, err
	}
	var err error
	k8s.clientSet, err = kubernetes.NewForConfig(k8s.cfg)
	if err != nil {
		return nil, err
	}
	k8s.discoveryClient, err = discovery.NewDiscoveryClientForConfig(k8s.cfg)
	if err != nil {
		return nil, err
	}
	k8s.deferredDiscoveryRESTMapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(k8s.discoveryClient))
	k8s.dynamicClient, err = dynamic.NewForConfig(k8s.cfg)
	if err != nil {
		return nil, err
	}
	k8s.scheme = runtime.NewScheme()
	if err = v1.AddToScheme(k8s.scheme); err != nil {
		return nil, err
	}
	k8s.parameterCodec = runtime.NewParameterCodec(k8s.scheme)
	return k8s, nil
}

func (k *Kubernetes) WatchKubeConfig(onKubeConfigChange func() error) {
	if k.clientCmdConfig == nil {
		return
	}
	kubeConfigFiles := k.clientCmdConfig.ConfigAccess().GetLoadingPrecedence()
	if len(kubeConfigFiles) == 0 {
		return
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	for _, file := range kubeConfigFiles {
		_ = watcher.Add(file)
	}
	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}
				_ = onKubeConfigChange()
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	if k.CloseWatchKubeConfig != nil {
		_ = k.CloseWatchKubeConfig()
	}
	k.CloseWatchKubeConfig = watcher.Close
}

func (k *Kubernetes) Close() {
	if k.CloseWatchKubeConfig != nil {
		_ = k.CloseWatchKubeConfig()
	}
}

func marshal(v any) (string, error) {
	switch t := v.(type) {
	case []unstructured.Unstructured:
		for i := range t {
			t[i].SetManagedFields(nil)
		}
	case []*unstructured.Unstructured:
		for i := range t {
			t[i].SetManagedFields(nil)
		}
	case unstructured.Unstructured:
		t.SetManagedFields(nil)
	case *unstructured.Unstructured:
		t.SetManagedFields(nil)
	}
	ret, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}

// KubeconfigPath returns the kubeconfig path used by this Kubernetes client
func (k *Kubernetes) KubeconfigPath() string {
	return k.Kubeconfig
}

// CurrentContext returns the current context from the kubeconfig
func (k *Kubernetes) CurrentContext() string {
	if k.clientCmdConfig == nil {
		return ""
	}
	if rawConfig, err := k.clientCmdConfig.RawConfig(); err == nil {
		return rawConfig.CurrentContext
	}
	return ""
}

// ConfiguredNamespace returns the namespace configured in the kubeconfig/context
func (k *Kubernetes) ConfiguredNamespace() string {
	if k.clientCmdConfig == nil {
		return ""
	}
	if ns, _, nsErr := k.clientCmdConfig.Namespace(); nsErr == nil {
		return ns
	}
	return ""
}

func (k *Kubernetes) namespaceOrDefault(namespace string) string {
	if namespace == "" {
		return k.ConfiguredNamespace()
	}
	return namespace
}
