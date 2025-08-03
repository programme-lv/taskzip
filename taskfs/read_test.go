package taskfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	task, err := Read("testdata/kvadrputekl", WithCheckAllFilesRead(false))
	require.NoError(t, err)

	require.Equal(t, "kvadrputekl", task.ShortID)

	require.Equal(t, "Kvadrātveida putekļsūcējs", task.FullName["lv"])
	require.Equal(t, "Square vacuum cleaner", task.FullName["en"])
	require.Equal(t, 2, len(task.FullName))

	require.Contains(t, task.ReadMe, "this is an example readme.md file")
}
