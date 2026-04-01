package cli

import (
	"errors"
	"testing"

	"swag-cli/internal/nginx"
)

type fakeSiteToggler struct {
	status    nginx.SiteStatus
	err       error
	subdomain string
}

func (f *fakeSiteToggler) ToggleSite(subdomain string) (nginx.SiteStatus, error) {
	f.subdomain = subdomain
	return f.status, f.err
}

func TestBuildAddConfigDataDefaultsSubdomainToContainerName(t *testing.T) {
	t.Parallel()

	got := buildAddConfigData("my-app", "", 8080, "http")
	if got.Subdomain != "my-app" {
		t.Fatalf("buildAddConfigData() subdomain = %q, want %q", got.Subdomain, "my-app")
	}
}

func TestBuildAddConfigDataPreservesExplicitSubdomain(t *testing.T) {
	t.Parallel()

	got := buildAddConfigData("my-app", "app", 8080, "https")
	if got.Subdomain != "app" || got.Protocol != "https" {
		t.Fatalf("buildAddConfigData() = %+v", got)
	}
}

func TestToggleSiteUsesManager(t *testing.T) {
	t.Parallel()

	manager := &fakeSiteToggler{status: nginx.StatusDisabled}
	status, err := toggleSite(manager, "app")
	if err != nil {
		t.Fatalf("toggleSite() error = %v", err)
	}
	if status != nginx.StatusDisabled || manager.subdomain != "app" {
		t.Fatalf("toggleSite() status=%s subdomain=%q", status, manager.subdomain)
	}
}

func TestToggleSitePropagatesError(t *testing.T) {
	t.Parallel()

	manager := &fakeSiteToggler{err: errors.New("boom")}
	if _, err := toggleSite(manager, "app"); err == nil {
		t.Fatalf("expected toggleSite error")
	}
}

func TestBuildHomepageConfig(t *testing.T) {
	t.Parallel()

	got := buildHomepageConfig("app", "example.com", 8443, "https", true)
	if got.UpstreamApp != "app" || got.Domain != "example.com" || got.UpstreamPort != 8443 || got.UpstreamProto != "https" || !got.KeepServerNameUnderscore {
		t.Fatalf("buildHomepageConfig() = %+v", got)
	}
}
