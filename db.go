package main

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

//go:embed init.sql
var sqlScripts embed.FS

type Info struct {
	Title      string
	MagnetData string
	ForumName  string
}

var db *sql.DB

func initDb(filePath string) error {
	absDbFilePath, err := filepath.Abs(filePath)
	if err == nil {
		filePath = absDbFilePath
	}

	fileExists, err := isFileExists(filePath)
	if err != nil {
		return err
	}

	db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=rwc", filePath))
	if err != nil {
		return err
	}

	if fileExists {
		logrus.WithField("filePath", filePath).Info("Чтение базы данных...")

		var rowsCount int
		err = db.QueryRow("SELECT count(*) FROM data").Scan(&rowsCount)
		if err != nil {
			return err
		}

		logrus.WithField("rowCount", strconv.Itoa(rowsCount)).Debugln("Подсчитано кол-во строк в базе данных.")
		return nil
	}

	// database not exists
	logrus.WithField("filePath", filePath).Info("Создание базы данных...")

	scriptBytes, err := sqlScripts.ReadFile("init.sql")
	if err != nil {
		db.Close()
		os.Remove(filePath)
		return fmt.Errorf("ошибка при получении скрипта: %v", err)
	}

	err = bulkExec(db, string(scriptBytes))
	if err != nil {
		db.Close()
		os.Remove(filePath)
		return fmt.Errorf("ошибка при выполнении скрипта: %v", err)
	}

	logrus.WithField("filePath", filePath).Info("База данных создана.")

	return nil
}

func bulkExec(db *sql.DB, script string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при старте транзакции: %v", err)
	}

	for i, v := range strings.Split(script, ";") {
		if strings.TrimSpace(v) == "" {
			continue
		}

		_, err := tx.Exec(v)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ошибка при выполнении команды №%d: %v", i+1, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %v", err)
	}

	return nil
}

func search(query string) ([]Info, error) {
	sqlText := "select titles.title, data.magnet, forums.name from titles(?) left join data on data.id = titles.rowid left join forums on forums.id = data.forum_id order by forums.id, rank"
	rows, err := db.Query(sqlText, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]Info, 0)
	for rows.Next() {
		if rows.Err() != nil {
			return nil, err
		}

		var (
			title     string
			magnet    string
			forumName string
		)

		err = rows.Scan(&title, &magnet, &forumName)
		if err != nil {
			return nil, err
		}

		results = append(results, Info{
			Title:      title,
			MagnetData: magnet,
			ForumName:  forumName})
	}

	return results, nil
}

func lastDbId() (int, error) {
	var maxId int
	err := db.QueryRow("SELECT ifnull(max(id),0) FROM data").Scan(&maxId)
	if err != nil {
		return 0, err
	}

	return maxId, nil
}

func recordCount() (int, error) {
	var recordCount int
	err := db.QueryRow("SELECT count(*) FROM data").Scan(&recordCount)
	if err != nil {
		return 0, err
	}

	return recordCount, nil
}

func isFileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}
