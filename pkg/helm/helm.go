package helm

import (
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"log"
	"sigs.k8s.io/yaml"
)

type Kubernetes interface {
	genericclioptions.RESTClientGetter
	NamespaceOrDefault(namespace string) string
}

type Helm struct {
	kubernetes Kubernetes
}

// NewHelm creates a new Helm instance
func NewHelm(kubernetes Kubernetes, namespace string) *Helm {
	settings := cli.New()
	if namespace != "" {
		settings.SetNamespace(namespace)
	}
	return &Helm{kubernetes: kubernetes}
}

// ReleasesList lists all the releases for the specified namespace (or current namespace if). Or allNamespaces is true, it lists all releases across all namespaces.
func (h *Helm) ReleasesList(namespace string, allNamespaces bool) (string, error) {
	cfg := new(action.Configuration)
	applicableNamespace := ""
	if !allNamespaces {
		applicableNamespace = h.kubernetes.NamespaceOrDefault(namespace)
	}
	if err := cfg.Init(h.kubernetes, applicableNamespace, "", log.Printf); err != nil {
		return "", err
	}
	list := action.NewList(cfg)
	list.AllNamespaces = allNamespaces
	releases, err := list.Run()
	if err != nil {
		return "", err
	} else if len(releases) == 0 {
		return "No Helm releases found", nil
	}
	ret, err := yaml.Marshal(releases)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}
