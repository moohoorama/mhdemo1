package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// Assets mirrors assets/graphics/index.json (built by tools/build_assets.py).
type Assets struct {
	Units      map[string]*UnitArt
	Tiles      map[int]Sprite
	TileShadow map[int]Sprite
	Animations map[string]TileAnimation
	Structures map[string]Sprite
	Rise       int
	Maps       map[string]*MapArt
	Factions   map[string][4]color.RGBA
	teamKeys   [4]color.RGBA
	Portraits  map[string]*ebiten.Image // officer art key -> ink-wash portrait
	Props      map[string]Prop
}

// Prop is scene art at the characters' pixel scale (not K times larger like map tiles).
// Ground props (floors, bridges) are baked into the ground; the rest are depth sorted.
type Prop struct {
	Sprite
	Ground bool
}

// Sprite is an image whose pivot lands on the drawing position.
type Sprite struct {
	Image          *ebiten.Image
	PivotX, PivotY float64
}

type TileAnimation struct {
	Frames []int
	MS     int
}

type Animation struct {
	FirstColumn int   `json:"first_column"`
	Frames      int   `json:"frames"`
	MS          []int `json:"ms"`
	Loop        bool  `json:"loop"`
}

// UnitArt is one character image: a row per facing, animations side by side.
// Team-colored pixels are recolored per faction on first use.
type UnitArt struct {
	Name        string
	cell        [2]int
	Pivot       [2]float64
	rows        map[string]int
	Animations  map[string]Animation
	source      *image.RGBA
	sheets      map[string]*ebiten.Image
	shadow      *ebiten.Image
	shadowCell  [2]int
	ShadowPivot [2]float64
	teamKeys    [4]color.RGBA
	factions    map[string][4]color.RGBA
}

// MapArt is a battle map's draw data in map pixels: cell (u, v) has its centre at
// ((u - v) * 16, (u + v + 1) * 8).
type MapArt struct {
	ID, Name    string
	W, H        int
	Tiles       []string
	Bounds      image.Rectangle
	Ground      []GroundTile
	Decorations []Decoration
	Props       []MapProp
	Deformed    bool // battle map: forests and mountains drawn as deformed cells, like villages
}

// MapProp places a prop with its pivot at cell (U, V), fractions allowed. With U1, V1 it
// fills every cell from (U, V) to (U1, V1) (floors).
type MapProp struct {
	Name         string
	U, V, U1, V1 float64
	Fill         bool
}
type GroundTile struct {
	X, Y   int
	Layers []int
}
type Decoration struct {
	X, Y      int
	Animation string
	Phase     int
}

