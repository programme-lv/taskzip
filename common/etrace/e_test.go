package etrace_test

import (
	"errors"
	"io"
	"testing"

	"github.com/programme-lv/task-zip/common/etrace"
	"github.com/stretchr/testify/assert"
)

func TestNew_ErrorString(t *testing.T) {
	e := etrace.New(etrace.Warning, "oops")
	assert.Equal(t, "oops", e.Error())
	assert.Nil(t, errors.Unwrap(e))
}

func TestWrap_NonEtrace(t *testing.T) {
	base := errors.New("boom")
	e := etrace.Wrap("ctx", base)
	assert.Equal(t, "ctx: internal error", e.Error())
	assert.True(t, errors.Is(e, base))
	assert.Equal(t, base, errors.Unwrap(e))
	dbg := e.Debug()
	assert.Contains(t, dbg, "stack trace:")
	assert.Contains(t, dbg, "cause:")
	assert.Contains(t, dbg, "ctx")
	assert.Contains(t, dbg, "boom")
}

func TestWrap_Existing(t *testing.T) {
	e := etrace.New(etrace.Warning, "s")
	e = etrace.Wrap("first", e)
	e = etrace.Wrap("second", e)
	assert.Equal(t, "second: first: s", e.Error())
}

func TestTrace_NonEtrace(t *testing.T) {
	base := errors.New("b")
	e := etrace.Trace(base)
	assert.Equal(t, "internal error", e.Error())
	assert.True(t, errors.Is(e, base))
	assert.Equal(t, base, errors.Unwrap(e))
	dbg := e.Debug()
	assert.Contains(t, dbg, "stack trace:")
	assert.Contains(t, dbg, "cause:")
	assert.Contains(t, dbg, "b")
}

func TestIs_DelegatesToCause(t *testing.T) {
	e := etrace.Wrap("w", io.EOF)
	assert.True(t, errors.Is(e, io.EOF))
}

func TestIsCritical_NilError(t *testing.T) {
	assert.False(t, etrace.IsCritical(nil))
}

func TestIsCritical_SingleWarning(t *testing.T) {
	w := etrace.NewWarning("warning message")
	assert.False(t, etrace.IsCritical(w))
}

func TestIsCritical_SingleCritical(t *testing.T) {
	e := etrace.NewError("critical error")
	assert.True(t, etrace.IsCritical(e))
}

func TestIsCritical_NonEtraceError(t *testing.T) {
	err := errors.New("regular error")
	assert.True(t, etrace.IsCritical(err))
}

func TestIsCritical_JoinedOnlyWarnings(t *testing.T) {
	w1 := etrace.NewWarning("warning 1")
	w2 := etrace.NewWarning("warning 2")
	joined := errors.Join(w1, w2)
	assert.False(t, etrace.IsCritical(joined))
}

func TestIsCritical_JoinedWithCritical(t *testing.T) {
	w1 := etrace.NewWarning("warning 1")
	e1 := etrace.NewError("critical error")
	w2 := etrace.NewWarning("warning 2")
	joined := errors.Join(w1, e1, w2)
	assert.True(t, etrace.IsCritical(joined))
}

func TestIsCritical_WrappedWarning(t *testing.T) {
	w := etrace.NewWarning("base warning")
	wrapped := etrace.Wrap("context", w)
	assert.False(t, etrace.IsCritical(wrapped))
}

func TestIsCritical_WrappedCritical(t *testing.T) {
	e := etrace.NewError("base error")
	wrapped := etrace.Wrap("context", e)
	assert.True(t, etrace.IsCritical(wrapped))
}
