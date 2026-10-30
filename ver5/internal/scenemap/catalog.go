package scenemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
// both are open (_pc).
type WallFamily struct {
	Block   string `json:"block"`
	Parapet string `json:"parapet"`
	Shade   string `json:"shade"`
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
	} `json:"heights"`
	Walls        map[string]WallFamily `json:"walls"`
	Gates        map[string]string     `json:"gates"`        // a gate object -> its slices (<slices>_<axis>_<k>)
	GatePassage  [2]int                `json:"gate_passage"` // the small tiles of a gate's passage, from its point
	CornerShadow struct {
		Sprite string `json:"sprite"`
		Offset [2]int `json:"offset"`
	} `json:"corner_shadow"`
	Footprints map[string][4]float64 `json:"footprints"`
	// Attachments are what an object carries wherever it stands: the flames of a campfire, the light of a lantern.
	Attachments map[string][]Attachment `json:"attachments"`
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

// footprint is the ground an object covers, from its anchor: [du, dv, w, d] in small tiles.
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
