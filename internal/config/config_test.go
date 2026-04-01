package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveToLoadFromRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")

	want := Config{
		SwagDir:       "C:\\swag",
		SwagContainer: "swag-prod",
		Network:       "swag-net",
	}

	if err := SaveTo(p, want); err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}

	got, err := LoadFrom(p)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if got != normalize(want) {
		t.Fatalf("round-trip mismatch: got=%+v want=%+v", got, normalize(want))
	}
}

func TestLoadFromNotExistReturnsDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "not-exist.json")

	got, err := LoadFrom(p)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if got != Default() {
		t.Fatalf("LoadFrom() should return Default() when missing file: got=%+v", got)
	}
}

func TestImportFromEmptyFileReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "empty.json")

	if err := os.WriteFile(p, []byte("   \n\t"), 0o644); err != nil {
		t.Fatalf("write empty file error = %v", err)
	}

	if _, err := ImportFrom(p); err == nil {
		t.Fatalf("ImportFrom() should error for empty file")
	}
}

func TestExportToImportFromRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "export.json")

	want := Config{
		SwagDir:       "/data/swag",
		SwagContainer: "swag",
		Network:       "swag",
	}

	if err := ExportTo(p, want, true); err != nil {
		t.Fatalf("ExportTo() error = %v", err)
	}

	got, err := ImportFrom(p)
	if err != nil {
		t.Fatalf("ImportFrom() error = %v", err)
	}

	if got != normalize(want) {
		t.Fatalf("round-trip mismatch: got=%+v want=%+v", got, normalize(want))
	}
}

func TestDefaultSiteConfPathPrefersSiteConfs(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	primary := filepath.Join(baseDir, "config", "nginx", "site-confs", "default")
	legacy := filepath.Join(baseDir, "config", "nginx", "site-conf", "default")
	if err := os.MkdirAll(filepath.Dir(primary), 0o755); err != nil {
		t.Fatalf("mkdir primary error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("mkdir legacy error = %v", err)
	}
	if err := os.WriteFile(primary, []byte("primary"), 0o644); err != nil {
		t.Fatalf("write primary error = %v", err)
	}
	if err := os.WriteFile(legacy, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write legacy error = %v", err)
	}

	cfg := Config{SwagDir: baseDir}
	got, err := cfg.DefaultSiteConfPath()
	if err != nil {
		t.Fatalf("DefaultSiteConfPath() error = %v", err)
	}
	if got != primary {
		t.Fatalf("DefaultSiteConfPath() = %s, want %s", got, primary)
	}
}

func TestDefaultSiteConfPathFallsBackToLegacySiteConf(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	legacy := filepath.Join(baseDir, "config", "nginx", "site-conf", "default")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("mkdir legacy error = %v", err)
	}
	if err := os.WriteFile(legacy, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write legacy error = %v", err)
	}

	cfg := Config{SwagDir: baseDir}
	got, err := cfg.DefaultSiteConfPath()
	if err != nil {
		t.Fatalf("DefaultSiteConfPath() error = %v", err)
	}
	if got != legacy {
		t.Fatalf("DefaultSiteConfPath() = %s, want %s", got, legacy)
	}
}

func TestSetAndGetNormalizeKeysAndValues(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if err := Set(&cfg, " SwAg-CoNtAiNeR ", "  my-swag  "); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, ok := Get(cfg, "swag-container")
	if !ok {
		t.Fatalf("Get() should recognize normalized key")
	}
	if got != "my-swag" {
		t.Fatalf("Get() = %q, want %q", got, "my-swag")
	}
}

func TestProxyConfsDirExpandsHomePath(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	cfg := Config{SwagDir: "~"}
	got := cfg.ProxyConfsDir()
	want := filepath.Join(home, "config", "nginx", "proxy-confs")
	if got != want {
		t.Fatalf("ProxyConfsDir() = %s, want %s", got, want)
	}
}
