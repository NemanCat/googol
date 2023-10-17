// Googol генератор статических html - страниц из шаблонов
// модуль работы с блогом

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
	"time"
)

//класс Рубрика блога
type Tag struct {
	XMLName  xml.Name `xml:"tag"`
	Id       int      `xml:"id,attr"`
	Name     string   `xml:"name,attr"`
	Epigraph string   `xml:"epigraph"`
	Posts    int
	//Tagid    int
}

//структура Список рубрик блога
type TagsList struct {
	XMLName xml.Name `xml:"tags"`
	Tags    []Tag    `xml:"tag"`
}

//структура поста блога
type Post struct {
	//------------------------
	//загружаемые из файла поля
	//дата поста
	Date string `xml:"date"`
	//ФИО автора
	Author string `xml:"author"`
	//id рубрики
	Tagid int `xml:"tagid"`
	//заголовок поста
	Title string `xml:"title"`
	//список id сайтов
	Sites string `xml:"sites"`
	//аннотация поста
	Annotation string `xml:"annotation"`
	//краткая аннотация поста
	Short_annotation string `xml:"short_annotation"`
	//контент поста
	Content string `xml:"content"`
	//-----------------------------
	//вычисляемые поля
	//уникальный строковый идентификатор поста
	Fuseaction string
	//название рубрики поста
	Tag string
	//дата для сортировки списка
	SortDate  time.Time
	Day, Year int
	Month     string
}

//сортировка списка постов по дате
type SortedBlogPostList []Post

func (p SortedBlogPostList) Len() int {
	return len(p)
}

func (p SortedBlogPostList) Less(i, j int) bool {
	return p[i].SortDate.Before(p[j].SortDate)
}

