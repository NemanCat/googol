// Googol генератор статических html-страниц из шаблонов.
// Общие данные и функции приложения.

package main

import (
	"bytes"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"
)

// Подсказка по запуску приложения.
var HelpMessage = "Пример использования: googol -source=путь_к_исходной_директории -destination=путь_к_целевой_директории -domain=имя_домена_сайта"

// Сообщения об ошибках.
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

// Русские названия месяцев.
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

// CopyFile копирует файл source в файл dest и сохраняет права исходного файла.
func CopyFile(source string, dest string) error {
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

	if _, err = io.Copy(destfile, sourcefile); err != nil {
		return err
	}

	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	return os.Chmod(dest, sourceinfo.Mode())
}

// HashStringCrc32 вычисляет crc32-хэш строки.
func HashStringCrc32(source string) string {
	table := crc32.MakeTable(IEEE)
	return strconv.FormatUint(uint64(crc32.Checksum([]byte(source), table)), 10)
}

// HashFileCrc32WithError вычисляет crc32-хэш файла и возвращает ошибку чтения файла.
func HashFileCrc32WithError(filepath string) (string, error) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	return HashStringCrc32(string(content)), nil
}

// HashFileCrc32 вычисляет crc32-хэш файла.
// Функция сохранена для обратной совместимости со старым кодом.
// Новый код лучше писать через HashFileCrc32WithError, чтобы не терять ошибку чтения файла.
func HashFileCrc32(filepath string) string {
	hash, err := HashFileCrc32WithError(filepath)
	if err != nil {
		return ""
	}

	return hash
}

// IsStringInList проверяет наличие строки в массиве строк.
func IsStringInList(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}

	return false
}

// ParseFileView парсит файл шаблона.
// pagepath — полный путь к файлу.
// templatesDir — директория шаблонов с расширением *.tmpl, может быть пустой строкой.
// data — данные, передаваемые шаблону.
// fuseaction — имя корневого шаблона.
func ParseFileView(pagepath string, templatesDir string, data interface{}, fuseaction string) (string, error) {
	var doc bytes.Buffer

	// Функции для шаблонов.
	funcMap := template.FuncMap{
		// Прибавление единицы.
		"Inc": func(i int) int {
			return i + 1
		},
		// Вычитание единицы.
		"Dec": func(i int) int {
			return i - 1
		},
		// Первая буква в строке.
		"First": func(s string) string {
			runes := []rune(s)
			if len(runes) == 0 {
				return ""
			}

			return string(runes[0])
		},
	}

	// Создаём шаблон.
	t := template.New(fuseaction).Funcs(funcMap)

	// Загружаем дополнительные шаблоны, если они есть.
	if len(templatesDir) > 0 {
		_, err := t.ParseGlob(filepath.Join(templatesDir, "*.tmpl"))
		if err != nil {
			return "", err
		}
	}

	// Загружаем файл для парсинга.
	tmpl, err := ioutil.ReadFile(pagepath)
	if err != nil {
		return "", err
	}

	t, err = t.Parse(string(tmpl))
	if err != nil {
		return "", err
	}

	// Парсим файл шаблона.
	if err = t.Execute(&doc, data); err != nil {
		return "", err
	}

	return doc.String(), nil
}
