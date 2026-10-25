package scenemap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(gfx, "maps", "S_*.json"))
	require.NoError(t, err)
	seen := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		m, v2, err := Parse(data)
		require.NoError(t, err, path)
		if !v2 {
			continue
		}
		seen++
		t.Run(filepath.Base(path), func(t *testing.T) {
			out, err := m.Encode()
			require.NoError(t, err)
			assert.Equal(t, string(data), string(out), "written back as the file lays it out")
		})
	}
	assert.NotZero(t, seen)
}
