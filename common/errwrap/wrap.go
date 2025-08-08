package errwrap

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

// Wrap adds a message and a trace to the error
func Wrap(msg string, err error) TracedError {
	_, file, line, _ := runtime.Caller(1)
	path := srcFilePath(file)
	return TracedError{
		file: path,
		line: line,
		err:  err,
		msg:  msg,
	}
}

// Trace adds a trace - file path and line number - to the error
func Trace(err error) TracedError {
	_, file, line, _ := runtime.Caller(1)
	path := srcFilePath(file)
	return TracedError{
		file: path,
		line: line,
		err:  err,
	}
}

// Error defines strict bad request error that can be shown to the user
// Such errors should not include unexpected server errors
func Error(msg string) ErrorType {
	return ErrorType{msg: msg}
}

// Warning defines non-fatal errors that can be shown to the user
func Warning(msg string) WarningType {
	return WarningType{msg: msg}
}

// ExtractError extracts fatal error message from the error if any
func ExtractError(err error) (string, bool) {
	var error ErrorType
	if errors.As(err, &error) {
		return error.msg, true
	}
	return "", false
}

type ErrorType struct {
	msg string // note that in this (client) error, msg is public
}

func (e ErrorType) Error() string {
	return e.msg
}

type WarningType struct {
	msg string
}

func (w WarningType) Error() string {
	return w.msg
}

func GetErrorMsg(err error) string {
	e := ErrorType{}
	if errors.As(err, &e) {
		return e.msg
	}
	w := WarningType{}
	if errors.As(err, &w) {
		return w.msg
	}
	return err.Error()
}

func GetLeafErrors(err error) []error {
	leafErrors := []error{}
	if uw, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range uw.Unwrap() {
			leafErrors = append(leafErrors, GetLeafErrors(e)...)
		}
	} else if uw, ok := err.(interface{ Unwrap() error }); ok {
		leafErrors = append(leafErrors, GetLeafErrors(uw.Unwrap())...)
	} else {
		leafErrors = append(leafErrors, err)
	}
	return leafErrors
}

// IsCritical returns whether there is an error that isnt a warning
func IsCritical(err error) bool {
	for _, e := range GetLeafErrors(err) {
		if !errors.As(e, &WarningType{}) {
			return true
		}
	}
	return false
}

// GetWarnings returns all warning messages from the error
func GetWarnings(err error) []string {
	warnings := []string{}
	for _, e := range GetLeafErrors(err) {
		w := WarningType{}
		if errors.As(e, &w) {
			warnings = append(warnings, w.msg)
		}
	}
	return warnings
}

// by default we should not be attaching a trace to an error
// we should

type TracedError struct {
	file string
	line int
	msg  string

	err error
}

func (e TracedError) Error() string {
	path := srcFilePath(e.file)
	if e.msg != "" {
		return fmt.Sprintf("[%s:%d] %s -> %s", path, e.line, e.msg, e.err.Error())
	} else {
		if _, ok := e.err.(TracedError); ok {
			return fmt.Sprintf("[%s:%d] ... -> %s", path, e.line, e.err.Error())
		} else {
			return fmt.Sprintf("[%s:%d] %s", path, e.line, e.err.Error())
		}
	}
}

func (e TracedError) Is(target error) bool {
	if uw, ok := e.err.(interface{ Is(error) bool }); ok {
		return uw.Is(target)
	}
	return false
}

func (e TracedError) Unwrap() error {
	return e.err
}

func (e TracedError) Trace() string {
	panic("not implemented")
	// so we get a list of all errors and as we dfs we pass a trace to the error
	// okay, so first of all... if the error does not support unwrap, we just return the error
	// traces := trace(e)
	// traceStr := ""
	// for _, t := range traces {
	// 	for _, e := range t.trace {
	// 		traceStr += fmt.Sprintf("%s:%d: %s\n", e.file, e.line, e.msg)
	// 	}
	// }
	// return traceStr
}

type traceEntry struct {
	file string
	line int
	msg  string
}

type fullTracedError struct {
	trace []traceEntry
	err   error
}

func trace(err error) []fullTracedError {
	res := []fullTracedError{}
	if uw, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range uw.Unwrap() {
			result := trace(e)
			if e, ok := e.(TracedError); ok {
				for _, t := range result {
					t.trace = append(t.trace, traceEntry{
						file: e.file,
						line: e.line,
						msg:  e.msg,
					})
				}
			} else {
				res = append(res, result...)
			}
		}
	} else if uw, ok := err.(interface{ Unwrap() error }); ok {
		result := trace(uw.Unwrap())
		if e, ok := err.(TracedError); ok {
			for _, t := range result {
				t.trace = append(t.trace, traceEntry{
					file: e.file,
					line: e.line,
					msg:  e.msg,
				})
			}
		}
		res = append(res, result...)
	} else {
		if e, ok := err.(TracedError); ok {
			res = append(res, fullTracedError{
				trace: []traceEntry{
					{
						file: e.file,
						line: e.line,
						msg:  e.msg,
					},
				},
				err: err,
			})
		}
	}
	return res
}

func srcFilePath(absPath string) string {
	parent := filepath.Dir(absPath)
	base := filepath.Base(absPath)
	return filepath.Join(filepath.Base(parent), base)
}
