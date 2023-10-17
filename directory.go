// Googol генератор статических html - страниц из шаблонов
// синхронизация структуры поддиректорий исходной и целевой директории

package main

import (
	"errors"
	//"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//-----------------------------------------------------
//вычисление лишних поддиректорий в целевой директории
//destination_root - целевая директория
//source_root - исходная директория
//dirs_to_delete - список лишних поддиректорий
func excessDestDirs(destination_root string, source_root string, dirs_to_delete *[]string) filepath.WalkFunc {
	return func(current_path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if current_path != destination_root {
			if src, _ := os.Stat(current_path); src.IsDir() {
				//поддиректория с таким же именем в исходной директории
				source_dir := strings.Replace(current_path, destination_root, source_root, -1)
				//проверяем, существует ли такая поддиректория в исходной директории
				if _, err := os.Stat(source_dir); os.IsNotExist(err) {
					//заносим директорию в список намеченных к удалению
					*dirs_to_delete = append(*dirs_to_delete, current_path)
				}
			}
		}
		return nil
	}
}

//добавление несуществующих поддиректорий в целевую директорию
//destination_root - целевая директория
//source_root - исходная директория
func addDestDirs(destination_root string, source_root string) filepath.WalkFunc {
	return func(current_path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if src, _ := os.Stat(current_path); src.IsDir() {
			if current_path == source_root {
				return nil
			}
			if strings.HasPrefix(strings.Replace(path.Base(current_path), source_root+string(os.PathSeparator), "", -1), "__") {
				return filepath.SkipDir
			}
			//поддиректория с таким же именем в целевой директории
			destination_dir := strings.Replace(current_path, source_root, destination_root, -1)

			//проверяем, существует ли такая поддиректория в целевой директории
			if _, err := os.Stat(destination_dir); os.IsNotExist(err) {
				//создаём поддиректорию
				err = os.Mkdir(destination_dir, 0755)
				if err != nil {
					return errors.New(ErrorMessages["error_creating_dir"] + err.Error())
				}
			}
		}
		return nil
	}
}

//синхронизация структуры поддиректорий в исходной и целевой директориях
//source - исходная директория
//destination - целевая директория
//возвращает ошибку или nil в случае успешного завершения синхронизации
func syncDirs(source string, destination string) error {
	//проверяем существование исходной директории
	src, err := os.Stat(source)
	if os.IsNotExist(err) {
		return errors.New(ErrorMessages["directory_not_exists"] + source)
	}
	//проверяем является ли указанный исходный путь директорией
	if src.IsDir() == false {
		return errors.New(ErrorMessages["path_not_directory"] + source)
	}
	//проверяем существует ли целевая директория
	dest, err := os.Stat(destination)
	if os.IsNotExist(err) {
		return errors.New(ErrorMessages["directory_not_exists"] + destination)
	}
	//проверяем является ли указанный целевой путь директорией
	if dest.IsDir() == false {
		return errors.New(ErrorMessages["path_not_directory"] + destination)
	}
	//обходим поддиректории целевой директории и вычисляем лишние директории
	dirs_to_delete := []string{}
	err = filepath.Walk(destination, excessDestDirs(destination, source, &dirs_to_delete))
	if err != nil {
		return err
	}
	//пытаемся удалить лишние поддиректории в целевой директории
	for _, dir := range dirs_to_delete {
		if _, err := os.Stat(dir); err == nil {
			err = os.RemoveAll(dir)
			if err != nil {
				errors.New(ErrorMessages["directory_content_remove"] + dir)
			}
		}
	}
	//обходим поддиректории исходной директории и создаём в целевой директории отсутствующие поддиректории
	err = filepath.Walk(source, addDestDirs(destination, source))
	return nil
}
