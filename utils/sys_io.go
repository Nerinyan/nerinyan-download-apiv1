package utils

import (
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func IsExistFile(path string) bool {
	dir := filepath.Dir(path)
	fi, err := os.Stat(dir)
	return err == nil && !fi.IsDir()
}

func IsExistDir(path string) bool {
	dir := filepath.Dir(path)
	fi, err := os.Stat(dir)
	return err == nil && fi.IsDir()
}

func Save2File(data []byte, path string) (err error) {
	logger.Info(path)
	tmp := path + "." + strconv.FormatInt(time.Now().UnixNano(), 16)
	func() {
		f, err := create(tmp)
		if err != nil {
			return
		}
		defer f.Close()
		_, err = f.Write(data)
		if err != nil {
			return
		}
	}()

	return os.Rename(tmp, path)
}

func create(path string) (*os.File, error) {
	dir := filepath.Dir(path)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}
	return os.Create(path)
}
