package kubernetes

import (
	authorizationv1api "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
)

// AccessControlClientset is a limited clientset delegating interface to the standard kubernetes.Clientset
// Only a limited set of functions are implemented with a single point of access to the kubernetes API where
// apiVersion and kinds are checked for allowed access
type AccessControlClientset struct {
	delegate        kubernetes.Interface
	discoveryClient discovery.DiscoveryInterface
	staticConfig    *config.StaticConfig // TODO: maybe just store the denied resource slice
}

func (a *AccessControlClientset) DiscoveryClient() discovery.DiscoveryInterface {
	return a.discoveryClient
}

func (a *AccessControlClientset) Pods(namespace string) (corev1.PodInterface, error) {
	gvk := &schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	if !isAllowed(a.staticConfig, gvk) {
		return nil, isNotAllowedError(gvk)
	}
	return a.delegate.CoreV1().Pods(namespace), nil
}

func (a *AccessControlClientset) PodsExec(namespace, name string) (*rest.Request, error) {
	gvk := &schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	if !isAllowed(a.staticConfig, gvk) {
		return nil, isNotAllowedError(gvk)
	}
	// https://github.com/kubernetes/kubectl/blob/5366de04e168bcbc11f5e340d131a9ca8b7d0df4/pkg/cmd/exec/exec.go#L382-L397
	return a.delegate.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(name).
		SubResource("exec"), nil
}

func (a *AccessControlClientset) Services(namespace string) (corev1.ServiceInterface, error) {
	gvk := &schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}
	if !isAllowed(a.staticConfig, gvk) {
		return nil, isNotAllowedError(gvk)
	}
	return a.delegate.CoreV1().Services(namespace), nil
}

func (a *AccessControlClientset) SelfSubjectAccessReviews() (authorizationv1.SelfSubjectAccessReviewInterface, error) {
	gvk := &schema.GroupVersionKind{Group: authorizationv1api.GroupName, Version: authorizationv1api.SchemeGroupVersion.Version, Kind: "SelfSubjectAccessReview"}
	if !isAllowed(a.staticConfig, gvk) {
		return nil, isNotAllowedError(gvk)
	}
	return a.delegate.AuthorizationV1().SelfSubjectAccessReviews(), nil
}

func NewAccessControlClientset(cfg *rest.Config, staticConfig *config.StaticConfig) (*AccessControlClientset, error) {
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &AccessControlClientset{
		delegate: clientSet, discoveryClient: clientSet.DiscoveryClient, staticConfig: staticConfig,
	}, nil
}
