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

	require.Equal(t, "LIO", task.Origin.Olympiad)
	require.Equal(t, "2023/2024", task.Origin.Year)
	require.Equal(t, "school", task.Origin.OlyStage)
	require.Equal(t, "PPS", task.Origin.Org)
	require.Equal(t, []string{"Krišjānis Petručeņa"}, task.Origin.Authors)
	require.Contains(t, task.Origin.Notes["lv"], "Uzdevums no Latvijas 37")
	require.Contains(t, task.Origin.Notes["en"], "The problem is from")
	require.Equal(t, 2, len(task.Origin.Notes))

}
