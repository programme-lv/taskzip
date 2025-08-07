package errwrap_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/programme-lv/task-zip/common/errwrap"
	"github.com/programme-lv/task-zip/taskfs"
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

func TestIsCriticalAndGetAllWarnings(t *testing.T) {
	// Create a mock task with validation warnings
	origin := taskfs.Origin{
		Olympiad: "LIO",
		OlyStage: "",
		Org:      "",
		Notes:    taskfs.I18N[string]{},
		Authors:  []string{},
		Year:     "",
	}
	origin.OlyStage = "abracadabra"
	origin.Notes = make(map[string]string)
	origin.Notes["lv"] = strings.Repeat("a", 201)

	errs := origin.Validate()
	require.ErrorIs(t, errs, taskfs.WarnUnknownOlympStage)
	require.ErrorIs(t, errs, taskfs.WarnOriginNoteTooLong)
	require.False(t, errwrap.IsCritical(errs))

	warnings := errwrap.GetAllWarnings(errs)
	require.Len(t, warnings, 2)
	require.Contains(t, warnings, taskfs.WarnUnknownOlympStage)
	require.Contains(t, warnings, taskfs.WarnOriginNoteTooLong)
}
