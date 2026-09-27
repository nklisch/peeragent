package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run zip-windows.go <binary> <archive.zip>")
		os.Exit(2)
	}
	if err := writeArchive(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func writeArchive(source, destination string) error {
	binary, err := os.Open(source)
	if err != nil {
		return err
	}
	defer binary.Close()
	archive, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer archive.Close()
	writer := zip.NewWriter(archive)
	header := &zip.FileHeader{Name: filepath.Base(source), Method: zip.Deflate}
	header.SetMode(0o755)
	header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := io.Copy(entry, binary); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return archive.Close()
}
