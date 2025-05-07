package helm

import (
	"context"
	"log"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

// Helm provides methods to interact with Helm releases
// Mirrors the abstraction style of pkg/kubernetes

type Helm struct {
	settings *cli.EnvSettings
}

// NewHelm creates a new Helm instance using kubeconfig, context, and namespace settings
func NewHelm(kubeconfig, kubeContext, namespace string) *Helm {
	settings := cli.New()
	if kubeconfig != "" {
		settings.KubeConfig = kubeconfig
	}
	if kubeContext != "" {
		settings.KubeContext = kubeContext
	}
	if namespace != "" {
		settings.SetNamespace(namespace)
	}
	return &Helm{settings: settings}
}

// ReleasesList lists Helm releases in a specific namespace (or all namespaces if namespace is empty)
func (h *Helm) ReleasesList(ctx context.Context, namespace string) ([]*release.Release, error) {
	// If no namespace is given, use the default from kubeconfig
	if namespace == "" {
		namespace = h.settings.Namespace()
	}
	cfg := new(action.Configuration)
	if err := cfg.Init(h.settings.RESTClientGetter(), namespace, "", log.Printf); err != nil {
		return nil, err
	}
	list := action.NewList(cfg)
	// To list across all namespaces, set AllNamespaces to true
	if namespace == "" || namespace == "all" {
		list.AllNamespaces = true
	}
	return list.Run()
}
