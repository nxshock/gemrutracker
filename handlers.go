package main

import (
	"fmt"
	"io"

	"github.com/nxshock/gemini"
	"github.com/sirupsen/logrus"
)

func searchHandler(w gemini.ResponseWriter, r *gemini.Request) {
	logrus.WithFields(logrus.Fields{
		"Query":      r.Query,
		"RemoteAddr": r.RemoteAddr}).Debugln("Запрошен /search")

	if r.Query == "" {
		w.WriteHeader(gemini.StatusInput, "Введите запрос:")
		return
	}

	fmt.Fprintln(w, "# Результаты по запросу:", r.Query)

	results, err := search(r.Query)
	if err != nil {
		logrus.WithField("query", r.Query).Errorln(err)
		gemini.Error(w, err.Error(), gemini.StatusTemporaryFailure)
		return
	}

	var currentForum string
	for _, v := range results {
		if v.ForumName != currentForum {
			fmt.Fprintln(w, "##", v.ForumName)

			currentForum = v.ForumName
		}
		fmt.Fprintf(w, "=> %s %s\n", v.MagnetData, v.Title)
	}

	fmt.Fprintln(w, "## Навигация")
	fmt.Fprintln(w, "=> search ❓ Другой запрос")
	fmt.Fprintln(w, "=> / 🏠 Домой")
}

func handle(w gemini.ResponseWriter, r *gemini.Request) {
	logrus.WithField("RemoteAddr", r.RemoteAddr).Debugln("Запрошен /")

	f, err := content.Open("site/index.gmi")
	if err != nil {
		logrus.Errorln(err)
		gemini.Error(w, err.Error(), gemini.StatusTemporaryFailure)
		return
	}
	defer f.Close()

	io.Copy(w, f)
}
