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
	err2 := errwrap.Trace(err)
	err3 := errwrap.Wrap("test", err2)
	err4 := errwrap.Trace(err3)
	err5 := errwrap.Wrap("test", err4)
	err6 := errwrap.Trace(err5)
	err7 := errwrap.Wrap("test", err6)
	err8 := errwrap.Trace(err7)
	err9 := errwrap.Wrap("server error", fmt.Errorf("some error"))
	err10 := errwrap.Trace(err9)
	fmt.Println(err8)
	_, ok := errwrap.ExtractError(err9)
	require.False(t, ok)
	_, ok = errwrap.ExtractError(err10)
	require.False(t, ok)
	msg, ok := errwrap.ExtractError(err8)
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

	warnings := errwrap.GetWarnings(errs)
	require.Len(t, warnings, 2)
	require.Contains(t, warnings, taskfs.WarnUnknownOlympStage)
	require.Contains(t, warnings, taskfs.WarnOriginNoteTooLong)
}
