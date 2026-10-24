package gui

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/session"
)

// TestTownReachable walks every area of the town by its tile map from each of its doors (and the start): every
// door tile is open, and every door, sign and person who talks can be reached by A*.
func TestTownReachable(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	require.NoError(t, err)
	defer g.Close()
	require.NoError(t, g.OpenTown(filepath.Join("..", "..", "assets", "towns", "zhuo.json")))
	w := g.town
	for _, id := range sorted(w.t.Areas) {
		a := w.t.Areas[id]
		t.Run(id, func(t *testing.T) {
			starts := []tile{}
			for _, dr := range a.Doors {
				if dr.To != "" {
					starts = append(starts, tileAt(dr.At[0], dr.At[1]))
				}
			}
			if id == w.t.Start.Area {
				starts = append(starts, tileAt(w.t.Start.At[0], w.t.Start.At[1]))
			}
			at := starts[0]
			g.enterArea(id, []float64{float64(at.u), float64(at.v)}, "SE")
			m := g.townTiles()
			for _, dr := range a.Doors {
				assert.True(t, m.free(tileAt(dr.At[0], dr.At[1])), "door %q tile %v is not walkable", dr.Name, dr.At)
			}
			for _, from := range starts {
				for _, dr := range a.Doors {
					assert.NotNil(t, m.path(from, []tile{tileAt(dr.At[0], dr.At[1])}), "from %v: door %q unreachable", from, dr.Name)
				}
				for _, sg := range a.Signs {
					assert.NotNil(t, m.path(from, tileAt(sg.At[0], sg.At[1]).around()), "from %v: sign %q unreachable", from, sg.Name)
				}
				for _, key := range sorted(a.People) {
					if p := a.People[key]; len(p.Talk) > 0 {
						assert.NotNil(t, m.path(from, tileAt(p.At[0], p.At[1]).around()), "from %v: %s unreachable", from, key)
					}
				}
			}
		})
	}
}

// TestTilePath steps along u or v only and goes round what blocks the way.
func TestTilePath(t *testing.T) {
	m := &tileMap{w: 5, h: 3, open: []bool{
		true, true, true, true, true,
		true, false, false, false, true,
		true, true, true, true, true,
	}}
	route := m.path(tile{0, 1}, []tile{{4, 1}})
	require.Len(t, route, 7)
	for i := 1; i < len(route); i++ {
		assert.Equal(t, 1, manhattan(route[i-1], route[i]), "step %d", i)
	}
	assert.Nil(t, m.path(tile{0, 1}, []tile{{2, 1}}), "a blocked goal")
	to, ok := m.nearestReachable(tile{0, 0}, tile{2, 1})
	require.True(t, ok)
	assert.Equal(t, 1, manhattan(to, tile{2, 1}))
}
