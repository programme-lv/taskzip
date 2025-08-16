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
