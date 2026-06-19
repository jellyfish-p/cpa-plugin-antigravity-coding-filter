package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPackageLibraryWritesLibraryAsExecutableZipEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	libraryPath := filepath.Join(dir, "antigravity-coding-filter.dll")
	archivePath := filepath.Join(dir, "plugin.zip")
	libraryData := []byte("compiled plugin bytes")

	if err := os.WriteFile(libraryPath, libraryData, 0o644); err != nil {
		t.Fatalf("write library: %v", err)
	}

	archiveData, err := packageLibrary(libraryPath, archivePath)
	if err != nil {
		t.Fatalf("package library: %v", err)
	}
	if len(archiveData) == 0 {
		t.Fatalf("archiveData is empty")
	}

	reader, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("zip entry count = %d, want 1", len(reader.File))
	}

	entry := reader.File[0]
	if entry.Name != "antigravity-coding-filter.dll" {
		t.Fatalf("entry name = %q, want antigravity-coding-filter.dll", entry.Name)
	}
	if mode := entry.Mode().Perm(); mode != 0o755 {
		t.Fatalf("entry mode = %o, want 755", mode)
	}

	entryReader, err := entry.Open()
	if err != nil {
		t.Fatalf("open entry: %v", err)
	}
	defer entryReader.Close()

	var actual bytes.Buffer
	if _, err := actual.ReadFrom(entryReader); err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if !bytes.Equal(actual.Bytes(), libraryData) {
		t.Fatalf("entry content = %q, want %q", actual.Bytes(), libraryData)
	}
}
