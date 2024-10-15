package file_handlers

import (
	"fmt"
	"math/rand"
	"os"
)

func SaveDataInneficient(path string, data []byte) error {
	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = fp.Write(data)
	if err != nil {
		return err
	}
	return fp.Sync()

	// Limitations:
	// 1. Update content as a whole. Will not work for large data
	// 2. Read and modify data in memory.
}

func SaveDataSlightlyEfficient(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, rand.Int31())
	fp, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer func() {
		err := fp.Close()
		if err != nil {
			os.Remove(tmp)
		}
	}()

	_, err = fp.Write(data)
	if err != nil {
		return err
	}
	err = fp.Sync()
	if err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
