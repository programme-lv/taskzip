package taskfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadId(t *testing.T) {
	task, err := Read("testdata/kvadrputekl", WithCheckAllFilesRead(false))
	require.NoError(t, err)
	require.Equal(t, "kvadrputekl", task.ShortID)
}
