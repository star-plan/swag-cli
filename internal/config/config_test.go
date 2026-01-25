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

