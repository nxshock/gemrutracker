package main

import (
	"os"
	"path/filepath"

	"github.com/nxshock/fts"
	"github.com/nxshock/zkv"
)

type Info struct {
	Title      string
	MagnetData string
	ForumName  string
}

var (
	db    *zkv.Db
	index *fts.Index
)

func initDb(workDir string) error {
	var err error
	db, err = zkv.OpenWithConfig(filepath.Join(config.WorkDir, "db.zkv"), &zkv.Config{BlockDataSize: 64 * 1024})
	if err != nil {
		return err
	}

	index, err = fts.Open(filepath.Join(config.WorkDir, "index.dat"))
	if err != nil {
		return err
	}

	return nil
}

func search(query string) ([]Info, error) {
	ids, err := index.Search(query)
	if err != nil {
		return nil, err
	}

	var result []Info

	for _, id := range ids {
		var data Info
		err = db.Get(id, &data)
		if err != nil {
			return nil, err
		}

		result = append(result, data)
	}

	return result, nil
}

func lastDbId() (int, error) {
	var maxId int
	err := db.Get("maxId", &maxId)
	if err != nil && err == zkv.ErrNotFound {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return maxId, nil
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
