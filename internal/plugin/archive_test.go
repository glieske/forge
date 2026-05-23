package plugin

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractRejectsTarTraversal(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	data := tarGzWithEntry(t, "../outside", []byte("bad"))

	if err := Extract("plugin.tar.gz", data, dest); err == nil {
		t.Fatal("expected traversal archive to be rejected")
	}
	if _, err := os.Stat(filepath.Join(root, "outside")); !os.IsNotExist(err) {
		t.Fatalf("outside file was created or stat failed: %v", err)
	}
}

func TestExtractRejectsZipTraversal(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	data := zipWithEntry(t, "..\\outside", []byte("bad"))

	if err := Extract("plugin.zip", data, dest); err == nil {
		t.Fatal("expected traversal archive to be rejected")
	}
	if _, err := os.Stat(filepath.Join(root, "outside")); !os.IsNotExist(err) {
		t.Fatalf("outside file was created or stat failed: %v", err)
	}
}

func TestExtractAllowsNestedFiles(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "dest")
	data := tarGzWithEntry(t, "bin/forge-plugin", []byte("ok"))

	if err := Extract("plugin.tar.gz", data, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "bin", "forge-plugin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ok" {
		t.Fatalf("content = %q", got)
	}
}

func tarGzWithEntry(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipWithEntry(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
