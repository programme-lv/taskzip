package lio2024_test

import (
	"testing"

	"github.com/programme-lv/task-zip/external/lio/lio2024"
	"github.com/programme-lv/task-zip/taskfs"
	"github.com/stretchr/testify/require"
)

func TestParsingLio2024TaskWithoutAChecker(t *testing.T) {
	task, err := lio2024.ParseLio2024TaskDir("testdata/kp")
	require.NoError(t, err)
	errs := task.Validate()
	t.Log(errs)
	require.Contains(t, errs, taskfs.WarnUnknownOlympStage.Error())

	// basic info
	require.Equal(t, "kp", task.ShortID)
	require.Equal(t, "Kvadrātveida putekļsūcējs", task.FullName["lv"])

	// metadata
	require.Equal(t, 0, task.Metadata.Difficulty)
	require.Equal(t, []string{}, task.Metadata.ProblemTags)

	// origin
	require.Equal(t, "LIO", task.Origin.Olympiad)
	require.Equal(t, "", task.Origin.OlyStage)
	require.Equal(t, "", task.Origin.Org)
	require.Equal(t, "", task.Origin.Notes["lv"])
}

func TestParsingLio2024TaskWithAChecker(t *testing.T) {
	// TODO: parse task "tornis"
}

func TestParsingLio2024TaskWithAnInteractor(t *testing.T) {
	// TODO: parse task "uzmini"
}
