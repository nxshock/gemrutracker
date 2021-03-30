package main

import (
	"errors"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
)

type Config struct {
	WorkDir                   string
	HttpProxyAddress          string
	ListenAddress             string
	UpdatePeriodHours         int
	LogLevel                  logrus.Level
	UpdateForwardRecordCount  int
	UpdateBackwardRecordCount int
	ParserThreadCount         int
	CertFilePath              string
	KeyFilePath               string
}

var config *Config

func initConfig(filePath string) error {
	if filePath == "" {
		filePath = defaultConfigFilePath
	}

	logrus.WithField("filePath", filePath).Info("Чтение файла настроек...")

	_, err := toml.DecodeFile(filePath, &config)
	if err != nil {
		return err
	}

	if config.UpdatePeriodHours < 0 {
		config.UpdatePeriodHours = 0
	}
	if config.LogLevel == 0 {
		config.LogLevel = logrus.InfoLevel
	}
	if config.UpdateForwardRecordCount <= 0 {
		config.UpdateForwardRecordCount = 1000
	}
	if config.UpdateBackwardRecordCount <= 0 {
		config.UpdateBackwardRecordCount = 1000
	}
	if config.ParserThreadCount <= 0 {
		config.ParserThreadCount = runtime.NumCPU()
	}

	if config.CertFilePath == "" {
		return errors.New("не указан путь к сертификату (config.CertFilePath)")
	}
	if config.KeyFilePath == "" {
		return errors.New("не указан путь к ключу (config.KeyFilePath)")
	}

	return nil
}
