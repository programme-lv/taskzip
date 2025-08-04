package taskfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func wrap(msg string, errs ...error) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	if len(errs) > 0 {
		err := errors.Join(errs...)
		return fmt.Errorf("[%s/%s:%d] %s\n%w", dir, file, line, msg, err)
	} else {
		err := errors.New(msg)
		return fmt.Errorf("[%s/%s:%d] %w", dir, file, line, err)
	}
}

func filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func mapSlice[S any, T any](ss []S, f func(S) T) []T {
	res := make([]T, len(ss))
	for i, s := range ss {
		res[i] = f(s)
	}
	return res
}

func doesDirExist(dirPath string) bool {
	_, err := os.Stat(dirPath)
	return err == nil
}
