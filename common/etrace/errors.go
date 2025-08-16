package etrace

import (
	"errors"
	"fmt"
	"runtime"
)

type Severity int

const (
	Warning Severity = iota
	Critical
)

/* NO "INFO" SEVERITY
We skip severity level "info". It's not an error.
*/

type eTrace struct {
	Fname string // filename. for developers
	Line  int    // line of code. for developers
	WrapM string // user-facing wrap msg, if any
}

/* "USER-FACING" MEANS SAFE
By user-facing we mean human-facing and secure
as it must not leak e.g. system environment info.
*/

type Error struct {
	level Severity // how to handle and format
	summ  string   // short, user-facing summary
	cause error    // do not show to the end-user
	trace []eTrace // [0] is oldest, last is new
}

/* ERROR "CODE" REMOVED
For now, I've removed the machine-readable "code"
attribute to simplify. Error struct can be evolved
over time. It's also NOT expected that error
will be re-interpreted by another system soon.
*/

/* ERROR SUMMARY TRANSLATION
In general, error "summ" could be a function that
accepts a language code and then returns a string
For this library, we will assume english.
*/

/* ERROR "CAUSE" IS FOR DEVELOPERS
Error attributes "code" and "cause" are not
meant to be user-facing. Error "cause" can be
added later or remain empty if the error is
self-contained. In general, clients can be
provided with an event identifier to contact
support instead of stack trace. "Cause" is
private to the developers of the system.
*/

// Error returns summary + wrap messages without
// further information such as cause and trace.
func (e Error) Error() string {
	result := e.summ
	for _, t := range e.trace {
		if t.WrapM != "" {
			result = fmt.Sprintf(
				"%s: %s",
				t.WrapM, result)
		}
	}
	return result
}

// Debug returns a string with full stack trace
// and error cause. Trace entries are on newlines.
func (e Error) Debug() string {
	result := e.summ
	result += "\n\nstack trace:\n"
	for _, t := range e.trace {
		result += fmt.Sprintf(
			"\t%s:%d: %s\n",
			t.Fname, t.Line, t.WrapM)
	}
	result += "\n\ncause:\n"
	result += e.cause.Error()
	return result
}

func (e Error) Unwrap() error {
	return e.cause
}

func (e Error) Is(target error) bool {
	return errors.Is(e.cause, target)
}

// New is a constructor for expected errors.
// Define new errors in global scope to avoid
// unexpected runtime panics (that may be added).
func New(lvl Severity, msg string) Error {
	return Error{
		level: lvl,
		summ:  msg,
	}
}

func NewWarning(msg string) Error {
	return Error{
		level: Warning,
		summ:  msg,
	}
}

func NewError(msg string) Error {
	return Error{
		level: Critical,
		summ:  msg,
	}
}

func Wrap(msg string, err error) Error {
	// if is Error, then we add to it's trace
	// otherwise, new Error with critical severity

	var existing Error
	if errors.As(err, &existing) {
		_, file, line, _ := runtime.Caller(1)
		existing.trace = append(existing.trace, eTrace{
			Fname: file,
			Line:  line,
			WrapM: msg,
		})
		return existing
	}

	_, file, line, _ := runtime.Caller(1)
	return Error{
		level: Critical,
		summ:  "internal error",
		cause: err,
		trace: []eTrace{{
			Fname: file,
			Line:  line,
			WrapM: msg,
		}},
	}
}

func Trace(err error) Error {
	// if is Error, then we add to it's trace
	// otherwise, new Error with critical severity

	var existing Error
	if errors.As(err, &existing) {
		_, file, line, _ := runtime.Caller(1)
		existing.trace = append(existing.trace, eTrace{
			Fname: file,
			Line:  line,
		})
		return existing
	}

	_, file, line, _ := runtime.Caller(1)
	return Error{
		level: Critical,
		summ:  "internal error",
		cause: err,
		trace: []eTrace{{
			Fname: file,
			Line:  line,
		}},
	}
}
