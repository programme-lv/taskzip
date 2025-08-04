package taskfs

import "os"

func Write(task Task, dirPath string) error {
	// does dir exist?
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		return wrap("dir already exists")
	}

	return nil
}
