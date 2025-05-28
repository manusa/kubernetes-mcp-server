package kubernetes

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/manusa/kubernetes-mcp-server/pkg/helm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	discoveryClient             discovery.CachedDiscoveryInterface
	deferredDiscoveryRESTMapper *restmapper.DeferredDiscoveryRESTMapper
	dynamicClient               *dynamic.DynamicClient
	Helm                        *helm.Helm
}

func NewKubernetes(kubeconfig string) (*Kubernetes, error) {
	k8s := &Kubernetes{
		Kubeconfig: kubeconfig,
	}
	if err := resolveKubernetesConfigurations(k8s); err != nil {
		return nil, err
	}
	// TODO: Won't work because not all client-go clients use the shared context (e.g. discovery client uses context.TODO())
	//k8s.cfg.Wrap(func(original http.RoundTripper) http.RoundTripper {
	//	return &impersonateRoundTripper{original}
	//})
	var err error
	k8s.clientSet, err = kubernetes.NewForConfig(k8s.cfg)
	if err != nil {
		return nil, err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(k8s.cfg)
	if err != nil {
		return nil, err
	}
	k8s.discoveryClient = memory.NewMemCacheClient(discoveryClient)
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
	k8s.Helm = helm.NewHelm(k8s)
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

func (k *Kubernetes) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return k.discoveryClient, nil
}

func (k *Kubernetes) ToRESTMapper() (meta.RESTMapper, error) {
	return k.deferredDiscoveryRESTMapper, nil
}

func (k *Kubernetes) Derived(ctx context.Context) *Kubernetes {
	bearerToken, ok := ctx.Value(AuthorizationBearerTokenHeader).(string)
	if !ok {
		return k
	}
	var _ error // TODO: ignored --> should be handled eventually
	derivedCfg := rest.CopyConfig(k.cfg)
	derivedCfg.BearerToken = bearerToken
	derivedCfg.BearerTokenFile = ""
	derivedCfg.AuthProvider = nil
	derivedCfg.Username = ""
	derivedCfg.Password = ""
	derivedCfg.Impersonate = rest.ImpersonationConfig{}
	clientcmdapiConfig, _ := k.clientCmdConfig.RawConfig()
	clientcmdapiConfig.AuthInfos = make(map[string]*clientcmdapi.AuthInfo)
	derived := &Kubernetes{
		Kubeconfig:      k.Kubeconfig,
		clientCmdConfig: clientcmd.NewDefaultClientConfig(clientcmdapiConfig, nil),
		cfg:             derivedCfg,
	}
	derived.clientSet, _ = kubernetes.NewForConfig(derived.cfg)
	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(derived.cfg)
	derived.discoveryClient = memory.NewMemCacheClient(discoveryClient)
	derived.deferredDiscoveryRESTMapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(derived.discoveryClient))
	derived.dynamicClient, _ = dynamic.NewForConfig(derived.cfg)
	derived.scheme = runtime.NewScheme()
	derived.parameterCodec = runtime.NewParameterCodec(derived.scheme)
	derived.Helm = helm.NewHelm(derived)
	return derived
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
