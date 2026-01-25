package swagexport

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestExportMinimalFiltersProxyConfs(t *testing.T) {
	t.Parallel()

	swagDir := t.TempDir()
	mustWrite(t, filepath.Join(swagDir, "compose.yaml"), "version: '3'\n")

	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "nginx.conf"), "worker_processes  1;\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "site-confs", "default"), "# default\n")

	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "proxy-confs", "a.subdomain.conf"), "# a\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "proxy-confs", "b.subdomain.conf.disabled"), "# b\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "proxy-confs", "c.subdomain.conf.sample"), "# sample\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "proxy-confs", "README.md"), "readme\n")

	mustWrite(t, filepath.Join(swagDir, "config", "dns-conf", "cloudflare.ini"), "token=secret\n")
	mustWrite(t, filepath.Join(swagDir, "config", "log", "nginx", "access.log"), "log\n")

	outZip := filepath.Join(t.TempDir(), "out.zip")
	res, err := Export(Options{
		SwagDir:       swagDir,
		OutPath:       outZip,
		Profile:       ProfileMinimal,
		ProxyConfOnly: true,
		Now: func() time.Time {
			return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if res.OutPath != outZip {
		t.Fatalf("unexpected outPath: %s", res.OutPath)
	}

	files := listZipFiles(t, outZip)
	assertHas(t, files, "compose.yaml")
	assertHas(t, files, "config/nginx/nginx.conf")
	assertHas(t, files, "config/nginx/site-confs/default")
	assertHas(t, files, "config/nginx/proxy-confs/a.subdomain.conf")
	assertHas(t, files, "config/nginx/proxy-confs/b.subdomain.conf.disabled")
	assertHas(t, files, manifestName)

	assertNotHas(t, files, "config/nginx/proxy-confs/c.subdomain.conf.sample")
	assertNotHas(t, files, "config/nginx/proxy-confs/README.md")
	assertNotHas(t, files, "config/dns-conf/cloudflare.ini")
	assertNotHas(t, files, "config/log/nginx/access.log")
}

func TestExportStandardIncludesCustomDirs(t *testing.T) {
	t.Parallel()

	swagDir := t.TempDir()
	mustWrite(t, filepath.Join(swagDir, "compose.yaml"), "version: '3'\n")

	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "nginx.conf"), "worker_processes  1;\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "site-confs", "default"), "# default\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "proxy-confs", "a.subdomain.conf"), "# a\n")

	mustWrite(t, filepath.Join(swagDir, "config", "custom-cont-init.d", "10-test.sh"), "echo hi\n")
	mustWrite(t, filepath.Join(swagDir, "config", "custom-services.d", "20-test.sh"), "echo hi\n")
	mustWrite(t, filepath.Join(swagDir, "config", "crontabs", "root"), "* * * * * echo hi\n")
	mustWrite(t, filepath.Join(swagDir, "config", "php", "php-local.ini"), "x=1\n")
	mustWrite(t, filepath.Join(swagDir, "config", "www", "index.html"), "<html></html>\n")

	outZip := filepath.Join(t.TempDir(), "out.zip")
	files := mustExportAndList(t, Options{
		SwagDir:       swagDir,
		OutPath:       outZip,
		Profile:       ProfileStandard,
		ProxyConfOnly: true,
		Now: func() time.Time {
			return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	})

	assertHas(t, files, "config/custom-cont-init.d/10-test.sh")
	assertHas(t, files, "config/custom-services.d/20-test.sh")
	assertHas(t, files, "config/crontabs/root")
	assertHas(t, files, "config/php/php-local.ini")
	assertHas(t, files, "config/www/index.html")
}

func TestExportFullIncludeSecretsIncludesDNSAndLetsEncrypt(t *testing.T) {
	t.Parallel()

	swagDir := t.TempDir()
	mustWrite(t, filepath.Join(swagDir, "compose.yaml"), "version: '3'\n")

	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "nginx.conf"), "worker_processes  1;\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "site-confs", "default"), "# default\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "proxy-confs", "a.subdomain.conf"), "# a\n")

	mustWrite(t, filepath.Join(swagDir, "config", "dns-conf", "cloudflare.ini"), "token=secret\n")
	mustWrite(t, filepath.Join(swagDir, "config", "keys", "cert.key"), "PRIVATEKEY\n")
	mustWrite(t, filepath.Join(swagDir, "config", "etc", "letsencrypt", "live", "example", "fullchain.pem"), "CERT\n")

	mustWrite(t, filepath.Join(swagDir, "config", "fail2ban", "jail.local"), "[sshd]\n")
	mustWrite(t, filepath.Join(swagDir, "config", "fail2ban", "fail2ban.sqlite3"), "sqlite\n")

	outZip := filepath.Join(t.TempDir(), "out.zip")
	files := mustExportAndList(t, Options{
		SwagDir:        swagDir,
		OutPath:        outZip,
		Profile:        ProfileFull,
		IncludeSecrets: true,
		ProxyConfOnly:  true,
		Now: func() time.Time {
			return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	})

	assertHas(t, files, "config/dns-conf/cloudflare.ini")
	assertHas(t, files, "config/keys/cert.key")
	assertHas(t, files, "config/etc/letsencrypt/live/example/fullchain.pem")

	assertHas(t, files, "config/fail2ban/jail.local")
	assertNotHas(t, files, "config/fail2ban/fail2ban.sqlite3")
}

func TestExportExcludeGlob(t *testing.T) {
	t.Parallel()

	swagDir := t.TempDir()
	mustWrite(t, filepath.Join(swagDir, "compose.yaml"), "version: '3'\n")

	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "nginx.conf"), "worker_processes  1;\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "site-confs", "default"), "# default\n")
	mustWrite(t, filepath.Join(swagDir, "config", "nginx", "proxy-confs", "a.subdomain.conf"), "# a\n")

	outZip := filepath.Join(t.TempDir(), "out.zip")
	files := mustExportAndList(t, Options{
		SwagDir:       swagDir,
		OutPath:       outZip,
		Profile:       ProfileMinimal,
		ProxyConfOnly: true,
		ExcludeGlobs:  []string{"config/nginx/nginx.conf"},
		Now: func() time.Time {
			return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	})

	assertNotHas(t, files, "config/nginx/nginx.conf")
	assertHas(t, files, "config/nginx/site-confs/default")
}

func mustExportAndList(t *testing.T, opts Options) []string {
	t.Helper()

	if _, err := Export(opts); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	return listZipFiles(t, opts.OutPath)
}

func listZipFiles(t *testing.T, zipPath string) []string {
	t.Helper()

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip error = %v", err)
	}
	defer func() { _ = r.Close() }()

	var names []string
	for _, f := range r.File {
		names = append(names, strings.TrimPrefix(f.Name, "/"))
	}
	sort.Strings(names)
	return names
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file error = %v", err)
	}
}

func assertHas(t *testing.T, files []string, want string) {
	t.Helper()
	for _, f := range files {
		if f == want {
			return
		}
	}
	t.Fatalf("zip missing %s, got: %v", want, files)
}

func assertNotHas(t *testing.T, files []string, unwanted string) {
	t.Helper()
	for _, f := range files {
		if f == unwanted {
			t.Fatalf("zip should not contain %s, got: %v", unwanted, files)
		}
	}
}

