// Googol генератор статических html-страниц из шаблонов.
// Модуль работы с блогом.

package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Tag описывает рубрику блога.
type Tag struct {
	XMLName  xml.Name `xml:"tag"`
	Id       int      `xml:"id,attr"`
	Name     string   `xml:"name,attr"`
	Epigraph string   `xml:"epigraph"`
	Posts    int
}

// TagsList описывает список рубрик блога.
type TagsList struct {
	XMLName xml.Name `xml:"tags"`
	Tags    []Tag    `xml:"tag"`
}

// Post описывает пост блога.
type Post struct {
	// Загружаемые из файла поля.
	Date             string `xml:"date"`
	Author           string `xml:"author"`
	Tagid            int    `xml:"tagid"`
	Title            string `xml:"title"`
	Sites            string `xml:"sites"`
	Annotation       string `xml:"annotation"`
	Short_annotation string `xml:"short_annotation"`
	Content          string `xml:"content"`

	// Вычисляемые поля.
	Fuseaction string
	Tag        string
	SortDate   time.Time
	Day        int
	Year       int
	Month      string
}

// SortedBlogPostList используется для сортировки списка постов по дате.
type SortedBlogPostList []Post

func (p SortedBlogPostList) Len() int           { return len(p) }
func (p SortedBlogPostList) Less(i, j int) bool { return p[i].SortDate.Before(p[j].SortDate) }
func (p SortedBlogPostList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// blogPageData описывает данные для шаблона страницы ленты блога.
type blogPageData struct {
	Fuseaction     string
	Tags           []Tag
	Blog           SortedBlogPostList
	Pagenum        int
	Next_page      int
	Total          int
	Tagid          int
	Posts_per_page int
}

// loadTags загружает список рубрик блога.
// settingsDir — директория файлов настроек сайта.
func loadTags(settingsDir string) (*TagsList, error) {
	var tags TagsList

	// Проверяем, существует ли в директории настроек файл со списком рубрик блога tags.xml.
	tagsListFile := filepath.Join(settingsDir, "tags.xml")
	if _, err := os.Stat(tagsListFile); os.IsNotExist(err) {
		return nil, errors.New("Не найден файл списка рубрик блога")
	}

	// Загружаем список рубрик блога.
	raw, err := ioutil.ReadFile(tagsListFile)
	if err != nil {
		return nil, err
	}
	if err = xml.Unmarshal(raw, &tags); err != nil {
		return nil, err
	}

	for i := range tags.Tags {
		tags.Tags[i].Posts = 0
	}

	return &tags, nil
}

// findTagByID ищет рубрику по id и возвращает указатель на неё.
func findTagByID(tags *TagsList, id int) *Tag {
	if tags == nil {
		return nil
	}

	for i := range tags.Tags {
		if tags.Tags[i].Id == id {
			return &tags.Tags[i]
		}
	}

	return nil
}

// handleBlogFiles обходит поддиректории и обрабатывает xml-файлы постов блога.
func handleBlogFiles(posts *SortedBlogPostList, tags *TagsList, totalPosts *int) filepath.WalkFunc {
	return func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() {
			return nil
		}

		_, filename := filepath.Split(currentPath)
		if filepath.Ext(filename) != ".xml" {
			return nil
		}

		raw, err := ioutil.ReadFile(currentPath)
		if err != nil {
			return err
		}

		var post Post
		if err = xml.Unmarshal(raw, &post); err != nil {
			return err
		}

		tag := findTagByID(tags, post.Tagid)
		if tag == nil {
			return fmt.Errorf("пост %s содержит неизвестный tagid=%d", currentPath, post.Tagid)
		}

		post.SortDate, err = time.Parse("02.01.2006", post.Date)
		if err != nil {
			return fmt.Errorf("пост %s содержит некорректную дату %q: %w", currentPath, post.Date, err)
		}

		*totalPosts++
		tag.Posts++

		// Уникальный строковый идентификатор поста.
		post.Fuseaction = strings.TrimSuffix(filename, filepath.Ext(filename))
		// Название рубрики поста.
		post.Tag = tag.Name
		// Поля даты для шаблонов.
		post.Day = post.SortDate.Day()
		post.Month = RussianMonth[post.SortDate.Month()]
		post.Year = post.SortDate.Year()

		// Добавляем пост в список постов блога.
		*posts = append(*posts, post)

		return nil
	}
}

// loadBlog загружает список постов блога.
// postsSourceDir — исходная директория постов блога.
// tags — список рубрик блога.
func loadBlog(postsSourceDir string, tags *TagsList) (*SortedBlogPostList, int, error) {
	var posts SortedBlogPostList
	totalPosts := 0

	if _, err := os.Stat(postsSourceDir); err == nil {
		// Обрабатываем все файлы с расширением xml из директории блога и её поддиректорий.
		err = filepath.Walk(postsSourceDir, handleBlogFiles(&posts, tags, &totalPosts))
		if err != nil {
			return nil, 0, err
		}
	} else if !os.IsNotExist(err) {
		return nil, 0, err
	}

	// Сортируем список постов блога.
	sort.Sort(sort.Reverse(posts))

	return &posts, totalPosts, nil
}

