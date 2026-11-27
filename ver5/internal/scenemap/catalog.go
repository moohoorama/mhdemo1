package scenemap

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Catalog is what the rules know of the pictures (assets/graphics/index.json) and of the scene rules
// (assets/graphics/scene.json).
type Catalog struct {
	Sprites map[string]Sprite
	// Structures are the battle map's block pictures (paving, walls, cliffs), one row of equal cells.
	Structures map[string]Sprite
	// Tiles are the ground tileset's pieces by id: a small tile is four of them.
	Tiles map[int]Sprite
	// Animations are the tileset's frame lists by name ("water", "grass_0", …).
	Animations map[string][]int
	// Anims are the looping strips (index.json "anims") by name, each with the rect and pivot of its frame 0.
	Anims   map[string]Sprite
	Rules   Rules
	Scenery *Scenery // nil when graphics/map/scenery.json is absent
}

// Sprite is a picture on one of the sheets; its pivot lands on the drawing point.
type Sprite struct {
	Sheet      string // "props", "extra" or "tiles"
	Rect       [4]int // x, y, w, h on the sheet
	Pivot      [2]int
	Ground     bool // lies flat on the ground (layer "ground")
	NoShadow   bool
	GateOffset *float64 // a gatehouse slice: game cells along its wall from the gate's point
}

// WallFamily is how a wall line is drawn: a block on each small tile of it, and on each side (_mu, _mv, _pu, _pv:
// toward -u, -v, +u, +v in the view) of a block no wall or gate tile touches, a parapet. Under a wall or gate tile
// lies its shadow on the ground in front, out of each open front side (_pu, _pv) and past the front corner when
// both are open (_pc). A family with Piece (a compound wall) draws instead one picture per small tile chosen by the
// sides another wall or gate tile closes (or the board's edge along the line): <piece>_<mask>, mask 1 -u, 2 -v,
// 4 +u, 8 +v in the view, "%02d".
type WallFamily struct {
	Block   string `json:"block"`
	Parapet string `json:"parapet"`
	Shade   string `json:"shade"`
	Piece   string `json:"piece"`
	// Cut is the family of the line's cut where it runs into a front edge of the board: <cut>_pu, <cut>_pv.
	Cut string `json:"cut"`
}

// Board is how the board's front edges are drawn (docs/presentation-scene.md, 판 단면): under each small tile of the
// +u and +v edges a piece of the ground's cross-section, <family>_pu_<k> or <family>_pv_<k> (k by the tile, of
// Variants), the family by the tile's ground glyph (Default for one not listed, Under below a raised tile); End_pu
// and End_pv close the two side corners. A board whose every tile is an Indoor floor (a room, a tent) shows no
// section: a room is no diorama, and walking out of it the cut under its floor looks odd.
type Board struct {
	Indoor   string            `json:"indoor"`
	Sections map[string]string `json:"sections"`
	Default  string            `json:"default"`
	Under    string            `json:"under"`
	End      string            `json:"end"`
	Variants int               `json:"variants"`
}

// Rules is scene.json.
type Rules struct {
	Surfaces map[string]int `json:"surfaces"`
	Walkable string         `json:"walkable"` // the ground glyphs a person can walk on
	// Floors are the glyphs drawn with a floor picture: by default one per game cell in place of the ground, over
	// its diamond filled with Edge; Tile floors (paving) one per small tile over the ground.
	Floors map[string]struct {
		Sprite string `json:"sprite"`
		Sheet  string `json:"sheet"`
		Tile   bool   `json:"tile"`
		Edge   string `json:"edge"`
	} `json:"floors"`
	Heights map[string]struct {
		Lift  int    `json:"lift"`
		Faces string `json:"faces"`
		Foot  string `json:"foot"`
		// Back is the family of the brink drawn on a top's edges that drop away from the camera (toward -v, -u),
		// whose faces the view does not show: <back>_ne_<k>, <back>_nw_<k>, one small tile of edge each.
		Back string `json:"back"`
		// Rim is the family of what stands on a top's edges over lower ground, among the people on the top (a wall
		// walk's parapets): <rim>_<side>_<k>, side sw (+v), se (+u), ne (-v) or nw (-u), one small tile of edge each.
		Rim string `json:"rim"`
		// Cut is the family of the cut where such a top runs into a front edge: <cut>_pu_<place>, <cut>_pv_<place>,
		// place a, m or b (the first, a middle or the last tile of the run along the edge) or ab (a run of one).
		Cut string `json:"cut"`
	} `json:"heights"`
	Walls       map[string]WallFamily `json:"walls"`
	Gates       map[string]string     `json:"gates"`        // a gate object -> its slices (<slices>_<axis>_<k>)
	GatePassage [2]int                `json:"gate_passage"` // the small tiles of a gate's passage, from its point
	// Layers are the layers a picture may come with, drawn over it, each its own picture <name>_<layer> taking its
	// own colour of a tint (Tint.Of): pictures drawn in greys to be tinted part by part.
	Layers       []string `json:"layers"`
	CornerShadow struct {
		Sprite string `json:"sprite"`
		Offset [2]int `json:"offset"`
	} `json:"corner_shadow"`
	Footprints map[string][4]float64 `json:"footprints"`
	// Decor is the scene maps' own ground decoration over the scenery's (map/scenery.json): per surface the expected
	// stamps per small tile of each group (Scatter, replacing the scenery's rate for that group), and groups of their
	// own (Groups, "<surface>.<group>"); a name "tile:<animation>" is the tileset animation's first frame.
	Decor struct {
		Scatter map[string]map[string]float64 `json:"scatter"`
		Groups  map[string][]string           `json:"groups"`
	} `json:"decor"`
	Board      Board                 `json:"board"`
	// Cuts are the cuts of walls laid as objects, one small tile each (<name>_u along u, <name>_v along v), where
	// the run reaches a front edge: object name -> <cut>_pu or <cut>_pv.
	Cuts map[string]string `json:"cuts"`
	// Attachments are what an object carries wherever it stands: the flames of a campfire, the light of a lantern.
	Attachments map[string][]Attachment `json:"attachments"`
	Flood       Flood                   `json:"flood"`
}

