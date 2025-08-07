package errwrap

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// Wrap adds a message and a trace to the error
func Wrap(msg string, err error) *tracedError {
	_, file, line, _ := runtime.Caller(1)
	path := srcFilePath(file)
	return &tracedError{
		file: path,
		line: line,
		err:  err,
		msg:  msg,
	}
}

// Trace adds a trace - file path and line number - to the error
func Trace(err error) *tracedError {
	_, file, line, _ := runtime.Caller(1)
	path := srcFilePath(file)
	return &tracedError{
		file: path,
		line: line,
		err:  err,
	}
}

// Error defines strict bad request error that can be shown to the user
// Such errors should not include unexpected server errors
func Error(msg string) errorType {
	return errorType{msg: msg}
}

// Warning defines non-fatal errors that can be shown to the user
func Warning(msg string) warningType {
	return warningType{msg: msg}
}

// ExtractError extracts fatal error message from the error if any
func ExtractError(err error) (string, bool) {
	var error errorType
	if errors.As(err, &error) {
		return error.msg, true
	}
	return "", false
}

type errorType struct {
	msg string // note that in this (client) error, msg is public
}

func (e errorType) Error() string {
	return e.msg
}

type warningType struct {
	msg string
}

func (w warningType) Error() string {
	return w.msg
}

func getLeafErrors(err error) []error {
	leafErrors := []error{}
	if uw, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range uw.Unwrap() {
			leafErrors = append(leafErrors, getLeafErrors(e)...)
		}
	} else if uw, ok := err.(interface{ Unwrap() error }); ok {
		leafErrors = append(leafErrors, getLeafErrors(uw.Unwrap())...)
	} else {
		leafErrors = append(leafErrors, err)
	}
	return leafErrors
}

// IsCritical returns whether there is an error that isnt a warning
func IsCritical(err error) bool {
	for _, e := range getLeafErrors(err) {
		if !errors.As(e, &warningType{}) {
			return true
		}
	}
	return false
}

// GetWarnings returns all warning messages from the error
func GetWarnings(err error) []string {
	warnings := []string{}
	for _, e := range getLeafErrors(err) {
		w := warningType{}
		if errors.As(e, &w) {
			warnings = append(warnings, w.msg)
		}
	}
	return warnings
}

// by default we should not be attaching a trace to an error
// we should

type leafError struct {
	msg   string
	trace []struct {
		file string
		line int
		msg  string
	}
}

type tracedError struct {
	file string
	line int
	msg  string

	err error
}

func (e tracedError) Error() string {
	return e.err.Error()
}

func (e tracedError) Is(target error) bool {
	// does support Is?
	if uw, ok := e.err.(interface{ Is(error) bool }); ok {
		return uw.Is(target)
	}
	return false
}

func (e tracedError) Unwrap() error {
	return e.err
}

func (e tracedError) Traced() string {
	// so we get a list of all errors and as we dfs we pass a trace to the error
	// okay, so first of all... if the error does not support unwrap, we just return the error
	return ""
}

func srcFilePath(absPath string) string {
	wd, err := os.Getwd()
	if err != nil {
		return absPath
	}
	relPath, err := filepath.Rel(wd, absPath)
	if err != nil {
		return absPath
	}
	return relPath
}
