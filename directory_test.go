package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncDirs_AddsMissingDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "destination")

	if err := os.MkdirAll(filepath.Join(source, "a", "b"), 0755); err != nil {
		t.Fatalf("не удалось создать исходные директории: %v", err)
	}
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}

	if err := syncDirs(source, destination); err != nil {
		t.Fatalf("syncDirs вернул ошибку: %v", err)
	}

	if info, err := os.Stat(filepath.Join(destination, "a", "b")); err != nil || !info.IsDir() {
		t.Fatalf("ожидалась созданная директория destination/a/b, err=%v", err)
	}
}

func TestSyncDirs_RemovesExcessDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "destination")

	if err := os.MkdirAll(filepath.Join(source, "keep"), 0755); err != nil {
		t.Fatalf("не удалось создать исходные директории: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(destination, "keep"), 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию keep: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(destination, "delete", "nested"), 0755); err != nil {
		t.Fatalf("не удалось создать лишнюю директорию: %v", err)
	}

	if err := syncDirs(source, destination); err != nil {
		t.Fatalf("syncDirs вернул ошибку: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destination, "delete")); !os.IsNotExist(err) {
		t.Fatalf("лишняя директория не была удалена, err=%v", err)
	}
	if info, err := os.Stat(filepath.Join(destination, "keep")); err != nil || !info.IsDir() {
		t.Fatalf("нужная директория keep должна сохраниться, err=%v", err)
	}
}

func TestSyncDirs_CreatesNestedDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "destination")

	if err := os.MkdirAll(filepath.Join(source, "public", "inner"), 0755); err != nil {
		t.Fatalf("не удалось создать вложенную исходную директорию: %v", err)
	}
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}

	if err := syncDirs(source, destination); err != nil {
		t.Fatalf("syncDirs вернул ошибку: %v", err)
	}

	if info, err := os.Stat(filepath.Join(destination, "public", "inner")); err != nil || !info.IsDir() {
		t.Fatalf("ожидалась созданная вложенная директория destination/public/inner, err=%v", err)
	}
}

func TestSyncDirs_ReturnsErrorForMissingSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	destination := filepath.Join(dir, "destination")
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}

	err := syncDirs(filepath.Join(dir, "missing_source"), destination)
	if err == nil {
		t.Fatal("ожидалась ошибка для отсутствующей исходной директории")
	}
	if !strings.Contains(err.Error(), ErrorMessages["directory_not_exists"]) {
		t.Fatalf("ошибка = %q, ожидалось сообщение об отсутствующей директории", err.Error())
	}
}

func TestSyncDirs_ReturnsErrorForFileSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sourceFile := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination")
	if err := os.WriteFile(sourceFile, []byte("file"), 0644); err != nil {
		t.Fatalf("не удалось создать файл-источник: %v", err)
	}
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}

	err := syncDirs(sourceFile, destination)
	if err == nil {
		t.Fatal("ожидалась ошибка для исходного пути, который не является директорией")
	}
	if !strings.Contains(err.Error(), ErrorMessages["path_not_directory"]) {
		t.Fatalf("ошибка = %q, ожидалось сообщение о пути, который не является директорией", err.Error())
	}
}
