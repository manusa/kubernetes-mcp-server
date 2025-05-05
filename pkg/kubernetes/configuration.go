package kubernetes

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
)

// InClusterConfig is a variable that holds the function to get the in-cluster config
// Exposed for testing
var InClusterConfig = func() (*rest.Config, error) {
	// TODO use kubernetes.default.svc instead of resolved server
	// Currently running into: `http: server gave HTTP response to HTTPS client`
	inClusterConfig, err := rest.InClusterConfig()
	if inClusterConfig != nil {
		inClusterConfig.Host = "https://kubernetes.default.svc"
	}
	return inClusterConfig, err
}

// resolveKubernetesConfigurations resolves the required kubernetes configurations and sets them in the Kubernetes struct
func resolveKubernetesConfigurations(kubernetes *Kubernetes) error {
	// Always set clientCmdConfig
	pathOptions := clientcmd.NewDefaultPathOptions()
	if kubernetes.Kubeconfig != "" {
		pathOptions.LoadingRules.ExplicitPath = kubernetes.Kubeconfig
	}
	kubernetes.clientCmdConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		pathOptions.LoadingRules,
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}})
	var err error
	if kubernetes.IsInCluster() {
		kubernetes.cfg, err = InClusterConfig()
		if err == nil && kubernetes.cfg != nil {
			return nil
		}
	}
	// Out of cluster
	kubernetes.cfg, err = kubernetes.clientCmdConfig.ClientConfig()
	if kubernetes.cfg != nil && kubernetes.cfg.UserAgent == "" {
		kubernetes.cfg.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	return err
}

func (k *Kubernetes) IsInCluster() bool {
	if k.Kubeconfig != "" {
		return false
	}
	cfg, err := InClusterConfig()
	return err == nil && cfg != nil
}

func (k *Kubernetes) ConfigurationView(minify bool) (string, error) {
	var cfg clientcmdapi.Config
	var err error
	if k.IsInCluster() {
		cfg = *clientcmdapi.NewConfig()
		cfg.Clusters["cluster"] = &clientcmdapi.Cluster{
			Server:                k.cfg.Host,
			InsecureSkipTLSVerify: k.cfg.Insecure,
		}
		cfg.AuthInfos["user"] = &clientcmdapi.AuthInfo{
			Token: k.cfg.BearerToken,
		}
		cfg.Contexts["context"] = &clientcmdapi.Context{
			Cluster:  "cluster",
			AuthInfo: "user",
		}
		cfg.CurrentContext = "context"
	} else if cfg, err = k.clientCmdConfig.RawConfig(); err != nil {
		return "", err
	}
	if minify {
		if err = clientcmdapi.MinifyConfig(&cfg); err != nil {
			return "", err
		}
	}
	if err = clientcmdapi.FlattenConfig(&cfg); err != nil {
		// ignore error
		//return "", err
	}
	convertedObj, err := latest.Scheme.ConvertToVersion(&cfg, latest.ExternalVersion)
	if err != nil {
		return "", err
	}
	return marshal(convertedObj)
}
