package errwrap

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

// ServerError wraps internal errors (not client's fault)
func ServerError(msg string, err error) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	if err != nil {
		return fmt.Errorf("[%s/%s:%d] %s\n%w", dir, file, line, msg, err)
	} else {
		return fmt.Errorf("[%s/%s:%d] %s", dir, file, line, msg)
	}
}

// ClientError initiates bad request (validation) errors
func ClientError(msg string) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	err := ClientErrorType{msg: msg}
	return fmt.Errorf("[%s/%s:%d] %w", dir, file, line, err)
}

// AddTrace wraps an error with the file and line number of the caller
func AddTrace(err error) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	return fmt.Errorf("[%s/%s:%d] ...\n%w", dir, file, line, err)
}

func ExtractClientError(err error) (string, bool) {
	var clientError ClientErrorType
	if errors.As(err, &clientError) {
		return clientError.msg, true
	}
	return "", false
}

type ClientErrorType struct {
	msg string // in a client error, msg is public
}

func (e ClientErrorType) Error() string {
	return e.msg
}
