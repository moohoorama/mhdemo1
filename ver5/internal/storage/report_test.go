package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestReportRestore(t *testing.T) {
	reports := func(s Store) []os.DirEntry {
		files, _ := os.ReadDir(filepath.Join(s.Dir, "reports"))
		return files
	}

	t.Run("a set of warnings is written once a run, with its state", func(t *testing.T) {
		s := Store{t.TempDir()}
		w := []string{"CorruptSave: unit a: status \"x\" " + t.Name()}
		require.NoError(t, s.ReportRestore(w, core.State{Phase: "battle"}))
		require.NoError(t, s.ReportRestore(w, core.State{Phase: "battle"}))
		files := reports(s)
		require.Len(t, files, 1)
		b, err := os.ReadFile(filepath.Join(s.Dir, "reports", files[0].Name()))
		require.NoError(t, err)
		var r Report
		require.NoError(t, json.Unmarshal(b, &r))
		assert.Equal(t, "restore", r.Kind)
		assert.Equal(t, w, r.Warnings)
		assert.Equal(t, "battle", r.State.Phase)
	})

	t.Run("no more than MaxReports files", func(t *testing.T) {
		s := Store{t.TempDir()}
		for i := 0; i < MaxReports+3; i++ {
			require.NoError(t, s.ReportRestore([]string{fmt.Sprint(t.Name(), i)}, core.State{}))
		}
		assert.Len(t, reports(s), MaxReports)
	})
}
