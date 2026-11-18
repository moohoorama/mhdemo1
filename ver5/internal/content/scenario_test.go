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

func TestAlternateChapterTellsTheSameNumber(t *testing.T) {
	d := &Data{
		Nodes: []Node{{ID: "a", Kind: "battle", Stage: "B1"}, {ID: "b", Kind: "battle", Stage: "B2"}, {ID: "c"},
			{ID: "x"}, {ID: "y", Kind: "battle", Stage: "B2V"}},
		Chapters: []Chapter{
			{ID: "01", Nodes: []string{"a"}},
			{ID: "02", Nodes: []string{"b", "c"}},
			{ID: "02v", Nodes: []string{"x", "y"}},
		},
		Stages: []Stage{{ID: "B1", Order: 1}, {ID: "B2", Order: 2}, {ID: "B2V", Order: 2}},
	}
	t.Run("the number is the ID's leading digits", func(t *testing.T) {
		assert.Equal(t, 2, Chapter{ID: "02v"}.Number())
		assert.Equal(t, 0, Chapter{ID: "trial"}.Number())
		assert.Equal(t, 2, d.ChapterNumber("x"))
		assert.Equal(t, 0, d.ChapterNumber("missing"))
	})
	t.Run("the other telling opens after the battles before the chapter it retells", func(t *testing.T) {
		assert.Equal(t, 1, d.LastBattleOrder("x"))
		assert.Equal(t, 2, d.LastBattleOrder("c"))
	})
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

func TestParseCond(t *testing.T) {
	for in, want := range map[string]Cond{
		"flag:oath":      {Op: "flag", Arg: "oath", N: 1},
		"!flag:oath>=3":  {Not: true, Op: "flag", Arg: "oath", N: 3},
		"alive:liubei":   {Op: "alive", Arg: "liubei", N: 1},
		"dead:x":         {Op: "dead", Arg: "x", N: 1},
		"party:x":        {Op: "party", Arg: "x", N: 1},
		"won:B01":        {Op: "won", Arg: "B01", N: 1},
		"choice:pick=결의": {Op: "choice", Arg: "pick", Arg2: "결의", N: 1},
	} {
		got, err := ParseCond(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "flag", "flag:", "unknown:x", "flag:x>=y", "flag:>=2", "choice:pick", "choice:=a"} {
		_, err := ParseCond(bad)
		assert.Error(t, err, bad)
	}
}

func TestLoadRejectsBadStoryNodes(t *testing.T) {
	for name, nodes := range map[string]string{
		"unknown speaker":   `{"ID":"a","Kind":"say","Speaker":"nobody","Text":"x"}`,
		"say without text":  `{"ID":"a","Kind":"say","Speaker":"hero"}`,
		"narrate speaker":   `{"ID":"a","Kind":"narrate","Speaker":"hero","Text":"x"}`,
		"scene no place":    `{"ID":"a","Kind":"scene"}`,
		"set no flags":      `{"ID":"a","Kind":"set"}`,
		"branch no cases":   `{"ID":"a","Kind":"branch"}`,
		"branch bad cond":   `{"ID":"a","Kind":"branch","Cases":[{"If":["flag"],"Next":"a"}]}`,
		"branch bad target": `{"ID":"a","Kind":"branch","Cases":[{"If":["flag:x"],"Next":"zz"}]}`,
		"branch ref":        `{"ID":"a","Kind":"branch","Cases":[{"If":["alive:nobody"],"Next":"a"}]}`,
		"unknown kind":      `{"ID":"a","Kind":"dialogue","Text":"x"}`,
		"loop without wait": `{"ID":"a","Kind":"set","Flags":{"x":1},"Next":"b"},{"ID":"b","Kind":"scene","Place":"p","Next":"a"}`,
		"branch loop":       `{"ID":"a","Kind":"branch","Cases":[{"If":["flag:x"],"Next":"a"}],"Else":"a"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "mini")
			copyTree(t, "../testdata/mini", dir)
			story := `{"Nodes":[` + nodes + `,{"ID":"intro","Kind":"narrate","Text":"x"}]}`
			scenario := filepath.Join(dir, "scenario", "mini")
			require.NoError(t, os.WriteFile(filepath.Join(scenario, "story01.json"), []byte(story), 0644))
			p := filepath.Join(scenario, "scenario.json")
			b, err := os.ReadFile(p)
			require.NoError(t, err)
			start := strings.Replace(string(b), `"Start": "intro"`, `"Start": "a"`, 1)
			require.NoError(t, os.WriteFile(p, []byte(start), 0644))
			_, err = Load(dir)
			assert.Error(t, err)
		})
	}
}

func TestLoadAcceptsEveryStoryNodeKind(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mini")
	copyTree(t, "../testdata/mini", dir)
	story := `{"Nodes":[
 {"ID":"a","Kind":"scene","Place":"p"},
 {"ID":"b","Kind":"say","Speaker":"hero·archer1","Text":"x"},
 {"ID":"c","Kind":"narrate","Text":"x"},
 {"ID":"d","Kind":"set","Flags":{"f":1}},
 {"ID":"e","Kind":"branch","Cases":[{"If":["flag:f","alive:hero"],"Next":"g"}],"Else":"g"},
 {"ID":"g","Kind":"choice","Text":"x","Choices":{"go":"h"}},
 {"ID":"h","Kind":"narrate","Text":"x"},
 {"ID":"i","Kind":"branch","Cases":[{"If":["choice:g=go","!won:S1"],"Next":"j"}]},
 {"ID":"j","Kind":"end"}]}`
	scenario := filepath.Join(dir, "scenario", "mini")
	require.NoError(t, os.WriteFile(filepath.Join(scenario, "story01.json"), []byte(story), 0644))
	p := filepath.Join(scenario, "scenario.json")
	b, err := os.ReadFile(p)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, []byte(strings.Replace(string(b), `"Start": "intro"`, `"Start": "a"`, 1)), 0644))
	_, err = Load(dir)
	require.NoError(t, err)
}
