package errwrap

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

// Unexpected wraps internal errors (not client's fault).
// Its the same as a fmt.Errorf but with a trace.
func Unexpected(msg string, err error) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	if err != nil {
		return fmt.Errorf("[%s/%s:%d] %s\n%w", dir, file, line, msg, err)
	} else {
		return fmt.Errorf("[%s/%s:%d] %s", dir, file, line, msg)
	}
}

// Error initiates domain error, caller / client bad request expected errors.
// client / caller errors occur when client violates an invariant.
// Or in other words, they violate a set of assumptions that must
// always be true and are inflexible.
// These are well-formed and can be shown to the user of the service.
func Error(msg string) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	err := ClientErrorType{msg: msg}
	return fmt.Errorf("[%s/%s:%d] %w", dir, file, line, err)
}

// Warning initiates non-fatal errors: stylistic, semantic, etc.
// A task after being trasnformed into a taskfs.Task from an external source
// may contain errors but that does not mean that the execution of the program
// should fail. Warning should be presented to the user so they can fix them.
// Warnings usually indicate missing data.
// A task to be published should have no warning messages.
func Warning(msg string) error {
	_, file, line, _ := runtime.Caller(1)
	dir := filepath.Base(filepath.Dir(file))
	file = filepath.Base(file)
	err := WarningType{msg: msg}
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
	msg string // note that in this (client) error, msg is public
}

func (e ClientErrorType) Error() string {
	return e.msg
}

type WarningType struct {
	msg string
}

func (e WarningType) Error() string {
	return e.msg
}

func IsCritical(err error) bool {
	return err != nil && !errors.Is(err, WarningType{})
}
