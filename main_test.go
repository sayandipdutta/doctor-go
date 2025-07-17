package main

import (
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/tiff"
)

func TestIsDirectory(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)
	if !IsDirectory(dir) {
		t.Errorf("Expected %s to be a directory", dir)
	}
	file := filepath.Join(dir, "file.txt")
	os.WriteFile(file, []byte("data"), 0644)
	if IsDirectory(file) {
		t.Errorf("Expected %s to not be a directory", file)
	}
}

func TestPrettyFormatMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	out := PrettyFormatMap(m)
	if !strings.Contains(out, "a: 1") || !strings.Contains(out, "b: 2") {
		t.Errorf("PrettyFormatMap output incorrect: %s", out)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	os.WriteFile(src, []byte("hello"), 0644)
	err := CopyFile(src, dst)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "hello" {
		t.Errorf("CopyFile did not copy data correctly")
	}
}

func TestIsDeed(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)
	os.Mkdir(filepath.Join(dir, INDEXED_FOLDER), 0755)
	if !isDeed(dir) {
		t.Errorf("isDeed should return true when QC folder exists")
	}
	dir2 := t.TempDir()
	if isDeed(dir2) {
		t.Errorf("isDeed should return false when QC folder does not exist")
	}
}

func TestIterDeeds(t *testing.T) {
	root := t.TempDir()
	defer os.RemoveAll(root)
	deed := filepath.Join(root, "deed1")
	os.Mkdir(deed, 0755)
	os.Mkdir(filepath.Join(deed, INDEXED_FOLDER), 0755)
	found := false
	for d := range iterDeeds(root) {
		if d == deed {
			found = true
		}
	}
	if !found {
		t.Errorf("iterDeeds did not yield expected deed path")
	}
}

func TestCopyTopsheetPerDeed(t *testing.T) {
	deed := t.TempDir()
	dest := t.TempDir()
	defer os.RemoveAll(deed)
	defer os.RemoveAll(dest)
	qc := filepath.Join(deed, INDEXED_FOLDER)
	scan := filepath.Join(deed, SCANNED_FOLDER)
	os.Mkdir(qc, 0755)
	os.Mkdir(scan, 0755)
	// Create dummy topsheet file
	ts := "A-Others.tif"
	os.WriteFile(filepath.Join(qc, ts), []byte("qcdata"), 0644)
	os.WriteFile(filepath.Join(scan, "A.tif"), []byte("scandata"), 0644)
	wg.Add(1)
	err := CopyTopsheetPerDeed(deed, dest, true, false, false)
	if err != nil {
		t.Errorf("CopyTopsheetPerDeed failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, ts)); err != nil {
		t.Errorf("Expected topsheet not copied: %v", err)
	}
}

func TestZipDoctypes(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("data1"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("data2"), 0644)
	dest := t.TempDir()
	err := zipDoctypes(dir, dest)
	if err != nil {
		t.Errorf("zipDoctypes failed: %v", err)
	}
	zipPath := filepath.Join(dest, filepath.Base(dir)+".zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Errorf("zip file not created: %v", err)
	}
}

func TestImageConv(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "img.tif")
	dst := filepath.Join(dir, "img.jpg")
	// Create dummy TIFF image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := range 10 {
		for y := range 10 {
			img.Set(x, y, color.RGBA{uint8(x * 10), uint8(y * 10), 0, 255})
		}
	}
	f, _ := os.Create(src)
	tiff.Encode(f, img, nil)
	f.Close()
	err := imageConv(src, dst, ".tif", ".jpg")
	if err != nil {
		t.Errorf("imageConv failed: %v", err)
	}
	out, err := os.Open(dst)
	if err != nil {
		t.Errorf("JPEG not created: %v", err)
	}
	defer out.Close()
	buf := make([]byte, 10)
	_, err = out.Read(buf)
	if err != nil && err != io.EOF {
		t.Errorf("Could not read JPEG: %v", err)
	}
}
