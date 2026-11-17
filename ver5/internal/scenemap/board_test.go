package scenemap

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The map's grid is a board cut out of a larger land (docs/presentation-scene.md, 판 단면).
func TestBoard(t *testing.T) {
	board := [4]int{0, 0, 10, 8}
	for _, tc := range []struct {
		name   string
		box    [4]float64
		tileOn bool
		want   bool
	}{
		{"wholly on the board", [4]float64{2, 2, 3, 3}, true, true},
		{"across the +u edge", [4]float64{9.5, 2, 10.5, 3}, true, false},
		{"across the +v edge", [4]float64{2, 7.5, 3, 8.5}, true, false},
		{"touching the +u edge from inside", [4]float64{9, 2, 10, 3}, true, true},
		{"a crown over the back edge", [4]float64{-.5, 2, .5, 3}, true, true},
		{"wholly past the back edge, its tile off the board", [4]float64{-1, 2, 0, 3}, false, false},
		{"a room's back wall, just behind the floor, its tile on it", [4]float64{-.5, 0, 0, 2}, true, true},
		{"a corner post across a back and a front edge", [4]float64{9.5, -.5, 10.5, .5}, false, true},
		{"wholly past a front edge", [4]float64{10, 2, 11, 3}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, OnBoard(tc.box, board, tc.tileOn))
		})
	}
	t.Run("a section piece under every small tile of the front edges, by its ground", func(t *testing.T) {
		m, c := load(t, "S_zhuo")
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		count := map[string]int{}
		for _, d := range p.Flat {
			if strings.HasPrefix(d.Name, "board_end") {
				count["end"]++
			} else if strings.HasPrefix(d.Name, "board_") {
				count[d.Name[strings.LastIndex(d.Name[:strings.LastIndexByte(d.Name, '_')], "_")+1:strings.LastIndexByte(d.Name, '_')]]++
			}
		}
		assert.Equal(t, m.H(), count["pu"], "+u edge: one per small tile")
		assert.Equal(t, m.W(), count["pv"], "+v edge")
		assert.Equal(t, 2, count["end"], "the two side corners")
	})
	t.Run("a room shows no section", func(t *testing.T) {
		m, c := load(t, "S_tavern")
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		for _, d := range p.Flat {
			assert.False(t, strings.HasPrefix(d.Name, "board_"), d.Name)
		}
	})
	t.Run("a city wall top on a front edge stands its cut, by its place in the run", func(t *testing.T) {
		m, c := load(t, "S_baimen")
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		var places []string
		for _, piece := range p.Pieces() {
			if strings.HasPrefix(piece.Name, "rwall_cut_pu_") {
				places = append(places, strings.TrimPrefix(piece.Name, "rwall_cut_pu_"))
			}
		}
		require.NotEmpty(t, places)
		assert.Contains(t, places, "a")
		assert.Contains(t, places, "b")
		assert.Contains(t, places, "m")
	})
	t.Run("a city wall top running into a front edge takes no face, rim or foot there", func(t *testing.T) {
		m, c := load(t, "S_baimen")
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		u1 := m.Origin[0] + m.W() - 1
		for _, piece := range p.Pieces() {
			if piece.At[0] != u1 || m.Step(piece.At[0], piece.At[1]) != 3 {
				continue
			}
			assert.False(t, strings.HasPrefix(piece.Name, "rrim_se"), "%s at %v", piece.Name, piece.At)
			assert.NotContains(t, piece.Name, "_04_", "%s at %v: a face toward +u", piece.Name, piece.At)
			assert.NotContains(t, piece.Name, "_12_", "%s at %v: a face toward +u", piece.Name, piece.At)
		}
	})
	t.Run("a wall laid as objects runs on past the edge: its straight piece there, not its end", func(t *testing.T) {
		m, c := load(t, "S_xudu")
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		names := map[[2]int]string{}
		for _, piece := range p.Pieces() {
			if strings.HasPrefix(piece.Name, "pwall_u") {
				names[piece.At] = piece.Name
			}
		}
		assert.Equal(t, "pwall_u", names[[2]int{m.Origin[0] + m.W() - 1, 4}], "on the +u edge")
		assert.Equal(t, "pwall_u", names[[2]int{m.Origin[0], 4}], "on the -u edge")
	})
	t.Run("nothing stands across a front edge", func(t *testing.T) {
		m, c := load(t, "S_mangdang")
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		u1, v1 := float64(m.Origin[0]+m.W()), float64(m.Origin[1]+m.H())
		for _, piece := range p.Pieces() {
			if piece.Rank == RankObject {
				assert.LessOrEqual(t, piece.Box[2], u1, piece.Name)
				assert.LessOrEqual(t, piece.Box[3], v1, piece.Name)
			}
		}
	})
}
