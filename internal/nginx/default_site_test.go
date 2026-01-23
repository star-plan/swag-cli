package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleDefaultSiteConf = `error_page 502 /502.html;

# redirect all traffic to https
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
    return 301 https://$host$request_uri;
}

# main server block
server {
    listen 443 ssl http2 default_server;
    listen [::]:443 ssl http2 default_server;

    root /config/www;
    index index.html index.htm index.php;

    server_name _;

    # enable subfolder method reverse proxy confs
    include /config/nginx/proxy-confs/*.subfolder.conf;
    
    # all ssl related config moved to ssl.conf
    include /config/nginx/ssl.conf;

    client_max_body_size 0;

    location / {
        # enable the next two lines for http auth
        #auth_basic "Restricted";
        #auth_basic_user_file /config/nginx/.htpasswd;

        # enable for Authelia
        #include /config/nginx/authelia-location.conf;

        try_files $uri $uri/ /index.html /index.php?$args =404;
    }
}
`

func TestDefaultSiteEditor_SetHomepage_ModifiesLocationAndServerName(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "default")
	if err := os.WriteFile(confPath, []byte(sampleDefaultSiteConf), 0o644); err != nil {
		t.Fatalf("write temp default: %v", err)
	}

	editor := NewDefaultSiteEditor(confPath)
	res, err := editor.SetHomepage(HomepageConfig{
		Domain:        "example.com",
		UpstreamApp:   "my-app",
		UpstreamPort:  8080,
		UpstreamProto: "http",
	}, false)
	if err != nil {
		t.Fatalf("SetHomepage error: %v", err)
	}
	if !res.Changed {
		t.Fatalf("expected changed=true")
	}
	if res.BackupPath == "" {
		t.Fatalf("expected backup path")
	}
	if _, err := os.Stat(res.BackupPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	updated, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	out := string(updated)

	if !strings.Contains(out, "server_name example.com;") {
		t.Fatalf("expected server_name example.com; got:\n%s", out)
	}
	if strings.Contains(out, "try_files ") {
		t.Fatalf("expected try_files removed; got:\n%s", out)
	}
	if !strings.Contains(out, "include /config/nginx/proxy.conf;") {
		t.Fatalf("expected proxy.conf include; got:\n%s", out)
	}
	if !strings.Contains(out, "include /config/nginx/resolver.conf;") {
		t.Fatalf("expected resolver.conf include; got:\n%s", out)
	}
	if !strings.Contains(out, "set $upstream_app my-app;") {
		t.Fatalf("expected upstream_app set; got:\n%s", out)
	}
	if !strings.Contains(out, "set $upstream_port 8080;") {
		t.Fatalf("expected upstream_port set; got:\n%s", out)
	}
	if !strings.Contains(out, "set $upstream_proto http;") {
		t.Fatalf("expected upstream_proto set; got:\n%s", out)
	}
	if !strings.Contains(out, "proxy_pass $upstream_proto://$upstream_app:$upstream_port;") {
		t.Fatalf("expected proxy_pass line; got:\n%s", out)
	}
}

func TestDefaultSiteEditor_SetHomepage_IsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "default")
	if err := os.WriteFile(confPath, []byte(sampleDefaultSiteConf), 0o644); err != nil {
		t.Fatalf("write temp default: %v", err)
	}

	editor := NewDefaultSiteEditor(confPath)

	_, err := editor.SetHomepage(HomepageConfig{
		Domain:        "example.com",
		UpstreamApp:   "my-app",
		UpstreamPort:  8080,
		UpstreamProto: "http",
	}, false)
	if err != nil {
		t.Fatalf("SetHomepage first error: %v", err)
	}

	res2, err := editor.SetHomepage(HomepageConfig{
		Domain:        "example.com",
		UpstreamApp:   "my-app",
		UpstreamPort:  8080,
		UpstreamProto: "http",
	}, false)
	if err != nil {
		t.Fatalf("SetHomepage second error: %v", err)
	}
	if res2.Changed {
		t.Fatalf("expected changed=false on second run")
	}
}

func TestDefaultSiteEditor_ClearHomepage_RestoresTryFilesAndUnderscore(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "default")
	if err := os.WriteFile(confPath, []byte(sampleDefaultSiteConf), 0o644); err != nil {
		t.Fatalf("write temp default: %v", err)
	}

	editor := NewDefaultSiteEditor(confPath)
	_, err := editor.SetHomepage(HomepageConfig{
		Domain:        "example.com",
		UpstreamApp:   "my-app",
		UpstreamPort:  8080,
		UpstreamProto: "http",
	}, false)
	if err != nil {
		t.Fatalf("SetHomepage error: %v", err)
	}

	res, err := editor.ClearHomepage("", true, false)
	if err != nil {
		t.Fatalf("ClearHomepage error: %v", err)
	}
	if !res.Changed {
		t.Fatalf("expected changed=true")
	}
	if res.BackupPath == "" {
		t.Fatalf("expected backup path")
	}
	if _, err := os.Stat(res.BackupPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	updated, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	out := string(updated)

	if !strings.Contains(out, "server_name _;") {
		t.Fatalf("expected server_name _; got:\n%s", out)
	}
	if !strings.Contains(out, "try_files $uri $uri/ /index.html /index.php?$args =404;") {
		t.Fatalf("expected try_files restored; got:\n%s", out)
	}
	if strings.Contains(out, "proxy_pass ") {
		t.Fatalf("expected proxy_pass removed; got:\n%s", out)
	}
}

func TestDefaultSiteEditor_SetHomepage_KeepServerNameUnderscore(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "default")
	if err := os.WriteFile(confPath, []byte(sampleDefaultSiteConf), 0o644); err != nil {
		t.Fatalf("write temp default: %v", err)
	}

	editor := NewDefaultSiteEditor(confPath)
	_, err := editor.SetHomepage(HomepageConfig{
		Domain:                   "example.com",
		UpstreamApp:              "my-app",
		UpstreamPort:             8080,
		UpstreamProto:            "http",
		KeepServerNameUnderscore: true,
	}, false)
	if err != nil {
		t.Fatalf("SetHomepage error: %v", err)
	}

	updated, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	out := string(updated)

	if strings.Contains(out, "server_name example.com;") {
		t.Fatalf("expected server_name not changed; got:\n%s", out)
	}
	if !strings.Contains(out, "server_name _;") {
		t.Fatalf("expected server_name _; got:\n%s", out)
	}
}

