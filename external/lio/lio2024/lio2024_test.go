package lio2024_test

import (
	"path/filepath"
	"testing"

	"github.com/programme-lv/taskzip/common/etrace"
	"github.com/programme-lv/taskzip/common/zips"
	"github.com/programme-lv/taskzip/external/lio/lio2024"
	"github.com/programme-lv/taskzip/taskfs"
	"github.com/stretchr/testify/require"
)

func TestParsingLio2024TaskWithoutAChecker(t *testing.T) {
	tmpDir := t.TempDir()
	err := zips.Unzip("testdata/kp.zip", tmpDir)
	require.NoError(t, err)

	taskDir := filepath.Join(tmpDir, "kp")
	task, err := lio2024.ParseLio2024TaskDir(taskDir)
	require.NoError(t, err)

	err = task.Validate()
	// etrace.PrintDebug(err)
	require.False(t, etrace.IsCritical(err))
	require.ErrorIs(t, err, taskfs.WarnStageNotSet)

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
	require.Equal(t, "lv", task.Origin.Lang)

	// testing
	require.Equal(t, "simple", task.Testing.TestingT)
	require.Equal(t, 500, task.Testing.CpuLimMs)
	require.Equal(t, 256, task.Testing.MemLimMiB)
	require.Equal(t, 15, len(task.Testing.Tests))
	require.Equal(t, "", task.Testing.Checker)
	require.Equal(t, "", task.Testing.Interactor)

	// statement
	require.Len(t, task.Statement.Stories, 1)
	require.Contains(t, task.Statement.Stories, "lv")
	require.Equal(t, "TODO", task.Statement.Stories["lv"].Story)
	sumOfStPoints := 0
	for _, subtask := range task.Statement.Subtasks {
		sumOfStPoints += subtask.Points
	}
	require.Equal(t, 100, sumOfStPoints)
	noOfSubtasksWithVisibleInput := 0
	for _, subtask := range task.Statement.Subtasks {
		if subtask.VisInput {
			noOfSubtasksWithVisibleInput++
		}
	}
	require.Equal(t, 1, noOfSubtasksWithVisibleInput)
	require.Len(t, task.Statement.Examples, 1)
	require.Len(t, task.Statement.Images, 3)
	subtasks := task.Statement.Subtasks
	require.Len(t, subtasks, 4)
	require.Equal(t, 3, subtasks[0].Points)
	require.Equal(t, 48, subtasks[1].Points)
	require.Equal(t, 28, subtasks[2].Points)
	require.Equal(t, 21, subtasks[3].Points)

	// readme
	require.Contains(t, task.ReadMe, "- [ ] specify the year & stage of the olympiad in task.toml")
	require.Contains(t, task.ReadMe, "- [ ] paste descriptive note of the olympiad in task.toml")
	require.Contains(t, task.ReadMe, "- [ ] port statement from .typ to .md in statement dir")
	require.Contains(t, task.ReadMe, "- [ ] subtask descriptions from .typ to task.toml")
	require.Contains(t, task.ReadMe, "- [ ] example notes from .typ to .md in example dir")
	require.Contains(t, task.ReadMe, "- [ ] should list the authors in origin in task.toml")
	require.Contains(t, task.ReadMe, "- [ ] determine difficulty based on # of ACs in contest")
	require.Contains(t, task.ReadMe, "- [ ] specify which subtasks should each model solution solve")

	// solutions
	require.Len(t, task.Solutions, 3)
	require.Equal(t, "kp_kp_ok.cpp", task.Solutions[0].Fname)
	require.Equal(t, "kp_kp_tle.cpp", task.Solutions[1].Fname)
	require.Equal(t, "kp_nv.cpp", task.Solutions[2].Fname)
	require.Equal(t, []int{1, 2, 3, 4}, task.Solutions[0].Subtasks)
	require.Equal(t, []int{}, task.Solutions[1].Subtasks)
	require.Equal(t, []int{}, task.Solutions[2].Subtasks)

	// archive
	require.Greater(t, len(task.Archive.Files), 20)

	require.NotEmpty(t, task.Archive.GetTestlibValidator())
	require.NotEmpty(t, task.Archive.GetOgStatementPdfs())
}

func TestParsingLio2024TaskWithAnInteractor(t *testing.T) {
	tmpDir := t.TempDir()
	err := zips.Unzip("testdata/uzmini.zip", tmpDir)
	require.NoError(t, err)

	taskDir := filepath.Join(tmpDir, "uzmini")
	task, err := lio2024.ParseLio2024TaskDir(taskDir)
	require.NoError(t, err)

	// testing
	require.Equal(t, "interactor", task.Testing.TestingT)
	require.Empty(t, task.Testing.Checker)
	require.NotEmpty(t, task.Testing.Interactor)

	// subtasks
	require.False(t, task.Statement.Subtasks[0].VisInput)
}
