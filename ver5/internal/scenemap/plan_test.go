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
	t.Run("the gate's slices take the wall blocks' places, one per small tile", func(t *testing.T) {
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		at := map[string][2]int{}
		blocks := map[[2]int]bool{}
		for _, s := range p.Solid {
			if _, ok := at[s.Name]; !ok {
				at[s.Name] = s.At
			}
			if s.Name == "swall2" {
				blocks[s.At] = true
			}
		}
		assert.Equal(t, [2]int{17, 4}, at["sgate2_open_v_0"])
		assert.Equal(t, [2]int{17, 9}, at["sgate2_open_v_5"])
		assert.Equal(t, [2]int{17, 13}, at["sgate2_open_v_9"])
		assert.True(t, blocks[[2]int{17, 2}])
		assert.False(t, blocks[[2]int{17, 9}])
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
		require.Len(t, got, 4)
		assert.Equal(t, Draw{Kind: DrawSprite, Name: "jshade_72_0", X: 256, Y: 162 - 72}, got[0], "as high as the wall block")
		assert.Empty(t, shadows(1), "turned a quarter, the wall runs along u: no face along v throws one")
	})
	t.Run("every turn builds", func(t *testing.T) {
		for turn := range 4 {
			_, err := Build(m, c, Options{Turn: turn, Decor: true})
			assert.NoError(t, err)
		}
	})
}

func TestParapets(t *testing.T) {
	m, c := load(t, "S_castle_town")
	draws := func(turn int) map[[2]int][]string {
		p, err := Build(m, c, Options{Turn: turn})
		require.NoError(t, err)
		out := map[[2]int][]string{}
		for _, s := range p.Solid {
			if s.Name == "swall2" {
				for _, d := range s.Draws {
					out[s.At] = append(out[s.At], d.Name)
				}
			}
		}
		return out
	}
	t.Run("on the sides no wall tile touches, behind first, over the shadow out of the open front sides", func(t *testing.T) {
		got := draws(0)
		assert.Equal(t, []string{"swall2", "sparapet_mu", "sparapet_mv"}, got[[2]int{-1, -1}], "the outer corner")
		assert.Equal(t, []string{"swall2_shade_pv", "swall2", "sparapet_mv", "sparapet_pv"}, got[[2]int{5, -1}], "along u")
		assert.Equal(t, []string{"swall2_shade_pu", "swall2", "sparapet_mu", "sparapet_pu"}, got[[2]int{-1, 20}], "along v")
	})
	t.Run("a line running into the board's edge stops there, open: no end parapet, no shadow past it", func(t *testing.T) {
		got := draws(0)
		assert.NotContains(t, got, [2]int{32, -1}, "past the grid's edge (u 32)")
		assert.Equal(t, []string{"swall2_shade_pv", "swall2", "sparapet_mv", "sparapet_pv"}, got[[2]int{31, -1}], "the last tile on the board")
	})
	t.Run("sides are taken in the view", func(t *testing.T) {
		assert.Equal(t, []string{"swall2_shade_pu", "swall2", "sparapet_mu", "sparapet_pu"}, draws(1)[[2]int{5, -1}], "turned a quarter, the u line runs along v")
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

func TestWalkable(t *testing.T) {
	m, c := load(t, "S_checkpoint")
	w := Walkable(m, c)
	for _, tc := range []struct {
		name string
		u, v int
		open bool
	}{
		{"the road east of the wall", 20, 9, true},
		{"the far cliff", 20, 0, false},
		{"the near ridge", 20, 32, false},
		{"the wall line", 17, 3, false},
		{"the gate's passage", 17, 9, true},
		{"the gate's wing", 17, 12, false},
		{"under the watchtower", 14, 4, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.open, w.Open(tc.u, tc.v))
		})
	}
}

// Within the walls of Zhuo the hero reaches the open gate but not the grass outside it.
func TestWalkWithin(t *testing.T) {
	m, c := load(t, "S_zhuo2")
	w := Walkable(m, c)
	require.True(t, w.Open(-3, 11), "the gate leads out before Within")
	w.Within(7, 11)
	for _, tc := range []struct {
		name string
		u, v int
		open bool
	}{
		{"the street", 7, 11, true},
		{"the gate's passage", -1, 11, true},
		{"the grass outside the gate", -2, 11, false},
		{"the grass outside the north wall", 10, -5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.open, w.Open(tc.u, tc.v))
		})
	}
}

// A light-giving object carries its flames and light wherever it stands: on the raised cliff top they rise with it.
func TestAttachments(t *testing.T) {
	_, c := load(t, "S_cliff_camp")
	for _, h := range []int{0, 2} {
		height := []string{strings.Repeat(string(rune('0'+h)), 4)}
		m := &Map{ID: "t", Format: 2, Ground: []string{"...."}, Height: height, Objects: []Object{{Name: "campfire", U: 1}}}
		p, err := Build(m, c, Options{})
		require.NoError(t, err)
		var fire Piece
		for _, piece := range p.Pieces() {
			if piece.Name == "campfire" {
				fire = piece
			}
		}
		lift := float64(c.Rules.Heights["2"].Lift) * float64(h/2)
		x, y := 16*(1.5-.5), 8*(1.5+.5)-lift
		assert.Equal(t, [2]float64{x, y}, fire.Anchor)
		require.Len(t, fire.Draws, 3, "the logs and two flames")
		flame := c.Anims["flame_small"]
		a := c.Rules.Attachments["campfire"][0]
		assert.Equal(t, Draw{Kind: DrawAnim, Name: "flame_small", X: round(x + float64(a.At[0]-flame.Pivot[0])), Y: round(y + float64(a.At[1]-flame.Pivot[1]))}, fire.Draws[1])
		require.Len(t, fire.Lights, 1)
		assert.Equal(t, y-6, fire.Lights[0].Y)
		assert.True(t, fire.Lights[0].Glow)
	}
}

// A raised level's tops are drawn over everything standing on lower ground, so nothing standing there may reach up
// into a top that lies behind it: the top would cover what stands in front of it.
func TestLevelsCoverOnlyWhatStandsBehind(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(gfx, "maps", "S_*.json"))
	require.NoError(t, err)
	c, err := LoadCatalog(gfx)
	require.NoError(t, err)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		m, ok, err := Parse(data)
		require.NoError(t, err)
		if !ok {
			continue
		}
		for turn := range 4 {
			p, err := Build(m, c, Options{Turn: turn, Decor: true})
			require.NoError(t, err)
			if len(p.Levels) == 0 {
				break
			}
			w := newView(m, c, turn)
			for v := w.v0; v < w.v1; v++ {
				for u := w.u0; u < w.u1; u++ {
					lift, _, _, err := w.height(w.step(u, v))
					require.NoError(t, err)
					if lift == 0 {
						continue
					}
					x, y := center(u, v)
					top := Key{Box: [4]float64{float64(u), float64(v), float64(u + 1), float64(v + 1)}, Lift: lift,
						Rect: image.Rect(x-16, y-8-lift, x+16, y+8-lift)}
					for _, q := range p.Pieces() {
						if q.Lift < lift && q.Rank != RankFace && q.Rect.Overlaps(top.Rect) && top.behind(q.Key) {
							t.Errorf("%s turn %d: %s at %v reaches into the top of (%d, %d) behind it", m.ID, turn, q.Name, q.At, u, v)
						}
					}
				}
			}
		}
	}
}
