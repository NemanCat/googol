// Googol генератор статических html - страниц из шаблонов
// модуль работы с публикациями

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

//класс Публикация
type Article struct {
	//заголовок публикации
	Title string `xml:"title"`
	//автор(ы) публикации
	Author string `xml:"author"`
	//аннотация
	Annotation string `xml:"annotation"`
	//ключевые слова
	Keywords string `xml:"keywords"`
	//description страницы
	Description string `xml:"description"`
	//страницы публикации
	Pages string `xml:"pages"`
	//вычисляемые поля
	//уникальный строковый идентификатор публикации
	Fuseaction string
	//заголовки страниц
	Pagetitles []string
	//контент страниц
	Content []string
}

//-------------------------------------------------------
//функция загрузки списка публикаций
func loadArticles(articles_dir string) (*[]Article, error) {
	var articles []Article
	dir, err := os.Open(articles_dir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	files, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.Mode().IsRegular() {
			if filepath.Ext(file.Name()) == ".xml" {
				raw, err := ioutil.ReadFile(filepath.Join(articles_dir, file.Name()))
				if err != nil {
					return nil, err
				}
				var article Article
				err = xml.Unmarshal(raw, &article)
				if err != nil {
					return nil, err
				}
				//вычисляемые поля
				article_folder_exists := false
				//уникальный строковый идентификатор публикации
				article.Fuseaction = strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
				//проверяем существует ли папка публикации в директории публикаций
				article_dir := filepath.Join(articles_dir, article.Fuseaction)
				if _, err := os.Stat(article_dir); !os.IsNotExist(err) {
					article_folder_exists = true
				}
				//заголовки и контент страниц
				titles := strings.Split(article.Pages, "|")
				for i, title := range titles {
					article.Pagetitles = append(article.Pagetitles, title)
					if article_folder_exists == true {
						content, err := ioutil.ReadFile(filepath.Join(article_dir, strconv.Itoa(i+1)+".html"))
						if err == nil {
							article.Content = append(article.Content, string(content))
						} else {
							article.Content = append(article.Content, "")
						}
					}
				}
				//добавляем в список публикаций
				articles = append(articles, article)
			}
		}
	}
	sort.Slice(articles, func(i, j int) bool { return articles[i].Title < articles[j].Title })
	return &articles, nil
}

//создание страницы аннотаций статей
//settings_dir - директория, в которой находятся шаблоны модуля публикаций
//articlesdir - исходная директория системы публикации статей
//destination_articlesdir - целевая директория системы публикации статей
//templates_path - директория шаблонов сайта
//articles - список статей
//domain - домен сайта
//sitemap - содержимое файла sitemap
func createArticlesPage(settings_dir string, articlesdir string, destination_articlesdir string, templates_path string, articles *[]Article, domain string, sitemap *string) error {
	//проверяем, существует ли шаблон страницы списка публикаций
	articles_template := filepath.Join(settings_dir, "articles.html")
	if _, err := os.Stat(articles_template); os.IsNotExist(err) {
		return errors.New(ErrorMessages["parse_template_error"] + "отсутствует шаблон страницы списка статей.")
	}
	//данные для парсинга шаблона
	data := struct {
		Fuseaction string
		Articles   *[]Article
	}{
		"articles.html",
		articles,
	}
	content, err := ParseFileView(articles_template, templates_path, data, "articles")
	if err != nil {
		return errors.New(ErrorMessages["parse_template_error"] + err.Error())
	}
	err = ioutil.WriteFile(filepath.Join(destination_articlesdir, "index.html"), []byte(content), 0755)
	if err != nil {
		return errors.New(ErrorMessages["error_creating_file"] + err.Error())
	}
	//url старницы на целевом сервере
	url := domain + "/articles/"
	*sitemap += "<url><loc>" + url + "</loc></url>"
	return nil
}

