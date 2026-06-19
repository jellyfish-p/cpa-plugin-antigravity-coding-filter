package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	libraryPath := flag.String("library", "", "path to the compiled plugin library")
	archivePath := flag.String("archive", "", "path to the output zip archive")
	checksumPath := flag.String("checksum", "", "path to the output checksum file")
	flag.Parse()

	if *libraryPath == "" || *archivePath == "" || *checksumPath == "" {
		fatalf("library, archive, and checksum are required")
	}

	archiveData, err := packageLibrary(*libraryPath, *archivePath)
	if err != nil {
		fatalf("%v", err)
	}

	checksum := sha256.Sum256(archiveData)
	line := fmt.Sprintf("%s  %s\n", hex.EncodeToString(checksum[:]), filepath.Base(*archivePath))
	if err := os.WriteFile(*checksumPath, []byte(line), 0o644); err != nil {
		fatalf("write checksum: %v", err)
	}
}

func packageLibrary(libraryPath, archivePath string) ([]byte, error) {
	library, err := os.Open(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("open library: %w", err)
	}
	defer func() {
		if err := library.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close library: %v\n", err)
		}
	}()

	info, err := library.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat library: %w", err)
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return nil, fmt.Errorf("create archive: %w", err)
	}
	archiveClosed := false
	defer func() {
		if !archiveClosed {
			if err := archive.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "close archive: %v\n", err)
			}
		}
	}()

	writer := zip.NewWriter(archive)
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return nil, fmt.Errorf("create zip header: %w", err)
	}
	header.Name = filepath.Base(libraryPath)
	header.Method = zip.Deflate
	header.SetMode(0o755)

	entry, err := writer.CreateHeader(header)
	if err != nil {
		return nil, fmt.Errorf("create zip entry: %w", err)
	}
	if _, err := io.Copy(entry, library); err != nil {
		return nil, fmt.Errorf("copy library: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close zip writer: %w", err)
	}
	if err := archive.Close(); err != nil {
		return nil, fmt.Errorf("close archive: %w", err)
	}
	archiveClosed = true

	data, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}
	return data, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