func hexColor(s string) color.RGBA {
	v, _ := strconv.ParseUint(s[1:], 16, 32)
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func loadRGBA(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := image.NewRGBA(im.Bounds())
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	return out, nil
}

// shadowed turns a black alpha-only shadow image into the shadow color.
func shadowed(im *image.RGBA, c color.RGBA) *ebiten.Image {
	for i := 0; i < len(im.Pix); i += 4 {
		a := uint32(im.Pix[i+3])
		im.Pix[i], im.Pix[i+1], im.Pix[i+2] = uint8(uint32(c.R)*a/255), uint8(uint32(c.G)*a/255), uint8(uint32(c.B)*a/255)
	}
	return ebiten.NewImageFromImage(im)
}

func LoadAssets(dir string) (*Assets, error) {
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, err
	}
	var index struct {
		Units map[string]struct {
			Name       string
			Image      string
			Cell       [2]int
			Pivot      [2]float64
			Rows       []string
			Animations map[string]Animation
		}
		Tileset struct {
			Image      string
			Sprites    map[string][6]int
			Animations map[string]struct {
				Frames []int
				MS     int
			}
		}
		Factions struct {
			TeamKeys []string `json:"team_keys"`
			Factions []struct {
				ID   string
				Ramp []string
			}
		}
		Maps       []string
		Structures struct {
			Image   string
			Cell    [2]int
			Pivot   [2]float64
			Rise    int
			Sprites map[string]int
		}
		Shadows struct {
			Units map[string]struct {
				Image string
				Cell  [2]int
				Pivot [2]float64
			}
			Tileset struct {
				Image   string
				Sprites map[string][6]int
			}
		}
		ShadowRGB [3]uint8 `json:"shadow_rgb"`
		Portraits map[string]string
		Props     struct {
			Image   string
			Sprites map[string]struct {
				Rect  [4]int
				Pivot [2]float64
				Layer string
			}
		}
	}
	if err = json.Unmarshal(b, &index); err != nil {
		return nil, err
	}
	a := &Assets{Units: map[string]*UnitArt{}, Tiles: map[int]Sprite{}, TileShadow: map[int]Sprite{},
		Animations: map[string]TileAnimation{}, Structures: map[string]Sprite{}, Maps: map[string]*MapArt{},
		Factions: map[string][4]color.RGBA{}, Rise: index.Structures.Rise, Portraits: map[string]*ebiten.Image{},
		Props: map[string]Prop{}}
	shade := color.RGBA{index.ShadowRGB[0], index.ShadowRGB[1], index.ShadowRGB[2], 255}
	for i, k := range index.Factions.TeamKeys {
		a.teamKeys[i] = hexColor(k)
	}
	for _, f := range index.Factions.Factions {
		var ramp [4]color.RGBA
		for i, c := range f.Ramp {
			ramp[i] = hexColor(c)
		}
		a.Factions[f.ID] = ramp
	}
	for key, u := range index.Units {
		src, err := loadRGBA(filepath.Join(dir, u.Image))
		if err != nil {
			return nil, err
		}
		s := index.Shadows.Units[key]
		sh, err := loadRGBA(filepath.Join(dir, s.Image))
		if err != nil {
			return nil, err
		}
		art := &UnitArt{Name: u.Name, cell: u.Cell, Pivot: u.Pivot, rows: map[string]int{}, Animations: u.Animations,
			source: src, sheets: map[string]*ebiten.Image{}, shadow: shadowed(sh, shade), shadowCell: s.Cell,
			ShadowPivot: s.Pivot, teamKeys: a.teamKeys, factions: a.Factions}
		for i, d := range u.Rows {
			art.rows[d] = i
		}
		a.Units[key] = art
	}
	tiles, err := loadRGBA(filepath.Join(dir, index.Tileset.Image))
	if err != nil {
		return nil, err
	}
	sheet := ebiten.NewImageFromImage(tiles)
	for id, r := range index.Tileset.Sprites {
		n, _ := strconv.Atoi(id)
		a.Tiles[n] = Sprite{sheet.SubImage(image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])).(*ebiten.Image), float64(r[4]), float64(r[5])}
	}
	for name, v := range index.Tileset.Animations {
		a.Animations[name] = TileAnimation{v.Frames, v.MS}
	}
	shadowTiles, err := loadRGBA(filepath.Join(dir, index.Shadows.Tileset.Image))
	if err != nil {
		return nil, err
	}
	shadowSheet := shadowed(shadowTiles, shade)
	for id, r := range index.Shadows.Tileset.Sprites {
		n, _ := strconv.Atoi(id)
		a.TileShadow[n] = Sprite{shadowSheet.SubImage(image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])).(*ebiten.Image), float64(r[4]), float64(r[5])}
	}
	st, err := loadRGBA(filepath.Join(dir, index.Structures.Image))
	if err != nil {
		return nil, err
	}
	stSheet := ebiten.NewImageFromImage(st)
	w, h := index.Structures.Cell[0], index.Structures.Cell[1]
	for name, i := range index.Structures.Sprites {
		a.Structures[name] = Sprite{stSheet.SubImage(image.Rect(i*w, 0, (i+1)*w, h)).(*ebiten.Image),
			index.Structures.Pivot[0], index.Structures.Pivot[1]}
	}
	if index.Props.Image != "" {
		pr, err := loadRGBA(filepath.Join(dir, index.Props.Image))
		if err != nil {
			return nil, err
		}
		prSheet := ebiten.NewImageFromImage(pr)
		for name, p := range index.Props.Sprites {
			r := image.Rect(p.Rect[0], p.Rect[1], p.Rect[0]+p.Rect[2], p.Rect[1]+p.Rect[3])
			a.Props[name] = Prop{Sprite{prSheet.SubImage(r).(*ebiten.Image), p.Pivot[0], p.Pivot[1]}, p.Layer == "ground"}
		}
	}
	for _, path := range index.Maps {
		b, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			return nil, err
		}
		var m struct {
			ID, Name    string
			Size        [2]int
			Tiles       []string
			Bounds      [4]int
			Ground      [][3]json.RawMessage
			Decorations [][4]json.RawMessage
			Props       [][]json.RawMessage
			Deformed    bool
		}
		if err = json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		art := &MapArt{ID: m.ID, Name: m.Name, W: m.Size[0], H: m.Size[1], Tiles: m.Tiles, Deformed: m.Deformed,
			Bounds: image.Rect(m.Bounds[0], m.Bounds[1], m.Bounds[2], m.Bounds[3])}
		for _, g := range m.Ground {
			var t GroundTile
			_ = json.Unmarshal(g[0], &t.X)
			_ = json.Unmarshal(g[1], &t.Y)
			_ = json.Unmarshal(g[2], &t.Layers)
			art.Ground = append(art.Ground, t)
		}
		for _, d := range m.Decorations {
			var t Decoration
			_ = json.Unmarshal(d[0], &t.X)
			_ = json.Unmarshal(d[1], &t.Y)
			_ = json.Unmarshal(d[2], &t.Animation)
			_ = json.Unmarshal(d[3], &t.Phase)
			art.Decorations = append(art.Decorations, t)
		}
		for _, p := range m.Props {
			var mp MapProp
			if len(p) < 3 {
				return nil, fmt.Errorf("%s: prop %s needs [name, u, v]", path, p)
			}
			_ = json.Unmarshal(p[0], &mp.Name)
			_ = json.Unmarshal(p[1], &mp.U)
			_ = json.Unmarshal(p[2], &mp.V)
			if len(p) >= 5 {
				mp.Fill = true
				_ = json.Unmarshal(p[3], &mp.U1)
				_ = json.Unmarshal(p[4], &mp.V1)
			}
			art.Props = append(art.Props, mp)
		}
		a.Maps[m.ID] = art
	}
	for key, file := range index.Portraits {
		im, err := loadRGBA(filepath.Join(dir, file))
		if err != nil {
			return nil, err
		}
		a.Portraits[key] = ebiten.NewImageFromImage(im)
	}
	return a, nil
}

