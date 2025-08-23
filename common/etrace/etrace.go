package etrace

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
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
	Fname string   // fname where error is defined
	Line  int      // line where error is defined
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

func (e Error) WithInternalCause(cause error) Error {
	e.cause = errors.Join(e.cause, cause)
	return e
}

// Debug returns a string with full stack trace
// and error cause. Trace entries are on newlines.
func (e Error) Debug() string {
	result := e.Error()
	if e.Fname != "" {
		result += fmt.Sprintf(" (%s:%d)", e.Fname, e.Line)
	}
	result += "\n"
	if len(e.trace) > 0 {
		numTraceEntries := len(e.trace)
		if numTraceEntries == 1 {
			t := e.trace[0]
			if t.WrapM == "" {
				result += fmt.Sprintf("\t- trace: %s:%d\n", t.Fname, t.Line)
			} else {
				result += fmt.Sprintf("\t- trace: %s:%d %s\n", t.Fname, t.Line, t.WrapM)
			}
		} else {
			result += "\t- trace:\n"
			numDigits := len(fmt.Sprintf("%d", numTraceEntries))
			for i := len(e.trace) - 1; i >= 0; i-- {
				t := e.trace[i]
				lineNum := len(e.trace) - i
				if t.WrapM == "" {
					result += fmt.Sprintf(
						"\t\t%*d. %s:%d\n",
						numDigits, lineNum, t.Fname, t.Line)
					continue
				}
				result += fmt.Sprintf(
					"\t\t%*d. %s:%d %s\n",
					numDigits, lineNum, t.Fname, t.Line, t.WrapM)
			}
		}
	}
	if e.cause != nil {
		result += "\t- cause:"
		debugStr := GetDebugStr(e.cause)
		debugStr = strings.TrimPrefix(debugStr, "ERROR:")
		debugStr = strings.TrimPrefix(debugStr, "WARN:")
		if len(strings.Split(debugStr, "\n")) > 1 {
			debugStr = prefixWithTabs(debugStr)
			result += fmt.Sprintf("\n%s", debugStr)
		} else {
			result += fmt.Sprintf(" %s\n", debugStr)
		}
	}
	return result
}

func PrintDebug(err error) {
	fmt.Println(GetDebugStr(err))
}

func prefixWithTabs(str string) string {
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		lines[i] = "\t" + line
	}
	return strings.Join(lines, "\n")
}

func GetDebugStr(err error) string {
	if err == nil {
		return ""
	}

	if u, ok := err.(interface{ Unwrap() []error }); ok {
		errs := u.Unwrap()
		result := ""
		for _, e := range errs {
			result += GetDebugStr(e)
		}
		return result
	}

	var etraceErr Error
	if errors.As(err, &etraceErr) {
		return etraceErr.Debug()
	}

	// if u, ok := err.(interface{ Unwrap() error }); ok {
	// 	return GetDebugStr(u.Error())
	// }

	return err.Error()
}

func (e Error) Unwrap() error {
	initialMsg := e.Error()
	for len(e.trace) > 0 && e.Error() != e.summ && e.Error() == initialMsg {
		e.trace = e.trace[:len(e.trace)-1]
	}
	if len(e.trace) == 0 && e.summ == "internal error" {
		return e.cause
	}
	return e.cause
}

func (e Error) Is(target error) bool {
	if eTraceErr, ok := target.(Error); ok {
		if eTraceErr.summ == e.summ &&
			eTraceErr.level == e.level &&
			eTraceErr.cause == e.cause {
			return true
		}
	}
	if e.Error() == target.Error() {
		return true
	}
	return errors.Is(e.Unwrap(), target)
}

// Severity returns the severity level of the error
func (e Error) Severity() Severity {
	return e.level
}

// New is a constructor for expected errors.
// Define new errors in global scope to avoid
// unexpected runtime panics (that may be added).
func New(lvl Severity, msg string) Error {
	_, file, line, _ := runtime.Caller(1)
	return Error{
		level: lvl,
		summ:  msg,
		Fname: file,
		Line:  line,
	}
}

func NewWarning(msg string) Error {
	_, file, line, _ := runtime.Caller(1)
	return Error{
		level: Warning,
		summ:  msg,
		Fname: file,
		Line:  line,
	}
}

func NewError(msg string) Error {
	_, file, line, _ := runtime.Caller(1)
	return Error{
		level: Critical,
		summ:  msg,
		Fname: file,
		Line:  line,
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

// IsCritical returns true if the error contains any critical errors.
// Returns false if the error is nil or contains only warnings.
//
// This function traverses the entire error tree (including errors.Join
// and wrapped errors) and returns true if any leaf error is either:
// - An etrace.Error with Critical severity
// - A non-etrace error (which are considered critical by default)
//
// Use this to distinguish between validation warnings that can be
// ignored and critical errors that should cause operation failure.
func IsCritical(err error) bool {
	if err == nil {
		return false
	}

	// Check all leaf errors in the error tree
	for _, leafErr := range getLeafErrors(err) {
		var etraceErr Error
		if errors.As(leafErr, &etraceErr) {
			if etraceErr.Severity() == Critical {
				return true
			}
		} else {
			// Non-etrace errors are considered critical
			return true
		}
	}

	return false
}

// getLeafErrors returns all leaf errors from an error tree,
// unwrapping joined errors and wrapped errors recursively.
func getLeafErrors(err error) []error {
	if err == nil {
		return nil
	}

	var leafErrors []error

	// Check if error supports multiple unwrapping (errors.Join)
	if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range unwrapper.Unwrap() {
			leafErrors = append(leafErrors, getLeafErrors(e)...)
		}
		return leafErrors
	}

	// Check if error supports single unwrapping
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		unwrapped := unwrapper.Unwrap()
		if unwrapped != nil {
			return getLeafErrors(unwrapped)
		}
		// If unwrapped is nil, treat this error as a leaf
	}

	// This is a leaf error
	return []error{err}
}
