// Googol генератор статических html - страниц из шаблонов
// модуль формирования страницы Вопросы и ответы

package main

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

//структура записи Вопрос-ответ
type QA struct {
	//------------------------
	//загружаемые из файла поля
	//дата записи
	Date string `xml:"date"`
	//имя спрашивающего
	Name string `xml:"name"`
	//вопрос
	Question string `xml:"question"`
	//ответ
	Answer string `xml:"answer"`
	//-----------------------------
	//вычисляемые поля
	//дата для сортировки списка
	SortDate  time.Time
	Day, Year int
	Month     string
}

//сортировка записей по дате
type SortedQAList []QA

func (p SortedQAList) Len() int {
	return len(p)
}

func (p SortedQAList) Less(i, j int) bool {
	return p[i].SortDate.Before(p[j].SortDate)
}

func (p SortedQAList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

//------------------------------------------------------------
//обход поддиректорий и обработка xml-файлов записей
func handleQAFiles(qas *SortedQAList, total_qas *int) filepath.WalkFunc {
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
				var qa QA
				err = xml.Unmarshal(raw, &qa)
				if err != nil {
					return err
				}

				*total_qas++
				//поле сортировки
				qa.SortDate, _ = time.Parse("02.01.2006", qa.Date)
				qa.Day = qa.SortDate.Day()
				qa.Month = RussianMonth[qa.SortDate.Month()]
				qa.Year = qa.SortDate.Year()
				//добавляем в список записей
				*qas = append(*qas, qa)
			}
		}
		return nil
	}
}

//функция загрузки списка вопросов и ответов
//qa_source_dir - исходная директория записей вопрос-ответ
func loadQA(qa_source_dir string) (*SortedQAList, int, error) {
	var qas SortedQAList
	total_qa := 0
	if _, err := os.Stat(qa_source_dir); err == nil {
		//обрабатываем все файлы с расширением xml из директории Вопросы и ответы и её поддиректорий
		err = filepath.Walk(qa_source_dir, handleQAFiles(&qas, &total_qa))
		if err != nil {
			return nil, 0, err
		}
	}
	//сортируем список постов блога
	sort.Sort(sort.Reverse(qas))
	return &qas, total_qa, nil
}

//--------------------------------------------------------------------
//формирование страницы Вопрос-ответ
//destination_dir - целевая директория
//settings_dir - директория файлов настроек сайта
//qa_dir - исходная директория записей
//templates_dir - директория шаблонов сайта
//domain - домен сайта
//sitemap - содержимое файла sitemap
func CreateQA(destination_dir string, settings_dir string, qa_dir string, templates_dir string, domain string, sitemap *string) error {
	//загружаем список вопросов и ответов
	qas, total_qas, err := loadQA(qa_dir)
	if err != nil {
		return err
	}
	//проверяем наличие в директории настроек сайта шаблона страницы Вопросы и ответы qa.html
	qa_template_path := filepath.Join(settings_dir, "qa.html")
	if _, err := os.Stat(qa_template_path); os.IsNotExist(err) {
		return errors.New("Не найден файл шаблона страницы Вопросы и ответы")
	}
	//парсим шаблон страницы
	//данные для передачи шаблону
	data := struct {
		Fuseaction string
		QA         SortedQAList
		total_qa   int
	}{
		"qa.html",
		*qas,
		total_qas,
	}
	content, err := ParseFileView(qa_template_path, templates_dir, data, "qa.html")
	if err != nil {
		return errors.New(ErrorMessages["parse_template_error"] + err.Error())
	}
	//сохраняем сформированную страницу
	err = ioutil.WriteFile(filepath.Join(destination_dir, "qa.html"), []byte(content), 0755)
	if err != nil {
		return errors.New(ErrorMessages["error_creating_file"] + err.Error())
	}
	return nil
}