// TileFrame is the sprite id of a tileset animation at a clock time.
func (a *Assets) TileFrame(name string, ms float64) int {
	t := a.Animations[name]
	return t.Frames[int(ms/float64(t.MS))%len(t.Frames)]
}

// sheet is the character image with its team key colors swapped for a faction ramp.
func (u *UnitArt) sheet(faction string) *ebiten.Image {
	if im, ok := u.sheets[faction]; ok {
		return im
	}
	ramp, ok := u.factions[faction]
	if !ok {
		ramp = u.teamKeys
	}
	im := image.NewRGBA(u.source.Bounds())
	copy(im.Pix, u.source.Pix)
	for i := 0; i < len(im.Pix); i += 4 {
		for k, key := range u.teamKeys {
			if im.Pix[i] == key.R && im.Pix[i+1] == key.G && im.Pix[i+2] == key.B && im.Pix[i+3] == 255 {
				im.Pix[i], im.Pix[i+1], im.Pix[i+2] = ramp[k].R, ramp[k].G, ramp[k].B
				break
			}
		}
	}
	e := ebiten.NewImageFromImage(im)
	u.sheets[faction] = e
	return e
}

// FrameIndex is the frame shown ms after an animation started; one-shot animations hold the last frame.
func (u *UnitArt) FrameIndex(anim string, ms float64) int {
	a := u.Animations[anim]
	total := 0
	for _, d := range a.MS {
		total += d
	}
	if a.Loop {
		ms = float64(int(ms) % max(1, total))
	} else if ms >= float64(total) {
		return len(a.MS) - 1
	}
	for i, d := range a.MS {
		if ms < float64(d) {
			return i
		}
		ms -= float64(d)
	}
	return len(a.MS) - 1
}

// Length is an animation's duration in seconds.
func (u *UnitArt) Length(anim string) float64 {
	total := 0
	for _, d := range u.Animations[anim].MS {
		total += d
	}
	return float64(total) / 1000
}

// row is the sheet row of a facing. Scene sprites have only the diagonals; a straight
// facing falls back to one next to it, toward the viewer when it can.
func (u *UnitArt) row(dir string) int {
	if r, ok := u.rows[dir]; ok {
		return r
	}
	return u.rows[map[string]string{"N": "NW", "E": "SE", "S": "SW", "W": "SW"}[dir]]
}

func (u *UnitArt) Frame(faction, dir, anim string, i int) *ebiten.Image {
	w, h := u.cell[0], u.cell[1]
	col, row := u.Animations[anim].FirstColumn+i, u.row(dir)
	return u.sheet(faction).SubImage(image.Rect(col*w, row*h, (col+1)*w, (row+1)*h)).(*ebiten.Image)
}

func (u *UnitArt) Shadow(dir, anim string, i int) *ebiten.Image {
	w, h := u.shadowCell[0], u.shadowCell[1]
	col, row := u.Animations[anim].FirstColumn+i, u.row(dir)
	return u.shadow.SubImage(image.Rect(col*w, row*h, (col+1)*w, (row+1)*h)).(*ebiten.Image)
}

// TopOffset is how far above the pivot the idle south frame's first opaque row is (for HP bars).
func (u *UnitArt) TopOffset() float64 {
	w, h := u.cell[0], u.cell[1]
	col, row := u.Animations["idle"].FirstColumn, u.row("S")
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if u.source.RGBAAt(col*w+x, row*h+y).A > 0 {
				return u.Pivot[1] - float64(y)
			}
		}
	}
	return u.Pivot[1]
}
