package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSampleFilesReadPlainAndCompressedRecords(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		name := "plain"
		if compressed {
			name = "gzip"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sample.json")
			want := []map[string]int{{"stage": 14, "victories": 1}}
			if compressed {
				saveSamples(path, want)
				path += ".gz"
			} else {
				save(path, want)
			}
			var got []map[string]int
			require.NoError(t, readSamples(path, &got))
			assert.Equal(t, want, got)
		})
	}
}
