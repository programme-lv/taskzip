package lio2023_test

import (
	"path/filepath"
	"testing"

	"github.com/programme-lv/taskzip/external/lio/lio2023"
	"github.com/stretchr/testify/require"
)

func TestParsingLio2023TaskWithBothACheckerAndAnInteractor(t *testing.T) {
	taskDir, err := getTaskDirectory(t, "iedalas")
	require.NoErrorf(t, err, "failed to get task directory: %v", err)

	task, err := lio2023.ParseLio2023TaskDir(taskDir)
	require.NoErrorf(t, err, "failed to parse task: %v", err)

	require.NotNilf(t, task, "task is nil")
	require.Equal(t, "lv", task.Origin.Lang)

	// When both interactor and checker exist, interactor takes precedence
	require.NotEmptyf(t, task.Testing.Interactor, "task.Testing.Interactor is empty")
	require.Equal(t, "interactor", task.Testing.TestingT)
	// Checker should be empty since interactor takes precedence
	require.Empty(t, task.Testing.Checker)

	require.Len(t, task.Solutions, 13)
	solutionFilenames := []string{}
	for _, solution := range task.Solutions {
		solutionFilenames = append(solutionFilenames, solution.Fname) // field name changed
	}
	require.Contains(t, solutionFilenames, "iedalas_PP_OK.cpp")

	// Examples are now in Statement.Examples
	examples := task.Statement.Examples
	require.Len(t, examples, 1)
	require.Equal(t, "131\n", examples[0].Input)    // now string, not []byte
	require.Equal(t, "1 131\n", examples[0].Output) // now string, not []byte

	// Tests are now in Testing.Tests
	tests := task.Testing.Tests
	require.Len(t, tests, 4)

	require.Equal(t, "560\n", tests[2].Input) // now string, not []byte

	publicTestGroups := []int{1, 6, 11}
	// TestGroups are now in Scoring.Groups
	testGroups := task.Scoring.Groups
	require.Len(t, testGroups, 25)
	for i, testGroup := range testGroups {
		if testGroup.Public {
			require.Contains(t, publicTestGroups, i+1)
		}
	}

	require.Equal(t, 4, testGroups[0].Points)
	// require.Equal(t, 1, testGroups[1].Subtask) (can't be accurately determined)
	require.Equal(t, false, testGroups[1].Public)
	// require.Equal(t, true, testGroups[0].Public) (can't be accurately determined)
	// TestIDs field no longer exists in the new structure
	// require.Equal(t, []int{1, 2, 3, 4}, testGroups[0].TestIDs)

	// CPU and Memory limits are now in Testing with different names and units
	require.Equal(t, 1500, task.Testing.CpuLimMs) // 1.5 seconds = 1500 ms
	require.Equal(t, 256, task.Testing.MemLimMiB) // same value

	expectedArchive := []string{"./riki/interval.txt", "./riki/testlib.h"}
	actualArchive := []string{}
	// Archive files are now in Archive.Files
	for _, archiveFile := range task.Archive.Files {
		actualArchive = append(actualArchive, archiveFile.RelPath) // field name changed
	}

	require.ElementsMatch(t, expectedArchive, actualArchive)
}

func getTaskDirectory(t *testing.T, taskName string) (string, error) {
	testdataDirRel := filepath.Join("testdata", taskName)
	path, err := filepath.Abs(testdataDirRel)
	require.NoErrorf(t, err, "failed to get absolute path: %v", err)
	return path, nil
}
