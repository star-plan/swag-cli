package tui

import (
	"testing"

	"swag-cli/internal/config"
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
