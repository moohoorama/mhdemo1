package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAfterFollowsNextThenChapterOrder(t *testing.T) {
	d := &Data{
		Nodes: []Node{{ID: "a", Next: "c"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}},
		Chapters: []Chapter{
			{ID: "1", Nodes: []string{"a", "b", "c"}},
			{ID: "2", Nodes: []string{"d", "e"}},
		},
	}
	for from, want := range map[string]string{"a": "c", "b": "c", "c": "d", "d": "e", "e": "", "missing": ""} {
		assert.Equal(t, want, d.After(from), from)
	}
}

func copyTree(t *testing.T, from, to string) {
	require.NoError(t, filepath.WalkDir(from, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		if e.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), b, 0644)
	}))
}

func TestLoadRejectsBadChapters(t *testing.T) {
	for name, edit := range map[string]func(string) string{
		"duplicate chapter": func(s string) string {
			return strings.Replace(s, `"Chapters": [`, `"Chapters": [{"ID": "01", "File": "story01.json"},`, 1)
		},
		"missing file": func(s string) string { return strings.Replace(s, `"story01.json"`, `"nope.json"`, 1) },
		"unknown start": func(s string) string {
			return strings.Replace(s, `"Start": "intro"`, `"Start": "nowhere"`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "mini")
			copyTree(t, "../testdata/mini", dir)
			p := filepath.Join(dir, "scenario", "mini", "scenario.json")
			b, err := os.ReadFile(p)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(p, []byte(edit(string(b))), 0644))
			_, err = Load(dir)
			assert.Error(t, err)
		})
	}
}

func TestLoadFindsScenarioBesideOrInsideData(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	assert.Equal(t, "hero", d.ScenarioID)
	assert.NotEmpty(t, d.Chapters)
	m, err := Load("../testdata/mini")
	require.NoError(t, err)
	assert.Equal(t, "mini", m.ScenarioID)
	assert.NotEqual(t, d.Hash, m.Hash)
}

func TestStageFileNames(t *testing.T) {
	assert.Equal(t, "stage01.json", StageFileName("B01"))
	assert.Equal(t, "map12.json", MapFileName("B12"))
	assert.Equal(t, "stageS1.json"[:5]+"1.json", StageFileName("S1"))
}
