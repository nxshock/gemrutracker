package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/nxshock/gwp"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"golang.org/x/text/encoding/charmap"
)

type Parser struct {
	httpClient *http.Client
}

var parser *Parser

func initParser() error {
	logrus.Info("Инициализация парсера...")

	parser = &Parser{
		httpClient: &http.Client{
			Timeout: 5 * time.Second}}

	if config.HttpProxyAddress != "" {
		dialer, err := proxy.SOCKS5("tcp", config.HttpProxyAddress, nil, proxy.Direct)
		if err != nil {
			return err
		}

		parser.httpClient.Transport = &http.Transport{Dial: dialer.Dial}
	}

	return nil
}

func (parser *Parser) Update() error {
	var stepRange = config.UpdateRecordCount + 1

	logrus.Info("Начато обновление базы данных...")

	maxId, err := lastDbId()
	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"от": maxId - stepRange,
		"до": maxId + stepRange,
	}).Debug("Начат процесс обновления...")

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при старте транзакции: %v", err)
	}

	workerPool := gwp.New(config.ParserThreadCount)

	for i := maxId - stepRange; i < maxId+stepRange; i++ {
		if i <= 0 {
			continue
		}

		workerPool.Add(func() error {
			info, err := parser.GetPage(i)
			if err != nil {
				logrus.WithField("id", i).Debug(err)
				return err
			}

			_, err = tx.Exec("INSERT OR REPLACE INTO data(id, magnet, forum_id) SELECT ?, ?, (SELECT forums.id FROM forums WHERE name = ?)", i, info.MagnetData, info.ForumName)
			if err != nil {
				logrus.WithField("id", i).Errorln("Ошибка при выполнении вставки в базу данных:", err)
				return err
			}

			_, err = tx.Exec("INSERT OR REPLACE INTO titles(rowid, title) SELECT ?, ?", i, info.Title)
			if err != nil {
				logrus.WithField("id", i).Errorln("Ошибка при выполнении вставки в базу данных:", err)
				return err
			}

			return nil
		})
	}

	workerPool.CloseAndWait()

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка при коммите транзакции: %v", err)
	}

	newLastId, err := lastDbId()
	if err != nil {
		logrus.WithError(err).Warn("Обновление базы данных завершено.")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"добавлено": newLastId - maxId,
		"ошибок":    workerPool.ErrorCount(),
	}).Info("Обновление базы данных завершено.") // TODO: вместо "добавлено" показывает изменение ID

	return nil
}

func (parser *Parser) GetPage(id int) (*Info, error) {
	resp, err := parser.httpClient.Get(fmt.Sprintf("https://rutracker.org/forum/viewtopic.php?t=%d", id))
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	title := decode(doc.Find("a#topic-title").First().Text())
	if title == "" {
		return nil, errors.New("title not found")
	}

	forumName := decode(doc.Find("td.nav > a").First().Text())
	if title == "" {
		return nil, errors.New("forum not found")
	}

	magnet, exists := doc.Find("a.magnet-link").First().Attr("href")
	if !exists {
		return nil, errors.New("magnet not found")
	}

	mu, _ := url.Parse(magnet)
	xt, exists := mu.Query()["xt"]
	if !exists || len(xt) == 0 {
		return nil, errors.New("xt not found")
	}

	info := &Info{
		Title:      title,
		ForumName:  forumName,
		MagnetData: strings.TrimPrefix(xt[0], "urn:btih:")}

	return info, nil
}

func decode(s string) string {
	decoded, err := charmap.Windows1251.NewDecoder().String(s)
	if err != nil {
		return s
	}
	return decoded
}
