package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeArticleXML(t *testing.T, dir string, filename string, title string, pages string) {
	t.Helper()

	content := `<article>
<title>` + title + `</title>
<author>Автор</author>
<annotation>Аннотация</annotation>
<keywords>ключи</keywords>
<description>описание</description>
<pages>` + pages + `</pages>
</article>`

	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("не удалось создать xml статьи: %v", err)
	}
}

func TestLoadArticles_LoadsContentForEachPage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeArticleXML(t, dir, "test.xml", "Тестовая статья", "Первая|Вторая")
	articleDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(articleDir, 0755); err != nil {
		t.Fatalf("не удалось создать директорию статьи: %v", err)
	}
	if err := os.WriteFile(filepath.Join(articleDir, "1.html"), []byte("Контент 1"), 0644); err != nil {
		t.Fatalf("не удалось создать первую страницу статьи: %v", err)
	}
	if err := os.WriteFile(filepath.Join(articleDir, "2.html"), []byte("Контент 2"), 0644); err != nil {
		t.Fatalf("не удалось создать вторую страницу статьи: %v", err)
	}

	articles, err := loadArticles(dir)
	if err != nil {
		t.Fatalf("loadArticles вернул ошибку: %v", err)
	}
	if len(*articles) != 1 {
		t.Fatalf("количество статей = %d, ожидалось 1", len(*articles))
	}
	article := (*articles)[0]
	if article.Fuseaction != "test" {
		t.Fatalf("Fuseaction = %q, ожидалось test", article.Fuseaction)
	}
	if len(article.Pagetitles) != 2 || len(article.Content) != 2 {
		t.Fatalf("Pagetitles и Content должны иметь длину 2, получено %d и %d", len(article.Pagetitles), len(article.Content))
	}
	if article.Content[0] != "Контент 1" || article.Content[1] != "Контент 2" {
		t.Fatalf("контент страниц загружен некорректно: %+v", article.Content)
	}
}

func TestLoadArticles_MissingContentFileKeepsContentLength(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeArticleXML(t, dir, "test.xml", "Тестовая статья", "Первая|Вторая")
	articleDir := filepath.Join(dir, "test")
	if err := os.MkdirAll(articleDir, 0755); err != nil {
		t.Fatalf("не удалось создать директорию статьи: %v", err)
	}
	if err := os.WriteFile(filepath.Join(articleDir, "1.html"), []byte("Контент 1"), 0644); err != nil {
		t.Fatalf("не удалось создать первую страницу статьи: %v", err)
	}

	articles, err := loadArticles(dir)
	if err != nil {
		t.Fatalf("loadArticles вернул ошибку: %v", err)
	}
	article := (*articles)[0]
	if len(article.Pagetitles) != 2 || len(article.Content) != 2 {
		t.Fatalf("Content должен иметь ту же длину, что и Pagetitles, получено %d и %d", len(article.Content), len(article.Pagetitles))
	}
	if article.Content[0] != "Контент 1" || article.Content[1] != "" {
		t.Fatalf("контент страниц загружен некорректно: %+v", article.Content)
	}
}

func TestCreateArticleFiles_ContentShorterThanPageTitlesDoesNotPanic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	settingsDir := filepath.Join(dir, "settings")
	destinationDir := filepath.Join(dir, "dest")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("не удалось создать директорию настроек: %v", err)
	}
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "page.html"), []byte(`{{.Title}}:{{.ThisTitle}}:{{.Content}}`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон страницы: %v", err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "article.html"), []byte(`{{.Title}} {{range .Pages}}{{.}} {{end}}`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон оглавления: %v", err)
	}

	article := Article{
		Title:      "Статья",
		Fuseaction: "article-1",
		Pagetitles: []string{"Первая", "Вторая"},
		Content:    []string{"Контент первой страницы"},
	}
	sitemap := ""

	if err := createArticleFiles(settingsDir, "", destinationDir, "", article, "https://example.test", &sitemap); err != nil {
		t.Fatalf("createArticleFiles вернул ошибку: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destinationDir, "article-1", "1.html")); err != nil {
		t.Fatalf("первая страница статьи должна быть создана: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destinationDir, "article-1", "2.html")); !os.IsNotExist(err) {
		t.Fatalf("вторая страница с отсутствующим контентом не должна создаваться, err=%v", err)
	}
}

func TestCreateArticleFiles_WithoutContentsTemplateUsesIndexForFirstPage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	settingsDir := filepath.Join(dir, "settings")
	destinationDir := filepath.Join(dir, "dest")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("не удалось создать директорию настроек: %v", err)
	}
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "page.html"), []byte(`{{.Content}}`), 0644); err != nil {
		t.Fatalf("не удалось создать шаблон страницы: %v", err)
	}

	article := Article{
		Title:      "Статья",
		Fuseaction: "article-1",
		Pagetitles: []string{"Первая"},
		Content:    []string{"Контент первой страницы"},
	}
	sitemap := ""

	if err := createArticleFiles(settingsDir, "", destinationDir, "", article, "https://example.test", &sitemap); err != nil {
		t.Fatalf("createArticleFiles вернул ошибку: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destinationDir, "article-1", "index.html"))
	if err != nil {
		t.Fatalf("первая страница должна быть создана как index.html: %v", err)
	}
	if string(content) != "Контент первой страницы" {
		t.Fatalf("содержимое первой страницы = %q", string(content))
	}
}

func TestCreateArticleFiles_MissingPageTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	settingsDir := filepath.Join(dir, "settings")
	destinationDir := filepath.Join(dir, "dest")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("не удалось создать директорию настроек: %v", err)
	}
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		t.Fatalf("не удалось создать целевую директорию: %v", err)
	}

	article := Article{Title: "Статья", Fuseaction: "article-1", Pagetitles: []string{"Первая"}, Content: []string{"Контент"}}
	sitemap := ""
	err := createArticleFiles(settingsDir, "", destinationDir, "", article, "https://example.test", &sitemap)
	if err == nil {
		t.Fatal("ожидалась ошибка при отсутствии шаблона page.html")
	}
}
