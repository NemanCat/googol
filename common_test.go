package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	dest := filepath.Join(dir, "dest.txt")

	if err := os.WriteFile(source, []byte("test content"), 0640); err != nil {
		t.Fatalf("не удалось создать исходный файл: %v", err)
	}

	if err := CopyFile(source, dest); err != nil {
		t.Fatalf("CopyFile вернул ошибку: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("не удалось прочитать целевой файл: %v", err)
	}
	if string(content) != "test content" {
		t.Fatalf("содержимое целевого файла = %q, ожидалось %q", string(content), "test content")
	}
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "missing.txt"), filepath.Join(dir, "dest.txt"))
	if err == nil {
		t.Fatal("ожидалась ошибка при копировании отсутствующего файла")
	}
}

func TestHashFileCrc32WithError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "data.txt")
	content := "crc content"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("не удалось создать файл: %v", err)
	}

	hash, err := HashFileCrc32WithError(file)
	if err != nil {
		t.Fatalf("HashFileCrc32WithError вернул ошибку: %v", err)
	}
	if hash != HashStringCrc32(content) {
		t.Fatalf("хэш файла = %q, ожидалось %q", hash, HashStringCrc32(content))
	}
}

func TestHashFileCrc32WithError_FileNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hash, err := HashFileCrc32WithError(filepath.Join(dir, "missing.txt"))
	if err == nil {
		t.Fatal("ожидалась ошибка при вычислении хэша отсутствующего файла")
	}
	if hash != "" {
		t.Fatalf("хэш отсутствующего файла = %q, ожидалась пустая строка", hash)
	}
}

func TestHashFileCrc32_BackwardCompatibilityOnMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if hash := HashFileCrc32(filepath.Join(dir, "missing.txt")); hash != "" {
		t.Fatalf("HashFileCrc32 для отсутствующего файла = %q, ожидалась пустая строка", hash)
	}
}

func TestParseFileView(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte(`Hello {{.Name}} {{Inc .Number}} {{First .Word}}`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон: %v", err)
	}

	data := struct {
		Name   string
		Number int
		Word   string
	}{"Googol", 41, "Тест"}

	result, err := ParseFileView(page, "", data, "page.html")
	if err != nil {
		t.Fatalf("ParseFileView вернул ошибку: %v", err)
	}
	if result != "Hello Googol 42 Т" {
		t.Fatalf("результат шаблона = %q, ожидалось %q", result, "Hello Googol 42 Т")
	}
}

func TestParseFileView_MissingPage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := ParseFileView(filepath.Join(dir, "missing.html"), "", nil, "missing.html")
	if err == nil {
		t.Fatal("ожидалась ошибка при парсинге отсутствующего файла страницы")
	}
}

func TestParseFileView_MissingTemplatesDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte(`content`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон: %v", err)
	}

	_, err := ParseFileView(page, filepath.Join(dir, "missing_templates"), nil, "page.html")
	if err == nil {
		t.Fatal("ожидалась ошибка при парсинге отсутствующей директории дополнительных шаблонов")
	}
}

func TestIsStringInList(t *testing.T) {
	t.Parallel()

	list := []string{"one", "two", "three"}
	if !IsStringInList("two", list) {
		t.Fatal("строка two должна быть найдена в списке")
	}
	if IsStringInList("four", list) {
		t.Fatal("строка four не должна быть найдена в списке")
	}
}
