// Googol генератор статических html-страниц из шаблонов.
// Модуль работы с публикациями.

package main

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Article описывает публикацию.
type Article struct {
	// Загружаемые из файла поля.
	Title       string `xml:"title"`
	Author      string `xml:"author"`
	Annotation  string `xml:"annotation"`
	Keywords    string `xml:"keywords"`
	Description string `xml:"description"`
	Pages       string `xml:"pages"`

	// Вычисляемые поля.
	Fuseaction string
	Pagetitles []string
	Content    []string
}

// loadArticles загружает список публикаций.
func loadArticles(articlesDir string) (*[]Article, error) {
	var articles []Article

	dir, err := os.Open(articlesDir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.Mode().IsRegular() || filepath.Ext(file.Name()) != ".xml" {
			continue
		}

		raw, err := ioutil.ReadFile(filepath.Join(articlesDir, file.Name()))
		if err != nil {
			return nil, err
		}

		var article Article
		if err = xml.Unmarshal(raw, &article); err != nil {
			return nil, err
		}

		// Уникальный строковый идентификатор публикации.
		article.Fuseaction = strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))

		// Проверяем, существует ли папка публикации в директории публикаций.
		articleDir := filepath.Join(articlesDir, article.Fuseaction)
		articleFolderExists := false
		if stat, err := os.Stat(articleDir); err == nil && stat.IsDir() {
			articleFolderExists = true
		} else if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		// Заголовки и контент страниц.
		for i, title := range strings.Split(article.Pages, "|") {
			article.Pagetitles = append(article.Pagetitles, title)

			content := ""
			if articleFolderExists {
				contentBytes, err := ioutil.ReadFile(filepath.Join(articleDir, strconv.Itoa(i+1)+".html"))
				if err == nil {
					content = string(contentBytes)
				} else if !os.IsNotExist(err) {
					return nil, err
				}
			}

			// Content всегда должен иметь ту же длину, что и Pagetitles.
			article.Content = append(article.Content, content)
		}

		// Добавляем публикацию в список публикаций.
		articles = append(articles, article)
	}

	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Title < articles[j].Title
	})

	return &articles, nil
}

// createArticlesPage создаёт страницу аннотаций статей.
// settingsDir — директория, в которой находятся шаблоны модуля публикаций.
// articlesDir — исходная директория системы публикации статей.
// destinationArticlesDir — целевая директория системы публикации статей.
// templatesPath — директория шаблонов сайта.
// articles — список статей.
// domain — домен сайта.
// sitemap — содержимое файла sitemap.
func createArticlesPage(settingsDir string, articlesDir string, destinationArticlesDir string, templatesPath string, articles *[]Article, domain string, sitemap *string) error {
	// Проверяем, существует ли шаблон страницы списка публикаций.
	articlesTemplate := filepath.Join(settingsDir, "articles.html")
	if _, err := os.Stat(articlesTemplate); os.IsNotExist(err) {
		return errors.New(ErrorMessages["parse_template_error"] + "отсутствует шаблон страницы списка статей.")
	}

	// Данные для парсинга шаблона.
	data := struct {
		Fuseaction string
		Articles   *[]Article
	}{
		"articles.html",
		articles,
	}

	content, err := ParseFileView(articlesTemplate, templatesPath, data, "articles")
	if err != nil {
		return errors.New(ErrorMessages["parse_template_error"] + err.Error())
	}

	if err = ioutil.WriteFile(filepath.Join(destinationArticlesDir, "index.html"), []byte(content), 0755); err != nil {
		return errors.New(ErrorMessages["error_creating_file"] + err.Error())
	}

	// URL страницы на целевом сервере.
	url := domain + "/articles/"
	*sitemap += "<url><loc>" + url + "</loc></url>"

	return nil
}

