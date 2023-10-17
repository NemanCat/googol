// Googol генератор статических html - страниц из шаблонов
// общие данные и функции приложения

package main

import (
	"bytes"
	//"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"
)

//подсказка
var HelpMessage = "Пример использования: googol -source=путь_к_исходной_директории -destination=путь_к_целевой_директории -domain=имя_домена_сайта"

//сообщения об ошибках
var ErrorMessages = map[string]string{
	"required_parameter":       "Не указано значение обязательного параметра: ",
	"directory_not_exists":     "Указанная директория не существует: ",
	"error_creating_dir":       "Ошибка при создании директории: ",
	"error_creating_file":      "Ошибка при создании файла: ",
	"directory_content_remove": "Ошибка при очистке директории: ",
	"path_not_directory":       "Указанный путь не является директорией: ",
	"copy_error":               "Ошибка при копировании файла или директории: ",
	"parse_template_error":     "Ошибка парсинга шаблона страницы: ",
}

const IEEE = 0xedb88320

//русские названия месяцев
var RussianMonth = map[time.Month]string{
	time.January:   "Января",
	time.February:  "Февраля",
	time.March:     "Марта",
	time.April:     "Апреля",
	time.May:       "Мая",
	time.June:      "Июня",
	time.July:      "Июля",
	time.August:    "Августа",
	time.September: "Сентября",
	time.October:   "Октября",
	time.November:  "Ноября",
	time.December:  "Декабря",
}

//--------------------------------------------------------------
//копирование файла
func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourcefile.Close()
	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destfile.Close()
	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}
	}
	return
}

//вычисление crc32 - хэша строки
func HashStringCrc32(source string) string {
	table := crc32.MakeTable(IEEE)
	return strconv.FormatUint(uint64(crc32.Checksum([]byte(source), table)), 10)
}

//вычисление crc32 - хэша файла
func HashFileCrc32(filepath string) string {
	content, _ := ioutil.ReadFile(filepath)
	return HashStringCrc32(string(content))
}

//поиск строки в массиве строк
func IsStringInList(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

//-------------------------------------------------------------------------------------
//парсинг файла
//pagepath - полный путь к файлу
//templates_dir - директория  шаблонов с расширением *.tmpl (может быть пустая строка)
//data - данные, передаваемые шаблону
func ParseFileView(pagepath string, templates_dir string, data interface{}, fuseaction string) (string, error) {
	var doc bytes.Buffer
	//функции для шаблонов
	funcMap := template.FuncMap{
		//прибавление единицы
		"Inc": func(i int) int {
			return i + 1
		},
		//вычитание единицы
		"Dec": func(i int) int {
			return i - 1
		},
		//первая буква в строке
		"First": func(s string) string {
			return string([]rune(s)[0])
		},
	}

	//создаём шаблон
	t := template.New(fuseaction).Funcs(funcMap)
	//загружаем дополнительные шаблоны если есть
	if len(templates_dir) > 0 {
		_, err := t.ParseGlob(filepath.Join(templates_dir, "*.tmpl"))
		if err != nil {
			return "", err
		}
	}
	//загружаем файл для парсинга
	tmpl, _ := ioutil.ReadFile(pagepath)
	t, err := t.Parse(string(tmpl))
	if err != nil {
		return "", err
	}
	//парсим файл шаблона
	err = t.Execute(&doc, data)
	if err != nil {
		return "", err
	}

	return doc.String(), nil
}
