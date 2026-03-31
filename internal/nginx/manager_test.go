package nginx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToggleSite_PreservesSubfolderExtension(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	enabledPath := filepath.Join(baseDir, "docs.subfolder.conf")
	if err := os.WriteFile(enabledPath, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write enabled config: %v", err)
	}

	manager := NewManager(baseDir)

	status, err := manager.ToggleSite("docs")
	if err != nil {
		t.Fatalf("ToggleSite disable error: %v", err)
	}
	if status != StatusDisabled {
		t.Fatalf("expected disabled status, got %s", status)
	}

	disabledPath := filepath.Join(baseDir, "docs.subfolder.conf.disabled")
	if _, err := os.Stat(disabledPath); err != nil {
		t.Fatalf("expected disabled subfolder config: %v", err)
	}
	if _, err := os.Stat(enabledPath); !os.IsNotExist(err) {
		t.Fatalf("expected enabled path to be renamed, got err=%v", err)
	}

	status, err = manager.ToggleSite("docs")
	if err != nil {
		t.Fatalf("ToggleSite enable error: %v", err)
	}
	if status != StatusEnabled {
		t.Fatalf("expected enabled status, got %s", status)
	}

	if _, err := os.Stat(enabledPath); err != nil {
		t.Fatalf("expected enabled subfolder config restored: %v", err)
	}
	if _, err := os.Stat(disabledPath); !os.IsNotExist(err) {
		t.Fatalf("expected disabled path to be renamed away, got err=%v", err)
	}
}

func TestListSites_ParsesUpstreamProto(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "secure.subdomain.conf")
	content := `server {
    location / {
        set $upstream_app secure-app;
        set $upstream_port 8443;
        set $upstream_proto https;
        proxy_pass $upstream_proto://$upstream_app:$upstream_port;
    }
}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	manager := NewManager(baseDir)
	sites, err := manager.ListSites()
	if err != nil {
		t.Fatalf("ListSites error: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}

	site := sites[0]
	if site.UpstreamProto != "https" {
		t.Fatalf("expected upstream proto https, got %q", site.UpstreamProto)
	}
	if site.TargetDest != "secure-app" {
		t.Fatalf("expected target secure-app, got %q", site.TargetDest)
	}
	if site.ContainerPort != "8443" {
		t.Fatalf("expected port 8443, got %q", site.ContainerPort)
	}
}
