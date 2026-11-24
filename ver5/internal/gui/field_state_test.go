package gui

import (
	"bytes"
	"fmt"
	"image"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/mapedit"
)

func TestLooksAreTheCellsWithAStateThatShows(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	st := content.Stage{Width: 5, Height: 1, Tiles: []string{"g-eg-Fg-sg--wbd"}}
	looks := Looks(d, &st)
	want := map[image.Point]string{{X: 0, Y: 0}: "ember", {X: 1, Y: 0}: "flame", {X: 2, Y: 0}: "soot", {X: 4, Y: 0}: "damaged"}
	if len(looks) != len(want) {
		t.Fatalf("looks %v", looks)
	}
	for p, l := range want {
		if looks[p] != l {
			t.Errorf("%v: %q, want %q", p, looks[p], l)
		}
	}
	if Looks(d, nil) != nil {
		t.Error("a missing map has no looks")
	}
}

func TestMapFXDemoPaintsAValidMapAndTheArtIsLoaded(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		t.Fatal(err)
	}
	a := d.g.assets
	for _, name := range []string{"wall_clean_00", "wall_clean_15", "gatehouse_closed_u_0", "gatehouse_closed_v_3", "fence_00", "fence_15", "ash_2_0", "burnt_village_0", "burnt_village_3", "burnt_camp", "burnt_forest_3", "bridge_u_damaged", "bridge_v_broken"} {
		if _, ok := a.Props[name]; !ok {
			t.Errorf("no picture %q", name)
		}
	}
	for _, name := range []string{"flame_t", "flame_t_small", "smoke_t"} {
		if s := a.Anims[name]; len(s.Frames) == 0 {
			t.Errorf("no flame strip %q", name)
		}
	}
	looks := LooksOf(d.g.S.Data, d.g.S.Engine.Observe())
	seen := map[string]bool{}
	for _, l := range looks {
		seen[l] = true
	}
	for _, l := range []string{"ember", "flame", "soot", "damaged", "broken"} {
		if !seen[l] {
			t.Errorf("the demo shows no %s cell", l)
		}
	}
}

// validCells lists every cell the data allows: a ground, a structure that may stand on it (or none), and a state that
// may lie over both (or none).
func validCells(d *content.Data) []string {
	var cells []string
	for _, g := range sortedKeys(d.Grounds) {
		structs := []string{"-"}
		for _, k := range sortedKeys(d.Structures) {
			if strings.Contains(d.Structures[k].Grounds, g) {
				structs = append(structs, k)
			}
		}
		for _, st := range structs {
			states := []string{"-"}
			for _, k := range sortedKeys(d.States) {
				sv := d.States[k]
				if (sv.On == "" || strings.Contains(sv.On, st)) && (sv.Grounds == "" || strings.Contains(sv.Grounds, g)) && (sv.On == "" || st != "-") {
					states = append(states, k)
				}
			}
			for _, k := range states {
				cells = append(cells, g+st+k)
			}
		}
	}
	return cells
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestFieldDrawsTheSameEveryFrame draws maps that hold every cell the data allows, in the cells' order and in shuffled
// orders so the neighbours change, and every state mark beside every other (a block of one cell ringed by the other).
// Each map is drawn again and again at one clock and camera: the pictures overlap their neighbours, so any order
// that changes from draw to draw (ranging over a map) shows as flicker on the edges.
func TestFieldDrawsTheSameEveryFrame(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for the native renderer")
	}
	dm, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := dm.g
	data := g.S.Data
	all := validCells(data)
	require.NotEmpty(t, all)

	// the state marks, each on grass, ringing and ringed by each other
	var marks []string
	for _, k := range sortedKeys(data.States) {
		if data.States[k].On == "" && (data.States[k].Grounds == "" || strings.Contains(data.States[k].Grounds, "g")) {
			marks = append(marks, "g-"+k)
		}
	}
	var pairRows []string
	const bw = 3
	blocks := len(marks) * len(marks)
	perRow := 8
	for b := 0; b < blocks; b += perRow {
		for dy := 0; dy < bw; dy++ {
			row := ""
			for k := b; k < min(b+perRow, blocks); k++ {
				for dx := 0; dx < bw; dx++ {
					if dx == 1 && dy == 1 {
						row += marks[k/len(marks)]
					} else {
						row += marks[k%len(marks)]
					}
				}
			}
			pairRows = append(pairRows, row+strings.Repeat("g--", perRow*bw-len(row)/3))
		}
	}
	maps := map[string]content.Stage{}
	tile := func(cells []string, width int) content.Stage {
		var rows []string
		for i := 0; i < len(cells); i += width {
			row := ""
			for k := i; k < i+width; k++ {
				row += cells[k%len(cells)]
			}
			rows = append(rows, row)
		}
		return content.Stage{Width: width, Height: len(rows), Tiles: rows}
	}
	const width = 16
	maps["every cell in order"] = tile(all, width)
	for seed := int64(1); seed <= 8; seed++ {
		shuffled := append([]string(nil), all...)
		rand.New(rand.NewSource(seed)).Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		maps[fmt.Sprintf("shuffled %d", seed)] = tile(shuffled, width)
	}
	maps["state marks ringed by each other"] = content.Stage{Width: perRow * bw, Height: len(pairRows), Tiles: pairRows}

	p := &onceRenderTest{run: func() error {
		for name, st := range maps {
			doc := mapedit.Import(st)
			art := artFromDocument(doc, data)
			f := NewField(g.assets, art)
			f.SetLooks(Looks(data, &st))
			mx, my := f.Center(float64(st.Width-1)/2, float64(st.Height-1)/2)
			const vw, vh, z = 1280, 850, 1.0
			hw, hh := vw/2/z, vh/2/z
			var first []byte
			for i := 0; i < 12; i++ {
				f.prepareRender(z, mx-hw, my-hh, vw, vh)
				f.DrawBounds = image.Rect(int(mx-hw)-64, int(my-hh)-96, int(mx+hw)+64, int(my+hh)+96)
				f.Draw(1234, nil, nil)
				img := image.NewRGBA(f.Canvas.Bounds())
				f.Canvas.ReadPixels(img.Pix)
				if first == nil {
					first = img.Pix
				} else if !bytes.Equal(first, img.Pix) {
					f.dispose()
					return fmt.Errorf("%s: draw %d differs from draw 0 at the same clock and camera", name, i)
				}
			}
			f.dispose()
		}
		return nil
	}}
	result := make(chan error, 1)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
}
