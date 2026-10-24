package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// Story scenes come from the GUI's scene data (assets/scenes/*.json and the scene maps under
// assets/graphics/maps): each stretch of dialogue nodes stands on a small map, where its cast stands and
// what they do before each line. The CLI draws the same thing in letters.

type sceneMap struct {
	Name  string
	Size  [2]int
	Tiles []string
	Props [][]json.RawMessage
}

type sceneBook struct {
	book *scenes.Book
	maps map[string]sceneMap
}

// loadScenes reads the scene data next to the content directory; nil when there is none.
func loadScenes(contentDir string) *sceneBook {
	root := filepath.Dir(filepath.Clean(contentDir))
	book, err := scenes.Load(filepath.Join(root, "scenes"))
	if err != nil {
		return nil
	}
	b := &sceneBook{book: book, maps: map[string]sceneMap{}}
	for _, def := range book.Scenes {
		if _, ok := b.maps[def.Map]; ok {
			continue
		}
		var m sceneMap
		raw, err := os.ReadFile(filepath.Join(root, "graphics", "maps", def.Map+".json"))
		if err != nil {
			continue
		}
		if sm, ok, err := scenemap.Parse(raw); ok && err == nil {
			b.maps[def.Map] = sceneMapOf(sm)
		} else if json.Unmarshal(raw, &m) == nil {
			b.maps[def.Map] = m
		}
	}
	return b
}

// sceneMapOf reads a format 2 scene map the way the text map shows one: a glyph per game cell, objects by game cell.
func sceneMapOf(m *scenemap.Map) sceneMap {
	out := sceneMap{Name: m.Name, Size: m.Size}
	for v := range m.Size[1] {
		row := make([]byte, m.Size[0])
		for u := range row {
			row[u] = m.Glyph(2*u, 2*v)
		}
		out.Tiles = append(out.Tiles, string(row))
	}
	for _, o := range m.Objects {
		u := (float64(o.U)-.5)/2 + float64(o.DX)/64 + float64(o.DY)/32
		v := (float64(o.V)-.5)/2 - float64(o.DX)/64 + float64(o.DY)/32
		p, _ := json.Marshal([]any{o.Name, u, v})
		var raw []json.RawMessage
		_ = json.Unmarshal(p, &raw)
		out.Props = append(out.Props, raw)
	}
	return out
}

var sceneGlyph = map[byte]string{'.': "   ", 'd': " . ", '~': "~~~", 'c': "###", 'h': "^^^", 'i': " _ ", 'v': "VIL", 'f': "TRE", 's': " : ", 'T': " = ", 'W': " = ", 'P': " = ", 'C': " = "}

// stage is what the CLI remembers of the running scene: which scene and where its cast stands.
type stage struct {
	scene string
	pos   map[string][2]float64
}

// sceneFor updates st for the dialogue node and writes what changed: the place and map when the
// scene changes, then what the cast does before the node's lines.
func (b *sceneBook) sceneFor(d *content.Data, st *stage, node, place string, out io.Writer) {
	if b == nil {
		return
	}
	def, _ := b.book.For(node)
	if def == nil {
		return
	}
	fresh := st.scene != def.ID
	if fresh {
		st.scene, st.pos = def.ID, map[string][2]float64{}
		for name, a := range def.Cast {
			if len(a.At) == 2 && !a.Off {
				st.pos[name] = [2]float64{a.At[0], a.At[1]}
			}
		}
	}
	changed := fresh
	var notes []string
	for _, line := range def.Beats[node] {
		for _, group := range line {
			for _, s := range group {
				who := s.Actor
				switch {
				case len(s.Enter) == 2 || len(s.Move) == 2 || s.Exit:
					to := s.Move
					if len(to) != 2 {
						to = s.Enter
					}
					if len(to) == 2 {
						st.pos[who] = [2]float64{to[0], to[1]}
						notes = append(notes, fmt.Sprintf("%s → (%.1f,%.1f)", who, to[0], to[1]))
					}
					if s.Exit {
						delete(st.pos, who)
						notes = append(notes, who+" 퇴장")
					}
					changed = true
				case s.Say != "":
					notes = append(notes, fmt.Sprintf("%s: \"%s\"", who, s.Say))
				case s.Play != "":
					notes = append(notes, who+": "+s.Play)
				case s.Emote != "":
					notes = append(notes, who+": "+s.Emote)
				case s.Effect != "":
					notes = append(notes, "효과 "+s.Effect)
				case s.Intro != "":
					notes = append(notes, "첫 등장: "+s.Intro)
				}
			}
		}
	}
	if !changed && len(notes) == 0 {
		return
	}
	m := b.maps[def.Map]
	if fresh {
		fmt.Fprintf(out, "\n── 장면: %s (%s) ──\n", place, m.Name)
	}
	for _, n := range notes {
		fmt.Fprintln(out, "  ·", n)
	}
	if changed {
		b.draw(d, st, m, out)
	}
}

func (b *sceneBook) draw(d *content.Data, st *stage, m sceneMap, out io.Writer) {
	w, h := m.Size[0], m.Size[1]
	if w == 0 || h == 0 {
		return
	}
	props := map[[2]int]string{}
	for _, p := range m.Props {
		var name string
		var u, v float64
		if len(p) < 3 || json.Unmarshal(p[0], &name) != nil || json.Unmarshal(p[1], &u) != nil || json.Unmarshal(p[2], &v) != nil {
			continue
		}
		props[[2]int{int(math.Round(u)), int(math.Round(v))}] = strings.ToUpper((name + "   ")[:3])
	}
	ids := map[string]string{}
	for id, c := range d.Characters {
		if !c.Template {
			ids[c.Name] = id
		}
	}
	cast := map[[2]int]string{}
	var names []string
	for name := range st.pos {
		names = append(names, name)
	}
	sort.Strings(names)
	var legend []string
	for _, name := range names {
		p := st.pos[name]
		g := "Sol"
		if id, ok := ids[scenes.Base(name)]; ok {
			g = strings.ToUpper(id[:1]) + (id + "   ")[1:3]
		}
		cast[[2]int{int(math.Round(p[0])), int(math.Round(p[1]))}] = g
		legend = append(legend, fmt.Sprintf("%s=%s(%.1f,%.1f)", g, name, p[0], p[1]))
	}
	grid(out, w, h, func(x, y int) string {
		if g, ok := cast[[2]int{x, y}]; ok {
			return g + " "
		}
		if g, ok := props[[2]int{x, y}]; ok {
			return g + " "
		}
		if y < len(m.Tiles) && x < len(m.Tiles[y]) {
			if g, ok := sceneGlyph[m.Tiles[y][x]]; ok {
				return g + " "
			}
		}
		return "    "
	})
	fmt.Fprintln(out, "  인물:", strings.Join(legend, " "))
}
