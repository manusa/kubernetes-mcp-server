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
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

const (
	AuthorizationHeader            = "Kubernetes-Authorization"
	AuthorizationBearerTokenHeader = "kubernetes-authorization-bearer-token"
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
	klog.V(5).Infof("%s header found, using provided bearer token", AuthorizationBearerTokenHeader)
	derivedCfg := rest.CopyConfig(k.cfg)

	// If we have a bearer token, use it for authentication
	if bearerToken != "" {
		derivedCfg.BearerToken = bearerToken
		derivedCfg.BearerTokenFile = ""
		derivedCfg.Username = ""
		derivedCfg.Password = ""

		// Only clear auth providers if we're using a bearer token
		// This preserves OIDC configuration when no token is provided
		derivedCfg.AuthProvider = nil
		derivedCfg.AuthConfigPersister = nil
		derivedCfg.ExecProvider = nil
	}

	derivedCfg.Impersonate = rest.ImpersonationConfig{}
	clientCmdApiConfig, err := k.clientCmdConfig.RawConfig()
	if err != nil {
		return k
	}
	clientCmdApiConfig.AuthInfos = make(map[string]*clientcmdapi.AuthInfo)
	derived := &Kubernetes{
		Kubeconfig:      k.Kubeconfig,
		clientCmdConfig: clientcmd.NewDefaultClientConfig(clientCmdApiConfig, nil),
		cfg:             derivedCfg,
		scheme:          k.scheme,
		parameterCodec:  k.parameterCodec,
	}
	derived.clientSet, err = kubernetes.NewForConfig(derived.cfg)
	if err != nil {
		return k
	}
	derived.discoveryClient = memory.NewMemCacheClient(discovery.NewDiscoveryClient(derived.clientSet.CoreV1().RESTClient()))
	derived.deferredDiscoveryRESTMapper = restmapper.NewDeferredDiscoveryRESTMapper(derived.discoveryClient)
	derived.dynamicClient, err = dynamic.NewForConfig(derived.cfg)
	if err != nil {
		return k
	}
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