// createArticleFiles создаёт файлы указанной статьи.
// settingsDir — директория, в которой находятся шаблоны модуля публикаций.
// articlesDir — исходная директория системы публикации статей.
// destinationArticlesDir — целевая директория системы публикации статей.
// templatesPath — директория шаблонов сайта.
// article — статья.
// domain — домен сайта.
// sitemap — содержимое файла sitemap.
func createArticleFiles(settingsDir string, articlesDir string, destinationArticlesDir string, templatesPath string, article Article, domain string, sitemap *string) error {
	// Проверяем, существует ли директория статьи в целевой директории системы публикаций.
	articleDestination := filepath.Join(destinationArticlesDir, article.Fuseaction)
	if _, err := os.Stat(articleDestination); os.IsNotExist(err) {
		// Пытаемся создать директорию статьи.
		if err = os.Mkdir(articleDestination, 0755); err != nil {
			return errors.New(ErrorMessages["error_creating_dir"] + err.Error())
		}
	}

	// Проверяем, существует ли в исходной директории статей шаблон оглавления статьи.
	contentsTemplateEnabled := true
	contentsTemplate := filepath.Join(settingsDir, "article.html")
	if _, err := os.Stat(contentsTemplate); os.IsNotExist(err) {
		contentsTemplateEnabled = false
	} else {
		// Формируем страницу контента статьи.
		data := struct {
			Fuseaction  string
			Title       string
			Annotation  string
			Keywords    string
			Description string
			Pages       []string
		}{
			"article.html",
			article.Title,
			article.Annotation,
			article.Keywords,
			article.Description,
			article.Pagetitles,
		}

		content, err := ParseFileView(contentsTemplate, templatesPath, data, "article.html")
		if err != nil {
			return errors.New(ErrorMessages["parse_template_error"] + err.Error())
		}

		if err = ioutil.WriteFile(filepath.Join(articleDestination, "index.html"), []byte(content), 0755); err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}

		// URL страницы на целевом сервере.
		url := domain + "/articles/" + article.Fuseaction + "/"
		*sitemap += "<url><loc>" + url + "</loc></url>"
	}

	firstPage := "index.html"
	if contentsTemplateEnabled {
		firstPage = "1.html"
	}

	// Формируем файлы страниц.
	pageTemplate := filepath.Join(settingsDir, "page.html")
	if _, err := os.Stat(pageTemplate); os.IsNotExist(err) {
		return errors.New(ErrorMessages["parse_template_error"] + "отсутствует шаблон страницы статьи.")
	}

	pagesNumbers := make([]int, len(article.Pagetitles))
	for k := 0; k < len(article.Pagetitles); k++ {
		pagesNumbers[k] = k
	}

	for i := 0; i < len(article.Pagetitles); i++ {
		pageContent := ""
		if i < len(article.Content) {
			pageContent = article.Content[i]
		}
		if len(pageContent) == 0 {
			continue
		}

		filename := firstPage
		if i > 0 {
			filename = strconv.Itoa(i+1) + ".html"
		}

		nextpageTitle := ""
		nextpageAddress := ""
		if i < len(article.Pagetitles)-1 {
			nextpageTitle = article.Pagetitles[i+1]
			nextpageAddress = strconv.Itoa(i+2) + ".html"
		}

		prevpageTitle := ""
		prevpageAddress := ""
		if i > 0 {
			prevpageTitle = article.Pagetitles[i-1]
			if i == 1 {
				prevpageAddress = firstPage
			} else {
				prevpageAddress = strconv.Itoa(i) + ".html"
			}
		}

		data := struct {
			Fuseaction   string
			Title        string
			Content      string
			Keywords     string
			Description  string
			ThisTitle    string
			NextTitle    string
			NextAddress  string
			PrevTitle    string
			PrevAddress  string
			Pagenum      int
			PagesCount   int
			PagesNumbers []int
		}{
			"page.html",
			article.Title,
			pageContent,
			article.Keywords,
			article.Description,
			article.Pagetitles[i],
			nextpageTitle,
			nextpageAddress,
			prevpageTitle,
			prevpageAddress,
			i,
			len(article.Pagetitles),
			pagesNumbers,
		}

		content, err := ParseFileView(pageTemplate, templatesPath, data, "page.html")
		if err != nil {
			return errors.New(ErrorMessages["parse_template_error"] + err.Error())
		}

		if err = ioutil.WriteFile(filepath.Join(articleDestination, filename), []byte(content), 0755); err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}

		// URL страницы на целевом сервере.
		url := domain + "/articles/" + article.Fuseaction + "/" + filename
		*sitemap += "<url><loc>" + url + "</loc></url>"
	}

	return nil
}

// CreateArticles формирует файлы публикаций.
// settingsDir — директория файлов настроек сайта.
// articlesDir — исходная директория файлов публикаций.
// destinationArticlesDir — целевая директория файлов публикаций.
// templatesDir — директория шаблонов сайта.
// domain — домен сайта.
// sitemap — содержимое файла sitemap.
func CreateArticles(settingsDir string, articlesDir string, destinationArticlesDir string, templatesDir string, domain string, sitemap *string) error {
	articles, err := loadArticles(articlesDir)
	if err != nil {
		return err
	}

	// Если в целевой директории отсутствует папка articles, создаём её.
	if _, err := os.Stat(destinationArticlesDir); os.IsNotExist(err) {
		if err = os.Mkdir(destinationArticlesDir, 0755); err != nil {
			return err
		}
	}

	if err = createArticlesPage(settingsDir, articlesDir, destinationArticlesDir, templatesDir, articles, domain, sitemap); err != nil {
		return err
	}

	// Создаём файлы публикаций.
	for _, article := range *articles {
		if err = createArticleFiles(settingsDir, articlesDir, destinationArticlesDir, templatesDir, article, domain, sitemap); err != nil {
			return err
		}
	}

	return nil
}
