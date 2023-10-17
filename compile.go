// Googol генератор статических html - страниц из шаблонов
// обход поддиректорий исходной директории и обработка файлов
package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//запись контента в файл на целевом сервере и сохранение md5 - хэша контента на исходном сервере
//destination_file - файл на целевом сервере
//content - контент для записи в целевой файл
//fuseaction - уникальный строковый идентификатор файла
//file - исходный файл
//source_root - корень исходной директории
func writeContentToDest(destination_file string, content string, fuseaction string, file string, source_root string) error {
	//определяем файловые реквизиты исходного файла
	sourceinfo, err := os.Stat(file)
	if err != nil {
		return errors.New(ErrorMessages["parse_template_error"] + err.Error())
	}
	//создаём файл на целевом сервере
	err = ioutil.WriteFile(destination_file, []byte(content), sourceinfo.Mode())
	if err != nil {
		return errors.New(ErrorMessages["error_creating_file"] + err.Error())
	}
	//сохраняем md5-хэш контента
	err = ioutil.WriteFile(filepath.Join(source_root, "__hash", fuseaction+".crc"), []byte(HashStringCrc32(content)), sourceinfo.Mode())
	if err != nil {
		return errors.New(ErrorMessages["error_creating_file"] + err.Error())
	}
	return nil
}

//обработка файла html/php
//file - полный путь к исходному файлу
//destination_root - корень целевой директории
//source_root - корень исходной директории
//blog - список постов блога (nil если блога нет или он пустой)
//tags - список рубрик блога
//articles - список публикаций
//sitemap - содержимое файла sitemap
//domain - целевой домен
func handleParseFile(file string, destination_root string, source_root string, sitemap *string, domain string) error {
	//если на исходном сервере нет директории __hash, создаём её
	if _, err := os.Stat(filepath.Join(source_root, "__hash")); os.IsNotExist(err) {
		err = os.Mkdir(filepath.Join(source_root, "__hash"), 0755)
		if err != nil {
			return errors.New(ErrorMessages["error_creating_dir"] + err.Error())
		}
	}
	//уникальный строковый идентификатор файла
	fuseaction := strings.Replace(strings.TrimLeft(filepath.ToSlash(strings.Replace(file, source_root, "", -1)), "/"), "/", "-", -1)
	//url старницы на целевом сервере
	url := domain + filepath.ToSlash(strings.Replace(file, source_root, "", -1))
	//поддиректория верхнего уровня
	top_subdir := strings.Split(fuseaction, "-")[0]
	//файлы в поддиректории assets не обрабатываются
	if top_subdir == "assets" {
		return nil
	}

	//директория шаблонов страниц
	template_dir := filepath.Join(source_root, "__templates")
	//данные для передачи шаблону
	data := struct {
		Fuseaction string
	}{
		fuseaction,
	}
	//парсим файл
	content, err := ParseFileView(file, template_dir, data, fuseaction)
	if err != nil {
		return errors.New(ErrorMessages["parse_template_error"] + err.Error())
	}

	//проверяем, существует ли файл с таким именем на целевом сервере
	destination_file := strings.Replace(file, source_root, destination_root, -1)
	hash_file := filepath.Join(source_root, "__hash", fuseaction+".crc")
	if _, err := os.Stat(destination_file); os.IsNotExist(err) {
		//копируем контент в файл на целевом сервере
		err = writeContentToDest(destination_file, content, fuseaction, file, source_root)
		if err != nil {
			return errors.New(ErrorMessages["error_creating_file"] + err.Error())
		}
	} else {
		//существует ли хэш - файл для исходного файла
		if _, err := os.Stat(hash_file); os.IsNotExist(err) {
			//копируем контент в файл на целевом сервере
			err = writeContentToDest(destination_file, content, fuseaction, file, source_root)
			if err != nil {
				return errors.New(ErrorMessages["error_creating_file"] + err.Error())
			}
		} else {
			//загружаем старый хэш из хэш - файла
			b, _ := ioutil.ReadFile(hash_file)
			old_hash := string(b)
			//подсчитываем новый хэш
			new_hash := HashStringCrc32(content)
			if old_hash != new_hash {
				//копируем контент в файл на целевом сервере
				err = writeContentToDest(destination_file, content, fuseaction, file, source_root)
				if err != nil {
					return errors.New(ErrorMessages["error_creating_file"] + err.Error())
				}
			}
		}
	}
	if fuseaction != "404.html" {
		*sitemap += "<url><loc>" + url + "</loc></url>"
	}
	return nil
}

//обработка файла не html/php
//file - полный путь к исходному файлу
//destination_root - корень целевой директории
//source_root - корень исходной директории
func handleCopyFile(file string, destination_root string, source_root string) error {
	//существует ли файл с таким именем на целевом сервере
	destination_file := strings.Replace(file, source_root, destination_root, -1)
	if _, err := os.Stat(destination_file); os.IsNotExist(err) {
		//пытаемся копировать файл на целевой сервер
		err = CopyFile(file, destination_file)
		if err != nil {
			return errors.New(ErrorMessages["copy_error"] + destination_file)
		}
	} else {
		//вычисляем crc - суммы исходного и целевого файла
		source_crc32 := HashFileCrc32(file)
		destination_crc32 := HashFileCrc32(destination_file)
		if source_crc32 != destination_crc32 {
			//копируем новую версию файла
			err = CopyFile(file, destination_file)
			if err != nil {
				return errors.New(ErrorMessages["copy_error"] + destination_file)
			}
		}
	}
	return nil
}

//обработка файлов в поддиректориях исходной директории
//destination_root - корень целевой директории
//source_root - корень исходной директории
//sitemap - содержимое файла sitemap
//domain - целевой домен
func handleSourceFile(destination_root string, source_root string, sitemap *string, domain string) filepath.WalkFunc {
	return func(current_path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if src, _ := os.Stat(current_path); src.IsDir() {
			//обработка директории
			_, folder := filepath.Split(current_path)
			//имя поддиректории не должно начинаться с символов __
			if strings.HasPrefix(folder, "__") {
				return filepath.SkipDir
			}
		} else {
			//обработка файла
			_, filename := filepath.Split(current_path)
			//имя файла не должно начинаться с символа _
			if strings.HasPrefix(filename, "_") {
				return nil
			}
			//по расширению файла определяем его обработчик
			ext := filepath.Ext(filename)
			if ext == ".html" || ext == ".php" {
				err = handleParseFile(current_path, destination_root, source_root /*blog, tags, articles,*/, sitemap, domain)
			} else {
				err = handleCopyFile(current_path, destination_root, source_root)
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
}

//обход поддиректорий исходной директории
//source - исходная директория
//destination - целевая директория
//sitemap - содержимое файла sitemap
//domain - целевой домен
func HandleSourceDir(source string, destination string, sitemap *string, domain string) error {
	err := filepath.Walk(source, handleSourceFile(destination, source, sitemap, domain))
	return err
}
