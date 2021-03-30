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

	beforeUpdateMaxId, err := lastDbId()
	if err != nil {
		return fmt.Errorf("lastDbId(): %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"от": beforeUpdateMaxId - stepRange,
		"до": beforeUpdateMaxId + stepRange,
	}).Debug("Начат процесс обновления...")

	workerPool := gwp.New(config.ParserThreadCount)

	newMaxId := beforeUpdateMaxId

	for i := beforeUpdateMaxId - stepRange; i <= beforeUpdateMaxId+stepRange; i++ {
		if i <= 0 {
			continue
		}

		workerPool.Add(func() error {
			n := i

			info, err := parser.GetPage(n)
			if err != nil {
				logrus.WithField("id", n).Debug(err)
				return err
			}

			if n > newMaxId {
				newMaxId = n
			}

			err = db.Set(n, *info)
			if err != nil {
				logrus.WithField("id", i).Errorln("Ошибка при выполнении вставки в базу данных:", err)
				return err
			}

			index.Add(n, info.Title)

			return nil
		})
	}

	workerPool.CloseAndWait()

	err = index.Save()
	if err != nil {
		return fmt.Errorf("ошибка при сохранении индекса: %v", err)
	}

	err = db.Set("maxId", newMaxId)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении maxId: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"добавлено": newMaxId - beforeUpdateMaxId,
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