// Flood is ground under flood water: its glyphs draw the ground Surfaces gives them with a sheet of muddy water over
// each small tile, a rim of foam where it meets dry ground. An object standing in it sinks Sink pixels (SinkDefault
// for one not listed) and is cut off at the waterline along the front of its footprint; a thin one listed in Ring
// (a tree's trunk, a post) is cut off round its anchor instead, that many pixels to each side.
type Flood struct {
	Glyphs      string         `json:"glyphs"`
	Color       string         `json:"color"` // the water, "#rrggbb"
	Alpha       float64        `json:"alpha"` // how much of the ground it hides
	Foam        string         `json:"foam"`  // the rim and the waterline round what stands in it
	Sink        map[string]int `json:"sink"`
	SinkDefault int            `json:"sink_default"`
	Ring        map[string]int `json:"ring"`
	// Kinds give a glyph its own water in place of Color and Alpha: a clear shallow river over sand (a ford)
	// instead of muddy flood. Such a tile counts open water ('~') beside it as wet, so no foam rims it there.
	Kinds map[string]FloodKind `json:"kinds"`
}

// FloodKind is the water of a Flood glyph that differs from the muddy default.
type FloodKind struct {
	Color string  `json:"color"`
	Alpha float64 `json:"alpha"`
}

// flooded reports whether glyph g is flooded ground.
func (c *Catalog) flooded(g byte) bool {
	return g != 0 && strings.IndexByte(c.Rules.Flood.Glyphs, g) >= 0
}

// Attachment is a looping strip drawn over an object (a flame), the light it casts at night, or both, at At canvas
// pixels from the object's anchor.
type Attachment struct {
	Anim  string  `json:"anim,omitempty"`
	At    [2]int  `json:"at"`
	Light float64 `json:"light,omitempty"` // the light's radius, world pixels at zoom 1
	Glow  bool    `json:"glow,omitempty"`  // a fire's warm glow over its light
}

// Light is a light an object casts at night, at a canvas point.
type Light struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Radius float64 `json:"r"`
	Glow   bool    `json:"glow,omitempty"`
}

