package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBlogPostXML(t *testing.T, dir string, filename string, tagID int, date string, title string) {
	t.Helper()

	content := fmt.Sprintf(`<post>
<date>%s</date>
<author>Автор</author>
<tagid>%d</tagid>
<title>%s</title>
<sites></sites>
<annotation>Аннотация</annotation>
<short_annotation>Кратко</short_annotation>
<content>Контент %s</content>
</post>`, date, tagID, title, title)

	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("не удалось создать xml поста: %v", err)
	}
}

func TestLoadTags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tags.xml"), []byte(`<tags><tag id="1" name="Новости"><epigraph>Эпиграф</epigraph></tag></tags>`), 0644); err != nil {
		t.Fatalf("не удалось создать tags.xml: %v", err)
	}

	tags, err := loadTags(dir)
	if err != nil {
		t.Fatalf("loadTags вернул ошибку: %v", err)
	}
	if len(tags.Tags) != 1 || tags.Tags[0].Id != 1 || tags.Tags[0].Name != "Новости" {
		t.Fatalf("рубрики загружены некорректно: %+v", tags.Tags)
	}
	if tags.Tags[0].Posts != 0 {
		t.Fatalf("счётчик постов должен быть обнулён, получено %d", tags.Tags[0].Posts)
	}
}

func TestLoadTags_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := loadTags(t.TempDir())
	if err == nil {
		t.Fatal("ожидалась ошибка при отсутствии tags.xml")
	}
}

func TestLoadBlog_InvalidTagID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	postsDir := filepath.Join(dir, "posts")
	if err := os.MkdirAll(postsDir, 0755); err != nil {
		t.Fatalf("не удалось создать директорию постов: %v", err)
	}
	writeBlogPostXML(t, postsDir, "bad.xml", 999, "01.01.2026", "Плохой tagid")

	tags := &TagsList{Tags: []Tag{{Id: 1, Name: "Новости"}}}
	_, _, err := loadBlog(postsDir, tags)
	if err == nil {
		t.Fatal("ожидалась ошибка для неизвестного tagid")
	}
	if !strings.Contains(err.Error(), "неизвестный tagid=999") {
		t.Fatalf("ошибка = %q, ожидалось сообщение о неизвестном tagid", err.Error())
	}
}

func TestLoadBlog_SortsPostsByDateDescAndCountsTags(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	postsDir := filepath.Join(dir, "posts")
	if err := os.MkdirAll(postsDir, 0755); err != nil {
		t.Fatalf("не удалось создать директорию постов: %v", err)
	}
	writeBlogPostXML(t, postsDir, "old.xml", 1, "01.01.2024", "Старый")
	writeBlogPostXML(t, postsDir, "new.xml", 2, "01.01.2026", "Новый")
	writeBlogPostXML(t, postsDir, "middle.xml", 1, "01.01.2025", "Средний")

	tags := &TagsList{Tags: []Tag{{Id: 1, Name: "Новости"}, {Id: 2, Name: "Разборы"}}}
	posts, total, err := loadBlog(postsDir, tags)
	if err != nil {
		t.Fatalf("loadBlog вернул ошибку: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, ожидалось 3", total)
	}
	if (*posts)[0].Title != "Новый" || (*posts)[1].Title != "Средний" || (*posts)[2].Title != "Старый" {
		t.Fatalf("посты отсортированы некорректно: %+v", *posts)
	}
	if tags.Tags[0].Posts != 2 || tags.Tags[1].Posts != 1 {
		t.Fatalf("счётчики рубрик некорректны: %+v", tags.Tags)
	}
}

func TestWriteBlogFeedPages_DoesNotLoseTenthPost(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "blog.html")
	targetDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}
	if err := os.WriteFile(templatePath, []byte(`{{range .Blog}}{{.Title}}
{{end}}`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон блога: %v", err)
	}

	posts := make([]Post, 11)
	for i := 0; i < 11; i++ {
		posts[i] = Post{Title: fmt.Sprintf("post-%02d", i+1)}
	}

	if err := writeBlogFeedPages(templatePath, "", targetDir, nil, posts, len(posts), 0, 10); err != nil {
		t.Fatalf("writeBlogFeedPages вернул ошибку: %v", err)
	}

	firstPage, err := os.ReadFile(filepath.Join(targetDir, "index.html"))
	if err != nil {
		t.Fatalf("не удалось прочитать первую страницу ленты: %v", err)
	}
	secondPage, err := os.ReadFile(filepath.Join(targetDir, "2.html"))
	if err != nil {
		t.Fatalf("не удалось прочитать вторую страницу ленты: %v", err)
	}

	first := string(firstPage)
	second := string(secondPage)
	if !strings.Contains(first, "post-10") {
		t.Fatalf("десятый пост потерян на первой странице: %q", first)
	}
	if !strings.Contains(second, "post-11") {
		t.Fatalf("одиннадцатый пост должен быть на второй странице: %q", second)
	}
	if strings.Contains(first, "post-11") {
		t.Fatalf("одиннадцатый пост не должен быть на первой странице: %q", first)
	}
}

func TestWriteBlogFeedPages_EmptyBlogCreatesIndex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "blog.html")
	targetDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}
	if err := os.WriteFile(templatePath, []byte(`total={{.Total}} posts={{len .Blog}}`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон блога: %v", err)
	}

	if err := writeBlogFeedPages(templatePath, "", targetDir, nil, nil, 0, 0, 10); err != nil {
		t.Fatalf("writeBlogFeedPages вернул ошибку для пустого блога: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "index.html"))
	if err != nil {
		t.Fatalf("для пустого блога должна быть создана index.html: %v", err)
	}
	if string(content) != "total=0 posts=0" {
		t.Fatalf("содержимое index.html = %q", string(content))
	}
}

func TestWriteBlogFeedPages_InvalidPostsPerPage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templatePath := filepath.Join(dir, "blog.html")
	if err := os.WriteFile(templatePath, []byte(`template`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон блога: %v", err)
	}

	err := writeBlogFeedPages(templatePath, "", dir, nil, nil, 0, 0, 0)
	if err == nil {
		t.Fatal("ожидалась ошибка при postsPerPage <= 0")
	}
}