// writeBlogFeedPages формирует страницы ленты блога.
func writeBlogFeedPages(blogTemplatePath string, templatesDir string, targetDir string, activeTags []Tag, posts []Post, totalPosts int, tagID int, postsPerPage int) error {
	if postsPerPage <= 0 {
		return errors.New("количество постов на страницу должно быть больше нуля")
	}

	currentPage := 1
	for start := 0; start < len(posts) || start == 0; start += postsPerPage {
		end := start + postsPerPage
		if end > len(posts) {
			end = len(posts)
		}

		nextPage := 0
		if end < len(posts) {
			nextPage = 1
		}

		data := blogPageData{
			Fuseaction:     "blog.html",
			Tags:           activeTags,
			Blog:           SortedBlogPostList(posts[start:end]),
			Pagenum:        currentPage,
			Next_page:      nextPage,
			Total:          totalPosts,
			Tagid:          tagID,
			Posts_per_page: postsPerPage,
		}

		content, err := ParseFileView(blogTemplatePath, templatesDir, data, "blog.html")
		if err != nil {
			return errors.New(ErrorMessages["parse_template_error"] + err.Error())
		}

		filename := "index.html"
		if currentPage > 1 {
			filename = strconv.Itoa(currentPage) + ".html"
		}

		if err = ioutil.WriteFile(filepath.Join(targetDir, filename), []byte(content), 0755); err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}

		currentPage++
		if end == len(posts) {
			break
		}
	}

	return nil
}

// CreateBlog формирует файлы блога.
// settingsDir — директория файлов настроек сайта.
// destinationBlogDir — целевая директория файлов публикаций.
// postsSourceDir — исходная директория постов блога.
// templatesDir — директория шаблонов сайта.
// domain — домен сайта.
// sitemap — содержимое файла sitemap.
func CreateBlog(settingsDir string, destinationBlogDir string, postsSourceDir string, templatesDir string, domain string, sitemap *string) error {
	// Количество постов блога на страницу.
	postsPerPage := 10

	// Загружаем список рубрик блога.
	tags, err := loadTags(settingsDir)
	if err != nil {
		return err
	}

	// Если в целевой директории отсутствует папка блога blog, создаём её.
	if _, err := os.Stat(destinationBlogDir); os.IsNotExist(err) {
		if err = os.Mkdir(destinationBlogDir, 0755); err != nil {
			return err
		}
	}

	// Если в целевой папке блога отсутствует папка постов posts, создаём её.
	postsDir := filepath.Join(destinationBlogDir, "posts")
	if _, err := os.Stat(postsDir); os.IsNotExist(err) {
		if err = os.Mkdir(postsDir, 0755); err != nil {
			return err
		}
	}

	// Загружаем список постов блога.
	posts, totalPosts, err := loadBlog(postsSourceDir, tags)
	if err != nil {
		return err
	}

	// Формируем список рубрик блога, в которых есть посты.
	activeTags := []Tag{}
	for _, value := range tags.Tags {
		if value.Posts > 0 {
			activeTags = append(activeTags, value)
		}
	}

	// Проверяем наличие в директории настроек сайта шаблона ленты блога blog.html.
	blogTemplatePath := filepath.Join(settingsDir, "blog.html")
	if _, err := os.Stat(blogTemplatePath); os.IsNotExist(err) {
		return errors.New("Не найден файл шаблона ленты блога")
	}

	// Формируем ленту блога без фильтрации.
	if err = writeBlogFeedPages(blogTemplatePath, templatesDir, destinationBlogDir, activeTags, []Post(*posts), totalPosts, 0, postsPerPage); err != nil {
		return err
	}

	// Для всех активных рубрик блога формируем собственный файл ленты.
	tagPosts := make(map[string][]Post)
	for _, value := range activeTags {
		tagPosts[value.Name] = []Post{}
	}
	for _, value := range *posts {
		tagPosts[value.Tag] = append(tagPosts[value.Tag], value)
	}

	for _, value := range activeTags {
		targetDir := filepath.Join(destinationBlogDir, strconv.Itoa(value.Id))
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			if err = os.Mkdir(targetDir, 0755); err != nil {
				return err
			}
		}

		if err = writeBlogFeedPages(blogTemplatePath, templatesDir, targetDir, activeTags, tagPosts[value.Name], len(tagPosts[value.Name]), value.Id, postsPerPage); err != nil {
			return err
		}
	}

	// Формируем страницы постов блога.
	postTemplatePath := filepath.Join(settingsDir, "post.html")
	if _, err := os.Stat(postTemplatePath); os.IsNotExist(err) {
		return errors.New("Не найден файл шаблона поста блога")
	}

	for _, value := range *posts {
		data := struct {
			Fuseaction string
			Tags       []Tag
			Blogpost   Post
			Total      int
		}{
			"post.html",
			activeTags,
			value,
			totalPosts,
		}

		content, err := ParseFileView(postTemplatePath, templatesDir, data, "post.html")
		if err != nil {
			return errors.New(ErrorMessages["parse_template_error"] + err.Error())
		}

		if err = ioutil.WriteFile(filepath.Join(postsDir, value.Fuseaction+".html"), []byte(content), 0755); err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}

		// URL страницы поста блога на целевом сервере.
		url := domain + "/blog/posts/" + value.Fuseaction + ".html"
		// Добавляем страницу в sitemap.xml.
		*sitemap += "<url><loc>" + url + "</loc></url>"
	}

	return nil
}
