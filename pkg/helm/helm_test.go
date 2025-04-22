package helm

import (
	"context"
	"testing"
)

func TestNewHelm(t *testing.T) {
	h := NewHelm()
	if h == nil {
		t.Fatal("expected non-nil Helm instance")
	}
	if h.settings == nil {
		t.Fatal("expected non-nil settings in Helm instance")
	}
}

func TestHelm_ReleasesList_DefaultNamespace(t *testing.T) {
	h := NewHelm()
	// Use a namespace that is likely to exist, or leave empty for default
	releases, err := h.ReleasesList(context.Background(), "")
	if err != nil {
		t.Skipf("skipping: could not list releases (likely no cluster/helm configured): %v", err)
	}
	// No assertion on releases count, just check type
	if releases == nil {
		t.Error("expected releases slice, got nil")
	}
}

func TestHelm_ReleasesList_AllNamespaces(t *testing.T) {
	h := NewHelm()
	releases, err := h.ReleasesList(context.Background(), "all")
	if err != nil {
		t.Skipf("skipping: could not list all releases (likely no cluster/helm configured): %v", err)
	}
	if releases == nil {
		t.Error("expected releases slice, got nil")
	}
}
