package main

import (
	"embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nxshock/gemini"
	"github.com/sirupsen/logrus"
)

//go:embed site/index.gmi
var content embed.FS

func init() {
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, DisableTimestamp: true})

	if len(os.Args) > 2 {
		printUsage()
		os.Exit(1)
	}

	var configPath string
	if len(os.Args) == 2 {
		configPath = os.Args[1]
	} else {
		configPath = defaultConfigFilePath
	}

	err := initConfig(configPath)
	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.SetLevel(config.LogLevel)

	err = initDb(config.WorkDir)
	if err != nil {
		logrus.Fatalln(err)
	}

	err = initParser()
	if err != nil {
		logrus.Fatalln(err)
	}

	if config.UpdatePeriodHours > 0 {
		logrus.Infof("Запланировано обновление базы данных каждые %d часов.", config.UpdatePeriodHours)
		go func() {
			for {
				err := parser.Update()
				if err != nil {
					logrus.Errorln("Ошибка при обновлении базы данных:", err)
				}
				time.Sleep(time.Hour * time.Duration(config.UpdatePeriodHours))
			}
		}()
	}

	gemini.HandleFunc("/search", searchHandler)
	gemini.HandleFunc("/", handle)
}

func main() {
	logrus.WithField("адрес", config.ListenAddress).Infof("Начало прослушивания адреса...")
	go func() {
		err := gemini.ListenAndServeTLS(config.ListenAddress, config.CertFilePath, config.KeyFilePath, nil)
		if err != nil {
			logrus.Fatalln(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	logrus.Infoln("Закрытие базы данных...")
	err := db.Close()
	if err != nil {
		logrus.Fatalf("ошибка при сохранении базы данных: %v", err)
	}
	err = index.Save()
	if err != nil {
		logrus.Fatalf("ошибка при сохранении индекса: %v", err)
	}

	logrus.Infoln("Программа остановлена.")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n%s <path to config file>\n", filepath.Base(os.Args[0]))
}