//создание файлов указанной статьи
//settings_dir - директория, в которой находятся шаблоны модуля публикаций
//articlesdir - исходная директория системы публикации статей
//destination_articlesdir - целевая директория системы публикации статей
//templates_path - директория шаблонов сайта
//article - статья
//domain - домен сайта
//sitemap - содержимое файла sitemap
func createArticleFiles(settings_dir string, articlesdir string, destination_articlesdir string, templates_path string, article Article, domain string, sitemap *string) error {
	//проверяем существует ли директория статьи в целевой директории системы публикаций
	article_destination := filepath.Join(destination_articlesdir, article.Fuseaction)
	if _, err := os.Stat(article_destination); os.IsNotExist(err) {
		//пытаемся создать директорию статьи
		err = os.Mkdir(article_destination, 0755)
		if err != nil {
			return errors.New(ErrorMessages["error_creating_dir"] + err.Error())
		}
	}
	//проверяем, существует ли в исходной директории статей шаблон оглавления статьи
	contents_template_enabled := true
	contents_template := filepath.Join(settings_dir, "article.html")
	if _, err := os.Stat(contents_template); os.IsNotExist(err) {
		contents_template_enabled = false
	} else {
		//формируем страницу контента статьи
		//данные для парсинга шаблона
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
		content, err := ParseFileView(contents_template, templates_path, data, "article.html")
		if err != nil {
			return errors.New(ErrorMessages["parse_template_error"] + err.Error())
		}
		err = ioutil.WriteFile(filepath.Join(article_destination, "index.html"), []byte(content), 0755)
		if err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}
		//url старницы на целевом сервере
		url := domain + "/articles/" + article.Fuseaction + "/"
		*sitemap += "<url><loc>" + url + "</loc></url>"

	}
	var first_page string
	if contents_template_enabled == true {
		first_page = "1.html"
	} else {
		first_page = "index.html"
	}
	//формируем файлы страниц
	page_template := filepath.Join(settings_dir, "page.html")
	if _, err := os.Stat(page_template); os.IsNotExist(err) {
		return errors.New(ErrorMessages["parse_template_error"] + "отсутствует шаблон страницы статьи.")
	}
	pages_numbers := make([]int, len(article.Pagetitles))
	for k := 0; k < len(article.Pagetitles); k++ {
		pages_numbers[k] = k
	}
	for i := 0; i < len(article.Pagetitles); i++ {
		if len(article.Content[i]) > 0 {
			var filename string
			var nextpage_title string
			var nextpage_address string
			var prevpage_title string
			var prevpage_address string
			if i == 0 {
				filename = first_page
			} else {
				filename = strconv.Itoa(i+1) + ".html"
			}
			if i == len(article.Pagetitles)-1 {
				nextpage_title = ""
				nextpage_address = ""
			} else {
				nextpage_title = article.Pagetitles[i+1]
				nextpage_address = strconv.Itoa(i+2) + ".html"
			}
			if i == 0 {
				prevpage_title = ""
				prevpage_address = ""
			} else {
				prevpage_title = article.Pagetitles[i-1]
				if i == 1 {
					prevpage_address = first_page
				} else {
					prevpage_address = strconv.Itoa(i) + ".html"
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
				article.Content[i],
				article.Keywords,
				article.Description,
				article.Pagetitles[i],
				nextpage_title,
				nextpage_address,
				prevpage_title,
				prevpage_address,
				i,
				len(article.Pagetitles),
				pages_numbers,
			}

			content, err := ParseFileView(page_template, templates_path, data, "page.html")
			if err != nil {
				return errors.New(ErrorMessages["parse_template_error"] + err.Error())
			}
			err = ioutil.WriteFile(filepath.Join(article_destination, filename), []byte(content), 0755)
			if err != nil {
				return errors.New(ErrorMessages["error_creating_file"] + err.Error())
			}
			//url старницы на целевом сервере
			url := domain + "/articles/" + article.Fuseaction + "/" + filename
			*sitemap += "<url><loc>" + url + "</loc></url>"
		}

	}
	return nil
}

//формирование файлов публикаций
//settings_dir - директория файлов настроек сайта
//articles_dir - исходная директория файлов публикаций
//destination_articlesdir - целевая директория файлов публикаций
//templates_dir - директория шаблонов сайта
//domain - домен сайта
//sitemap - содержимое файла sitemap
func CreateArticles(settings_dir string, articles_dir string, destination_articlesdir string, templates_dir string, domain string, sitemap *string) error {
	var articles *[]Article = nil
	articles, err := loadArticles(articles_dir)
	if err != nil {
		return err
	}
	//если в целевой директории отсутствует папка articles - создаём её
	if _, err := os.Stat(destination_articlesdir); os.IsNotExist(err) {
		//создаём поддиректорию
		err = os.Mkdir(destination_articlesdir, 0755)
		if err != nil {
			return err
		}
	}
	//
	err = createArticlesPage(settings_dir, articles_dir, destination_articlesdir, templates_dir, articles, domain, sitemap)
	if err != nil {
		return err
	}
	//создаём файлы публикаций
	for _, article := range *articles {
		err = createArticleFiles(settings_dir, articles_dir, destination_articlesdir, templates_dir, article, domain, sitemap)
		if err != nil {
			return err
		}
	}
	return nil
}
