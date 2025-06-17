package kubernetes

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/manusa/kubernetes-mcp-server/pkg/helm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	clientgokubernetes "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/metrics"
	"strings"
)

const (
	AuthorizationHeader = "kubernetes-authorization"
)

type CloseWatchKubeConfig func() error

type Kubernetes interface {
	WatchKubeConfig(onKubeConfigChange func() error)
	Close()
	Derived(ctx context.Context) DerivedKubernetes
	ConfigurationView(minify bool) (runtime.Object, error)
	IsOpenShift(ctx context.Context) bool
}

type DerivedKubernetes interface {
	IsOpenShift(ctx context.Context) bool
	CacheInvalidate()
	NewHelm() *helm.Helm
	EventsList(ctx context.Context, namespace string) ([]map[string]any, error)
	NamespacesList(ctx context.Context, options ResourceListOptions) (runtime.Unstructured, error)
	PodsListInAllNamespaces(ctx context.Context, options ResourceListOptions) (runtime.Unstructured, error)
	PodsListInNamespace(ctx context.Context, namespace string, options ResourceListOptions) (runtime.Unstructured, error)
	PodsGet(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error)
	PodsDelete(ctx context.Context, namespace, name string) (string, error)
	PodsLog(ctx context.Context, namespace, name, container string) (string, error)
	PodsRun(ctx context.Context, namespace, name, image string, port int32) ([]*unstructured.Unstructured, error)
	PodsTop(ctx context.Context, options PodsTopOptions) (*metrics.PodMetricsList, error)
	PodsExec(ctx context.Context, namespace, name, container string, command []string) (string, error)
	ProjectsList(ctx context.Context, options ResourceListOptions) (runtime.Unstructured, error)
	ResourcesList(ctx context.Context, gvk *schema.GroupVersionKind, namespace string, options ResourceListOptions) (runtime.Unstructured, error)
	ResourcesGet(ctx context.Context, gvk *schema.GroupVersionKind, namespace, name string) (*unstructured.Unstructured, error)
	ResourcesCreateOrUpdate(ctx context.Context, resource string) ([]*unstructured.Unstructured, error)
	ResourcesDelete(ctx context.Context, gvk *schema.GroupVersionKind, namespace, name string) error
}

type kubernetes struct {
	// Kubeconfig path override
	Kubeconfig                  string
	cfg                         *rest.Config
	clientCmdConfig             clientcmd.ClientConfig
	CloseWatchKubeConfig        CloseWatchKubeConfig
	scheme                      *runtime.Scheme
	parameterCodec              runtime.ParameterCodec
	clientSet                   clientgokubernetes.Interface
	discoveryClient             discovery.CachedDiscoveryInterface
	deferredDiscoveryRESTMapper *restmapper.DeferredDiscoveryRESTMapper
	dynamicClient               *dynamic.DynamicClient
}

func NewKubernetes(kubeconfig string) (Kubernetes, error) {
	k8s := &kubernetes{
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
	k8s.clientSet, err = clientgokubernetes.NewForConfig(k8s.cfg)
	if err != nil {
		return nil, err
	}
	k8s.discoveryClient = memory.NewMemCacheClient(discovery.NewDiscoveryClient(k8s.clientSet.CoreV1().RESTClient()))
	k8s.deferredDiscoveryRESTMapper = restmapper.NewDeferredDiscoveryRESTMapper(k8s.discoveryClient)
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

func (k *kubernetes) WatchKubeConfig(onKubeConfigChange func() error) {
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

func (k *kubernetes) Close() {
	if k.CloseWatchKubeConfig != nil {
		_ = k.CloseWatchKubeConfig()
	}
}

func (k *kubernetes) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return k.discoveryClient, nil
}

func (k *kubernetes) ToRESTMapper() (meta.RESTMapper, error) {
	return k.deferredDiscoveryRESTMapper, nil
}

func (k *kubernetes) Derived(ctx context.Context) DerivedKubernetes {
	authorization, ok := ctx.Value(AuthorizationHeader).(string)
	if !ok || !strings.HasPrefix(authorization, "Bearer ") {
		return k
	}
	klog.V(5).Infof("%s header found (Bearer), using provided bearer token", AuthorizationHeader)
	derivedCfg := rest.CopyConfig(k.cfg)
	derivedCfg.BearerToken = strings.TrimPrefix(authorization, "Bearer ")
	derivedCfg.BearerTokenFile = ""
	derivedCfg.Username = ""
	derivedCfg.Password = ""
	derivedCfg.AuthProvider = nil
	derivedCfg.AuthConfigPersister = nil
	derivedCfg.ExecProvider = nil
	derivedCfg.Impersonate = rest.ImpersonationConfig{}
	clientCmdApiConfig, err := k.clientCmdConfig.RawConfig()
	if err != nil {
		return k
	}
	clientCmdApiConfig.AuthInfos = make(map[string]*clientcmdapi.AuthInfo)
	derived := &kubernetes{
		Kubeconfig:      k.Kubeconfig,
		clientCmdConfig: clientcmd.NewDefaultClientConfig(clientCmdApiConfig, nil),
		cfg:             derivedCfg,
		scheme:          k.scheme,
		parameterCodec:  k.parameterCodec,
	}
	derived.clientSet, err = clientgokubernetes.NewForConfig(derived.cfg)
	if err != nil {
		return k
	}
	derived.discoveryClient = memory.NewMemCacheClient(discovery.NewDiscoveryClient(derived.clientSet.CoreV1().RESTClient()))
	derived.deferredDiscoveryRESTMapper = restmapper.NewDeferredDiscoveryRESTMapper(derived.discoveryClient)
	derived.dynamicClient, err = dynamic.NewForConfig(derived.cfg)
	if err != nil {
		return k
	}
	return derived
}

func (k *kubernetes) CacheInvalidate() {
	if k.discoveryClient != nil {
		k.discoveryClient.Invalidate()
	}
	if k.deferredDiscoveryRESTMapper != nil {
		k.deferredDiscoveryRESTMapper.Reset()
	}
}

func (k *kubernetes) NewHelm() *helm.Helm {
	// This is a derived Kubernetes, so it already has the Helm initialized
	return helm.NewHelm(k)
}
