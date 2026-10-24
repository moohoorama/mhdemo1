package scenemap

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gfx = "../../assets/graphics"

func load(t *testing.T, id string) (*Map, *Catalog) {
	t.Helper()
	c, err := LoadCatalog(gfx)
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(gfx, "maps", id+".json"))
	require.NoError(t, err)
	m, ok, err := Parse(data)
	require.NoError(t, err)
	require.True(t, ok, "%s is not a format 2 map", id)
	return m, c
}

func TestView(t *testing.T) {
	m := &Map{Ground: []string{"...", "..."}, Origin: [2]int{-1, 0}}
	for turn := range 4 {
		t.Run("rot undoes orig", func(t *testing.T) {
			w := newView(m, nil, turn)
			for _, p := range [][2]int{{0, 0}, {-3, 5}, {7, -2}} {
				u, v := w.rot(p[0], p[1])
				ou, ov := w.orig(u, v)
				assert.Equal(t, p, [2]int{ou, ov})
				cu, cv := w.point(float64(p[0])+.5, float64(p[1])+.5)
				assert.Equal(t, [2]float64{float64(u) + .5, float64(v) + .5}, [2]float64{cu, cv}, "a tile's centre turns with it")
			}
			assert.Equal(t, 6, (w.u1-w.u0)*(w.v1-w.v0))
		})
	}
}

func TestOrder(t *testing.T) {
	box := func(u0, v0, u1, v1 float64) Key {
		k := Key{Box: [4]float64{u0, v0, u1, v1}, Rank: RankObject, X: 16 * (u0 - v0)}
		k.Rect = image.Rect(-1000, -1000, 1000, 1000) // everything overlaps on screen
		return k
	}
	t.Run("plain order by front corner, height, kind", func(t *testing.T) {
		person := box(2, 2, 3, 3)
		person.Rank = RankPerson
		keys := []Key{person, box(2, 2, 3, 3), box(0, 0, 1, 1)}
		assert.Equal(t, []int{2, 1, 0}, Order(keys))
	})
	t.Run("a wide object comes after what lies wholly behind it", func(t *testing.T) {
		hall := box(0, 0, 4, 2) // front corner 6
		behind := box(1, -2, 2, -1)
		right := box(5, -1, 6, 0) // front 6 too, but wholly in front along u
		order := Order([]Key{right, hall, behind})
		pos := map[int]int{}
		for p, i := range order {
			pos[i] = p
		}
		assert.Less(t, pos[2], pos[1], "behind before the hall")
		assert.Less(t, pos[1], pos[0], "the hall before what stands past its end")
	})
}

func TestCheckpoint(t *testing.T) {
	m, c := load(t, "S_checkpoint")
	t.Run("the gate takes the wall's place and its edge post inside the ridge is not drawn", func(t *testing.T) {
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		at := map[string][2]int{}
		for _, s := range p.Solid {
			if _, ok := at[s.Name]; !ok {
				at[s.Name] = s.At
			}
		}
		assert.Equal(t, [2]int{17, 9}, at["sgate_open_v_5"])
		assert.NotContains(t, at, "sgate_open_v_11")
		assert.Equal(t, [2]int{17, 2}, at["swall_v"])
	})
	t.Run("corner shadow on the far cliff right of the wall, only while that face shows", func(t *testing.T) {
		shadows := func(turn int) []Draw {
			p, err := Build(m, c, Options{Turn: turn})
			require.NoError(t, err)
			var out []Draw
			for _, s := range p.Solid {
				for _, d := range s.Draws {
					if strings.HasPrefix(d.Name, "jshade_") {
						out = append(out, d)
					}
				}
			}
			return out
		}
		got := shadows(0)
		require.Len(t, got, 5)
		assert.Equal(t, Draw{Kind: DrawSprite, Name: "jshade_80_0", X: 256, Y: 162 - 80}, got[0], "the old map's place")
		assert.Empty(t, shadows(1), "turned a quarter, the wall runs along u: no face along v throws one")
	})
	t.Run("every turn builds", func(t *testing.T) {
		for turn := range 4 {
			_, err := Build(m, c, Options{Turn: turn, Decor: true})
			assert.NoError(t, err)
		}
	})
}

func TestTentFloors(t *testing.T) {
	m, c := load(t, "S_tent_mid")
	p, err := Build(m, c, Options{})
	require.NoError(t, err)
	floors := 0
	for _, d := range p.Flat {
		assert.NotEqual(t, DrawTile, d.Kind, "a floored room shows no ground")
		if d.Name == "floor_tent" {
			floors++
		}
	}
	assert.Equal(t, 30, floors)
}
