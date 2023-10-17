// Googol генератор статических html - страниц из шаблонов
// командная строка запуска приложения имеет вид:
// googol --source=<исходная корневая директория> --destination=<целевая корневая директория> --domain=<имя целевого домена>
// все параметры являются обязательными
// приложение выполняет следующие операции:
// 1. синхронизирует структуру поддиректорий в исходной и целевой директориях
//  1.1. удаление в целевой директории поддиректорий, отсутствующих в исходной директории
//  1.2. создание в целевой директории поддиректорий, существующих в исходной директории и отсутствующих в целевой директории
//  при этом поддиректории в исходной директории, имена которых начинаются с символа _, в целевую директорию не копируются
// 2. обходит все поддиректории исходной директории, чьи имена не начинаются с символа _, и обрабатывает все файлы, имена которых не начинаются с символа _
// 3. для всех файлов с расширением HTML и PHP выполняется парсинг, директорией шаблонов считается поддиректория __templates
//  если в целевой директории нет файла с таким именем, распарсенный файл записывается в целевую директорию
//  если в целевой директории есть файл с таким именем,	для целевого и вновь распарсенного файла вычисляется crc - сумма и целевой файл заменяется если
//  вновь распарсенный файл отличается от него
// 4. для всех файлов с любым другим расширением проверяется наличие файла с таким же именем в целевой директории
//	если в целевой директории нет такого файла, туда копируется файл из исходной директории
//  если в целевой директории есть такой файл, сравниваются crc - суммы целевого и исходного файла
//  исходный файл копируется на место целевого в случае отличия crc - сумм
// 5. если в исходной директории есть поддиректория с именем _blog - запускается модуль создания файлов блога
// 6. если в исходной директории есть поддиректория с именем _articles - запускается модуль создания файлов публикаций

package main

import (
	"flag"
	"fmt"

	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	//---------------------------------------
	//считываем параметры командной строки
	flagSet := flag.NewFlagSet("flag_set", flag.ExitOnError)
	//исходная корневая директория
	source := flagSet.String("source", "", "Укажите исходную директорию")
	//целевая корневая директория
	destination := flagSet.String("destination", "", "Укажите целевую директорию")
	//название целевого домена
	domain := flagSet.String("domain", "", "Укажите домен сайта")
	//проверяем параметры командной строки
	//парсим набор флагов для команды
	if err := flagSet.Parse(os.Args[1:]); err == nil {
		//проверяем, указан ли путь к исходной директории
		if len(*source) == 0 {
			fmt.Println(ErrorMessages["required_parameter"], "source")
			fmt.Println(HelpMessage)
			return
		}
		//проверяем, указан ли путь к целевой директории
		if len(*destination) == 0 {
			fmt.Println(ErrorMessages["required_parameter"], "destination")
			fmt.Println(HelpMessage)
			return
		}
		//проверяем, указан ли домен сайта
		if len(*domain) == 0 {
			fmt.Println(ErrorMessages["required_parameter"], "domain")
			fmt.Println(HelpMessage)
			return
		}
	}
	//---------------------------------------
	//синхронизация структуры поддиректорий в целевой и исходной директориях
	fmt.Print("Синхронизирую исходную и целевую директории...")
	err := syncDirs(*source, *destination)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("сделано")
	//----------------------------------------
	//содержимое файла файл sitemap
	sitemap := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">"
	//директория настроек сайта
	settings_dir := filepath.Join(*source, "__settings")
	//директория шаблонов сайта
	templates_dir := filepath.Join(*source, "__templates")
	//----------------------------------------
	//загружаем список публикаций, отсортированный по заголовку
	articles_dir := filepath.Join(*source, "__articles")
	if _, err := os.Stat(articles_dir); !os.IsNotExist(err) {
		fmt.Print("Формирование файлов публикаций...")
		destination_articlesdir := filepath.Join(*destination, "articles")
		err = CreateArticles(settings_dir, articles_dir, destination_articlesdir, templates_dir, *domain, &sitemap)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println("сделано")
	}
	//----------------------------------------
	//запуск модуля блога
	blog_dir := filepath.Join(*source, "__blog")
	if _, err := os.Stat(blog_dir); !os.IsNotExist(err) {
		fmt.Print("Формирование файлов блога...")
		destination_blogdir := filepath.Join(*destination, "blog")
		posts_sourcedir := filepath.Join(*source, "__blog")
		err = CreateBlog(settings_dir, destination_blogdir, posts_sourcedir, templates_dir, *domain, &sitemap)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println("сделано")
	}
	//--------------------------------------
	//запуск модуля вопросов и ответов
	qa_dir := filepath.Join(*source, "__qa")
	if _, err := os.Stat(qa_dir); !os.IsNotExist(err) {
		fmt.Print("Формирование страницы Вопросы и ответы...")
		err = CreateQA(*destination, settings_dir, qa_dir, templates_dir, *domain, &sitemap)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println("сделано")
	}
	//--------------------------------------
	//обход поддиректорий исходной директории и обработка файлов в них
	fmt.Print("Компилирую файлы и копирую в целевую директорию...")
	err = HandleSourceDir(*source, *destination, &sitemap, *domain)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//записываем файл sitemap.xml в целевую директорию
	sitemap += "</urlset>"
	err = ioutil.WriteFile(filepath.Join(*destination, "sitemap.xml"), []byte(sitemap), os.FileMode(int(0777)))
	if err != nil {
		fmt.Println(ErrorMessages["error_creating_file"] + err.Error())
		return
	}
	fmt.Println("сделано")
	//---------------------------------------
	fmt.Println("Сайт успешно скомпилирован и скопирован в целевую директорию")
}
