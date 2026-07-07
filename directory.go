// Googol генератор статических html-страниц из шаблонов.
// Синхронизация структуры поддиректорий исходной и целевой директории.

package main

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// excessDestDirs вычисляет лишние поддиректории в целевой директории.
// destinationRoot — целевая директория.
// sourceRoot — исходная директория.
// dirsToDelete — список лишних поддиректорий.
func excessDestDirs(destinationRoot string, sourceRoot string, dirsToDelete *[]string) filepath.WalkFunc {
	return func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if currentPath == destinationRoot {
			return nil
		}
		if info == nil || !info.IsDir() {
			return nil
		}

		// Поддиректория с таким же именем в исходной директории.
		sourceDir := strings.Replace(currentPath, destinationRoot, sourceRoot, 1)

		// Проверяем, существует ли такая поддиректория в исходной директории.
		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			// Заносим директорию в список намеченных к удалению.
			*dirsToDelete = append(*dirsToDelete, currentPath)
		}

		return nil
	}
}

// addDestDirs добавляет несуществующие поддиректории в целевую директорию.
// destinationRoot — целевая директория.
// sourceRoot — исходная директория.
func addDestDirs(destinationRoot string, sourceRoot string) filepath.WalkFunc {
	return func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || !info.IsDir() {
			return nil
		}
		if currentPath == sourceRoot {
			return nil
		}
		if strings.HasPrefix(path.Base(currentPath), "__") {
			return filepath.SkipDir
		}

		// Поддиректория с таким же именем в целевой директории.
		destinationDir := strings.Replace(currentPath, sourceRoot, destinationRoot, 1)

		// Проверяем, существует ли такая поддиректория в целевой директории.
		if _, err := os.Stat(destinationDir); os.IsNotExist(err) {
			// Создаём поддиректорию.
			if err = os.Mkdir(destinationDir, 0755); err != nil {
				return errors.New(ErrorMessages["error_creating_dir"] + err.Error())
			}
		}

		return nil
	}
}

// syncDirs синхронизирует структуру поддиректорий в исходной и целевой директориях.
// source — исходная директория.
// destination — целевая директория.
func syncDirs(source string, destination string) error {
	// Проверяем существование исходной директории.
	src, err := os.Stat(source)
	if os.IsNotExist(err) {
		return errors.New(ErrorMessages["directory_not_exists"] + source)
	}
	if err != nil {
		return err
	}

	// Проверяем, является ли указанный исходный путь директорией.
	if !src.IsDir() {
		return errors.New(ErrorMessages["path_not_directory"] + source)
	}

	// Проверяем существование целевой директории.
	dest, err := os.Stat(destination)
	if os.IsNotExist(err) {
		return errors.New(ErrorMessages["directory_not_exists"] + destination)
	}
	if err != nil {
		return err
	}

	// Проверяем, является ли указанный целевой путь директорией.
	if !dest.IsDir() {
		return errors.New(ErrorMessages["path_not_directory"] + destination)
	}

	// Обходим поддиректории целевой директории и вычисляем лишние директории.
	dirsToDelete := []string{}
	if err = filepath.Walk(destination, excessDestDirs(destination, source, &dirsToDelete)); err != nil {
		return err
	}

	// Пытаемся удалить лишние поддиректории в целевой директории.
	for _, dir := range dirsToDelete {
		if _, err = os.Stat(dir); err == nil {
			if err = os.RemoveAll(dir); err != nil {
				return errors.New(ErrorMessages["directory_content_remove"] + dir + ": " + err.Error())
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	// Обходим поддиректории исходной директории и создаём в целевой директории отсутствующие поддиректории.
	return filepath.Walk(source, addDestDirs(destination, source))
}
