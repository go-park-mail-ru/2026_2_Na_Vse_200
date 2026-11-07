package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalSaveOpenRemove(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocal(root)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}

	key := "images/avatars/test.png"
	if err := store.Save(key, strings.NewReader("png-bytes")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	file, err := store.Open(key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	buf := make([]byte, 32)
	n, err := file.Read(buf)
	if err != nil || string(buf[:n]) != "png-bytes" {
		file.Close()
		t.Fatalf("Read: %q %v", buf[:n], err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(key))); err != nil {
		t.Fatalf("file missing: %v", err)
	}

	if err := store.Remove(key); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	if err := store.Save("../secret.txt", strings.NewReader("x")); err == nil {
		t.Fatal("ожидалась ошибка для ..")
	}
}