func (p SortedBlogPostList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

//-------------------------------------------------------------
//функция загрузки списка рубрик блога
//settings_dir - директория файлов настроек сайта
func loadTags(settings_dir string) (*TagsList, error) {
	var Tags TagsList
	//проверяем существует ли в директории настроек файл со списком рубрик блога tags.xml
	tags_list_file := filepath.Join(settings_dir, "tags.xml")
	if _, err := os.Stat(tags_list_file); os.IsNotExist(err) {
		return nil, errors.New("Не найден файл списка рубрик блога")
	}
	//загружаем список рубрик блога
	raw, err := ioutil.ReadFile(tags_list_file)
	if err != nil {
		return nil, err
	}
	err = xml.Unmarshal(raw, &Tags)
	if err != nil {
		return nil, err
	}
	for _, tag := range Tags.Tags {
		tag.Posts = 0
	}
	return &Tags, nil
}

//----------------------------------------------------------
//обход поддиректорий и обработка xml-файлов постов блога
func handleBlogFiles(posts *SortedBlogPostList, tags *TagsList, total_posts *int) filepath.WalkFunc {
	return func(current_path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if src, _ := os.Stat(current_path); !src.IsDir() {
			//обработка файла
			_, filename := filepath.Split(current_path)
			ext := filepath.Ext(filename)
			if ext == ".xml" {
				raw, err := ioutil.ReadFile(current_path)
				if err != nil {
					return err
				}
				var post Post
				err = xml.Unmarshal(raw, &post)
				if err != nil {
					return err
				}

				*total_posts++
				//увеличиваем счётчик постов в рубрике
				tags.Tags[post.Tagid-1].Posts++
				//уникальный строковый идентификатор поста
				post.Fuseaction = strings.TrimSuffix(filename, filepath.Ext(filename))
				//название рубрики поста
				post.Tag = tags.Tags[post.Tagid-1].Name
				//поле сортировки
				post.SortDate, _ = time.Parse("02.01.2006", post.Date)
				post.Day = post.SortDate.Day()
				post.Month = RussianMonth[post.SortDate.Month()]
				post.Year = post.SortDate.Year()
				//добавляем в список постов блога
				*posts = append(*posts, post)
			}

		}
		return nil
	}
}

//функция загрузки списка постов блога
//posts_source_dir - исходная директория постов блога
//tags - список рубрик блога
//sitename - имя сайта
//limit - количество выбираемых постов, 0 если не ограничено
func loadBlog(posts_source_dir string, tags *TagsList) (*SortedBlogPostList, int, error) {
	var posts SortedBlogPostList
	total_posts := 0
	if _, err := os.Stat(posts_source_dir); err == nil {
		//обрабатываем все файлы с расширением xml из директории блога и её поддиректорий
		err = filepath.Walk(posts_source_dir, handleBlogFiles(&posts, tags, &total_posts))
		if err != nil {
			return nil, 0, err
		}
	}
	//сортируем список постов блога
	sort.Sort(sort.Reverse(posts))
	return &posts, total_posts, nil
}

//------------------------------------------------------------
//формирование файлов блога
//settings_dir - директория файлов настроек сайта
//destination_blogdir - целевая директория файлов публикаций
//posts_sourcedir - исходная директория постов блога
//templates_dir - директория шаблонов сайта
//domain - домен сайта
//sitemap - содержимое файла sitemap
func CreateBlog(settings_dir string, destination_blogdir string, posts_sourcedir string, templates_dir string, domain string, sitemap *string) error {
	//--------------------------------------------
	//количество постов блога на страницу
	posts_per_page := 10
	//--------------------------------------------
	//загружаем список рубрик блога
	tags, err := loadTags(settings_dir)
	if err != nil {
		return err
	}
	//если в целевой директории отсутствует папка блога blog - создаём её
	if _, err := os.Stat(destination_blogdir); os.IsNotExist(err) {
		//создаём директорию
		err = os.Mkdir(destination_blogdir, 0755)
		if err != nil {
			return err
		}
	}
	//если в целевой папке блога отсутствует папка постов posts - создаём её
	posts_dir := filepath.Join(destination_blogdir, "posts")
	if _, err := os.Stat(posts_dir); os.IsNotExist(err) {
		//создаём директорию
		err = os.Mkdir(posts_dir, 0755)
		if err != nil {
			return err
		}
	}
	//загружаем список постов блога
	posts, total_posts, err := loadBlog(posts_sourcedir, tags)
	if err != nil {
		return err
	}
	//формируем список рубрик блога, в которых есть посты
	active_tags := []Tag{}
	for _, value := range tags.Tags {
		//fmt.Println(value.Id)
		if value.Posts > 0 {
			active_tags = append(active_tags, value)
		}
	}
	//--------------------------------------------
	//формируем ленту блога без фильтрации
	//проверяем наличие в директории настроек сайта шаблона ленты блога blog.html
	blog_template_path := filepath.Join(settings_dir, "blog.html")
	if _, err := os.Stat(blog_template_path); os.IsNotExist(err) {
		return errors.New("Не найден файл шаблона ленты блога")
	}
	//разбиваем ленту блога на страницы
	current_blog_page := []Post{}
	current_posts_count := 1
	current_page := 1

	for _, value := range *posts {
		if current_posts_count < posts_per_page {
			//добавляем пост в список постов для текущей страницы
			current_blog_page = append(current_blog_page, value)
			current_posts_count++
		} else {
			//сохраняем очередную страницу
			//данные для передачи шаблону
			data := struct {
				Fuseaction     string
				Tags           []Tag
				Blog           SortedBlogPostList
				Pagenum        int
				Next_page      int
				Total          int
				Tagid          int
				Posts_per_page int
			}{
				"blog.html",
				active_tags,
				current_blog_page,
				current_page,
				1,
				total_posts,
				0,
				posts_per_page,
			}
			//формируем страницу
			content, err := ParseFileView(blog_template_path, templates_dir, data, "blog.html")
			if err != nil {
				return errors.New(ErrorMessages["parse_template_error"] + err.Error())
			}
			//сохраняем в целевую директорию
			if current_page == 1 {
				err = ioutil.WriteFile(filepath.Join(destination_blogdir, "index.html"), []byte(content), 0755)
			} else {
				err = ioutil.WriteFile(filepath.Join(destination_blogdir, strconv.Itoa(current_page)+".html"), []byte(content), 0755)
			}
			if err != nil {
				return errors.New(ErrorMessages["error_creating_file"] + err.Error())
			}
			//переустанавливаем счётчики
			current_page++
			current_posts_count = 1
			current_blog_page = current_blog_page[:0]
		}
	}

	//данные для передачи шаблону
	data := struct {
		Fuseaction     string
		Tags           []Tag
		Blog           SortedBlogPostList
		Pagenum        int
		Next_page      int
		Total          int
		Tagid          int
		Posts_per_page int
	}{
		"blog.html",
		active_tags,
		current_blog_page,
		current_page,
		0,
		total_posts,
		0,
		posts_per_page,
	}
	//формируем страницу
	content, err := ParseFileView(blog_template_path, templates_dir, data, "blog.html")
	if err != nil {
		return errors.New(ErrorMessages["parse_template_error"] + err.Error())
	}
	//сохраняем в целевую директорию
	if current_page == 1 {
		err = ioutil.WriteFile(filepath.Join(destination_blogdir, "index.html"), []byte(content), 0755)
	} else {
		err = ioutil.WriteFile(filepath.Join(destination_blogdir, strconv.Itoa(current_page)+".html"), []byte(content), 0755)
	}
	if err != nil {
		return errors.New(ErrorMessages["error_creating_file"] + err.Error())
	}

	//для всех активных рубрик блога формируем собственный файл ленты
	tag_posts := make(map[string][]Post)
	for _, value := range active_tags {
		tag_posts[value.Name] = []Post{}
	}

	for _, value := range *posts {
		tag_posts[value.Tag] = append(tag_posts[value.Tag], value)
	}

	for _, value := range active_tags {
		target_dir := filepath.Join(destination_blogdir, strconv.Itoa(value.Id))
		if _, err := os.Stat(target_dir); os.IsNotExist(err) {
			//создаём директорию
			err = os.Mkdir(target_dir, 0755)
			if err != nil {
				return err
			}
		}
		//разбиваем ленту блога на страницы
		current_blog_page := []Post{}
		current_posts_count := 1
		current_page := 1

		for _, post := range tag_posts[value.Name] {
			if current_posts_count < posts_per_page {
				//добавляем пост в список постов для текущей страницы
				current_blog_page = append(current_blog_page, post)
				current_posts_count++
			} else {
				//сохраняем очередную страницу
				//данные для передачи шаблону
				data := struct {
					Fuseaction     string
					Tags           []Tag
					Blog           SortedBlogPostList
					Pagenum        int
					Next_page      int
					Total          int
					Tagid          int
					Posts_per_page int
				}{
					"blog.html",
					active_tags,
					current_blog_page,
					current_page,
					1,
					total_posts,
					value.Id,
					posts_per_page,
				}
				//формируем страницу
				content, err := ParseFileView(blog_template_path, templates_dir, data, "blog.html")
				if err != nil {
					return errors.New(ErrorMessages["parse_template_error"] + err.Error())
				}
				//сохраняем в целевую директорию
				if current_page == 1 {
					err = ioutil.WriteFile(filepath.Join(target_dir, "index.html"), []byte(content), 0755)
				} else {
					err = ioutil.WriteFile(filepath.Join(target_dir, strconv.Itoa(current_page)+".html"), []byte(content), 0755)
				}
				if err != nil {
					return errors.New(ErrorMessages["error_creating_file"] + err.Error())
				}
				//переустанавливаем счётчики
				current_page++
				current_posts_count = 1
				current_blog_page = current_blog_page[:0]
			}
		}
		//данные для передачи шаблону
		data := struct {
			Fuseaction     string
			Tags           []Tag
			Blog           SortedBlogPostList
			Pagenum        int
			Next_page      int
			Total          int
			Tagid          int
			Posts_per_page int
		}{
			"blog.html",
			active_tags,
			current_blog_page,
			current_page,
			0,
			total_posts,
			value.Id,
			posts_per_page,
		}
		//формируем страницу
		content, err := ParseFileView(blog_template_path, templates_dir, data, "blog.html")
		if err != nil {
			return errors.New(ErrorMessages["parse_template_error"] + err.Error())
		}
		//сохраняем в целевую директорию
		if current_page == 1 {
			err = ioutil.WriteFile(filepath.Join(target_dir, "index.html"), []byte(content), 0755)
		} else {
			err = ioutil.WriteFile(filepath.Join(target_dir, strconv.Itoa(current_page)+".html"), []byte(content), 0755)
		}
		if err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}

	}
	//--------------------------------------------------------
	//формируем страницы постов блога
	//проверяем наличие в директории настроек сайта шаблона поста блога post.html
	post_template_path := filepath.Join(settings_dir, "post.html")
	if _, err := os.Stat(post_template_path); os.IsNotExist(err) {
		return errors.New("Не найден файл шаблона поста блога")
	}
	for _, value := range *posts {
		//данные для передачи шаблону
		data := struct {
			Fuseaction string
			Tags       []Tag
			Blogpost   Post
			Total      int
		}{
			"post.html",
			active_tags,
			value,
			total_posts,
		}
		//формируем страницу
		content, err := ParseFileView(post_template_path, templates_dir, data, "post.html")
		if err != nil {
			return errors.New(ErrorMessages["parse_template_error"] + err.Error())
		}
		//сохраняем в целевую директорию
		err = ioutil.WriteFile(filepath.Join(posts_dir, value.Fuseaction+".html"), []byte(content), 0755)
		if err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}
		//url страницы поста блога на целевом сервере
		url := domain + "/blog/posts/" + value.Fuseaction + ".html"
		//добавляем страницу в sitemap.xml
		*sitemap += "<url><loc>" + url + "</loc></url>"
	}

	return nil
}
