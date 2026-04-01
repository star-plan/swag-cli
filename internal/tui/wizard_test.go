package tui

import (
	"testing"

	"swag-cli/internal/config"
	"swag-cli/internal/nginx"
)

func TestMergeRuntimeConfigUsesPersistedValuesWhenNoOverrides(t *testing.T) {
	t.Parallel()

	base := config.Config{
		SwagDir:       "/persisted/swag",
		SwagContainer: "persisted-swag",
		Network:       "persisted-net",
	}

	got := mergeRuntimeConfig(base, "", "", "")

	if got != base {
		t.Fatalf("mergeRuntimeConfig() mismatch: got=%+v want=%+v", got, base)
	}
}

func TestMergeRuntimeConfigPrefersRuntimeOverrides(t *testing.T) {
	t.Parallel()

	base := config.Config{
		SwagDir:       "/persisted/swag",
		SwagContainer: "persisted-swag",
		Network:       "persisted-net",
	}

	got := mergeRuntimeConfig(base, "  /runtime/swag  ", " runtime-swag ", " runtime-net ")

	want := config.Config{
		SwagDir:       "/runtime/swag",
		SwagContainer: "runtime-swag",
		Network:       "runtime-net",
	}

	if got != want {
		t.Fatalf("mergeRuntimeConfig() mismatch: got=%+v want=%+v", got, want)
	}
}

func TestHomepageMenuOptionsIncludeClear(t *testing.T) {
	t.Parallel()

	got := homepageMenuOptions()
	if len(got) != 3 || got[1] != "清理主页" {
		t.Fatalf("homepageMenuOptions() = %v", got)
	}
}

func TestSiteActionOptionsProtectHomepage(t *testing.T) {
	t.Parallel()

	got := siteActionOptions(nginx.SiteConfig{Type: nginx.TypeHomepage})
	if len(got) != 1 || got[0] != "返回" {
		t.Fatalf("siteActionOptions(homepage) = %v", got)
	}
}

func TestMainMenuOptionsExposeReloadAndExport(t *testing.T) {
	t.Parallel()

	got := mainMenuOptions()
	wantReload := false
	wantExport := false
	for _, item := range got {
		if item == "重启 SWAG" {
			wantReload = true
		}
		if item == "导出 SWAG 配置" {
			wantExport = true
		}
	}
	if !wantReload || !wantExport {
		t.Fatalf("mainMenuOptions() = %v", got)
	}
}
