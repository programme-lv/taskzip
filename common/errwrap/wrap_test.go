package errwrap_test

import (
	"fmt"
	"testing"

	"github.com/programme-lv/task-zip/common/errwrap"
	"github.com/stretchr/testify/require"
)

func TestClientErrorWrapping(t *testing.T) {
	err := errwrap.Error("client supplied bad request")
	err2 := errwrap.AddTrace(err)
	err3 := errwrap.Unexpected("test", err2)
	err4 := errwrap.AddTrace(err3)
	err5 := errwrap.Unexpected("test", err4)
	err6 := errwrap.AddTrace(err5)
	err7 := errwrap.Unexpected("test", err6)
	err8 := errwrap.AddTrace(err7)
	err9 := errwrap.Unexpected("server error", fmt.Errorf("some error"))
	err10 := errwrap.AddTrace(err9)
	fmt.Println(err8)
	_, ok := errwrap.ExtractClientError(err9)
	require.False(t, ok)
	_, ok = errwrap.ExtractClientError(err10)
	require.False(t, ok)
	msg, ok := errwrap.ExtractClientError(err8)
	require.True(t, ok)
	require.Equal(t, "client supplied bad request", msg)

}