// LoadCatalog reads the catalog from a graphics directory (assets/graphics).
func LoadCatalog(dir string) (*Catalog, error) {
	index, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, err
	}
	rules, err := os.ReadFile(filepath.Join(dir, "scene.json"))
	if err != nil {
		return nil, err
	}
	scenery, err := os.ReadFile(filepath.Join(dir, "map", "scenery.json"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return NewCatalog(index, rules, scenery)
}

// NewCatalog builds a catalog from the contents of index.json, scene.json and map/scenery.json (nil: none).
func NewCatalog(index, rules, scenery []byte) (*Catalog, error) {
	type sheet struct {
		Sprites map[string]struct {
			Rect       [4]int   `json:"rect"`
			Pivot      [2]int   `json:"pivot"`
			Layer      string   `json:"layer"`
			Shadow     *bool    `json:"shadow"`
			GateOffset *float64 `json:"gate_offset"`
		} `json:"sprites"`
	}
	var ix struct {
		Tileset struct {
			Sprites    map[string][6]int `json:"sprites"`
			Animations map[string]struct {
				Frames []int `json:"frames"`
			} `json:"animations"`
		} `json:"tileset"`
		Props sheet `json:"props"`
		Extra sheet `json:"extra"`
		Anims map[string]struct {
			Rect  [4]int `json:"rect"`
			Pivot [2]int `json:"pivot"`
		} `json:"anims"`
		Structures struct {
			Cell    [2]int         `json:"cell"`
			Pivot   [2]int         `json:"pivot"`
			Sprites map[string]int `json:"sprites"`
		} `json:"structures"`
	}
	if err := json.Unmarshal(index, &ix); err != nil {
		return nil, fmt.Errorf("index.json: %w", err)
	}
	c := &Catalog{Sprites: map[string]Sprite{}, Structures: map[string]Sprite{}, Tiles: map[int]Sprite{}, Animations: map[string][]int{},
		Anims: map[string]Sprite{}}
	st := ix.Structures
	for name, i := range st.Sprites {
		c.Structures[name] = Sprite{Sheet: "structures", Rect: [4]int{i * st.Cell[0], 0, st.Cell[0], st.Cell[1]}, Pivot: st.Pivot}
	}
	if err := json.Unmarshal(rules, &c.Rules); err != nil {
		return nil, fmt.Errorf("scene.json: %w", err)
	}
	for id, r := range ix.Tileset.Sprites {
		n, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("index.json: tile %q: %w", id, err)
		}
		c.Tiles[n] = Sprite{Sheet: "tiles", Rect: [4]int{r[0], r[1], r[2], r[3]}, Pivot: [2]int{r[4], r[5]}, Ground: true}
	}
	for name, a := range ix.Tileset.Animations {
		c.Animations[name] = a.Frames
	}
	for name, a := range ix.Anims {
		c.Anims[name] = Sprite{Sheet: "anims", Rect: a.Rect, Pivot: a.Pivot}
	}
	for _, s := range []struct {
		name string
		sheet
	}{{"props", ix.Props}, {"extra", ix.Extra}} {
		for name, p := range s.Sprites {
			c.Sprites[name] = Sprite{Sheet: s.name, Rect: p.Rect, Pivot: p.Pivot, Ground: p.Layer == "ground",
				NoShadow: p.Shadow != nil && !*p.Shadow, GateOffset: p.GateOffset}
		}
	}
	if scenery != nil {
		var err error
		if c.Scenery, err = parseScenery(scenery); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// sceneScenery is the scenery with the scene maps' decoration rates and groups (Rules.Decor) over its own.
func (c *Catalog) sceneScenery() *Scenery {
	if c.Scenery == nil || c.Rules.Decor.Scatter == nil {
		return c.Scenery
	}
	s := *c.Scenery
	s.Scatter = map[string]map[string]float64{}
	for surf, groups := range c.Scenery.Scatter {
		s.Scatter[surf] = maps.Clone(groups)
	}
	for surf, groups := range c.Rules.Decor.Scatter {
		if s.Scatter[surf] == nil {
			s.Scatter[surf] = map[string]float64{}
		}
		maps.Copy(s.Scatter[surf], groups)
	}
	s.Groups = maps.Clone(c.Scenery.Groups)
	maps.Copy(s.Groups, c.Rules.Decor.Groups)
	return &s
}

// surface is how a glyph joins its neighbours: 0 water, 1 grass, 2 soil (build.py's SURFACE). Floors join as grass.
func (c *Catalog) surface(g byte) int {
	if s, ok := c.Rules.Surfaces[string(g)]; ok {
		return s
	}
	return 1
}

func (c *Catalog) isGate(name string) bool {
	_, ok := c.Rules.Gates[name]
	return ok
}

// Footprint is the ground an object covers, from its anchor: [du, dv, w, d] in small tiles.
func (c *Catalog) Footprint(name string) [4]float64 { return c.footprint(name) }

func (c *Catalog) footprint(name string) [4]float64 {
	if f, ok := c.Rules.Footprints[name]; ok {
		return f
	}
	return [4]float64{-.5, -.5, 1, 1}
}

// Attach lays out what an object named name carries when its anchor is at canvas point (x, y): the strips to draw
// over it (DrawAnim) and its lights.
func (c *Catalog) Attach(name string, x, y float64) (draws []Draw, lights []Light) {
	for _, a := range c.Rules.Attachments[name] {
		ax, ay := x+float64(a.At[0]), y+float64(a.At[1])
		if s, ok := c.Anims[a.Anim]; ok {
			draws = append(draws, Draw{Kind: DrawAnim, Name: a.Anim, X: round(ax - float64(s.Pivot[0])), Y: round(ay - float64(s.Pivot[1]))})
		}
		if a.Light > 0 {
			lights = append(lights, Light{ax, ay, a.Light, a.Glow})
		}
	}
	return draws, lights
}

func parseColor(s string) ([4]uint8, error) {
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return [4]uint8{}, fmt.Errorf("colour %q: %w", s, err)
	}
	return [4]uint8{r, g, b, 255}, nil
}
