package taskfs_test

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/programme-lv/taskzip/taskfs"
	"github.com/stretchr/testify/require"
)

func getTestTaskAfterWrite(t *testing.T) *taskfs.Task {
	task, err := taskfs.Read("testdata/kvadrputekl")
	require.NoError(t, err)

	tmpDir, err := os.MkdirTemp(t.TempDir(), "*")
	require.NoError(t, err)
	taskDir := filepath.Join(tmpDir, "task")

	err = taskfs.Write(task, taskDir)
	require.NoError(t, err)

	task, err = taskfs.Read(taskDir)
	require.NoError(t, err)
	return &task
}

func TestWriteBasicInfo(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	BasicInfo(t, task)
}

func TestWriteOrigin(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	Origin(t, task)
}

func TestWriteMetadata(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	Metadata(t, task)
}

func TestWriteSolutions(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	Solutions(t, task)
}

func TestWriteTesting(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	Testing(t, task)
}

func TestWriteStatement(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	Statement(t, task)
}

func TestWriteScoring(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	Scoring(t, task)
}

func TestWriteArchive(t *testing.T) {
	task := getTestTaskAfterWrite(t)
	Archive(t, task)
}

func TestRuneCountInString(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"stāsts", 6},
	}
	for _, test := range tests {
		got := utf8.RuneCountInString(test.input)
		require.Equal(t, test.want, got)
	}
}
