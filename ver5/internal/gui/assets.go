package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"maps"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"srpg/internal/content"
	"srpg/internal/duel"
	"srpg/internal/duelview"
	"srpg/internal/scenemap"
	"srpg/internal/sprite"
)

// Assets mirrors assets/graphics/index.json (built by tools/build_assets.py).
type propEntry struct {
	Rect   [4]int
	Pivot  [2]float64
	Layer  string
	Shadow *bool // false: no ground shadow under it in a story scene (shadows and overlays, cliffs); unset is true
}

type Assets struct {
	Duels         duel.Scripts
	Bubble        *duelview.Bubble
	Dragon        *ebiten.Image // grey dragon pattern for narration
	Ruyi          *ebiten.Image // grey ruyi pattern for dialogue
	Shou          *ebiten.Image // grey shou pattern for central windows
	groundPattern *ebiten.Image
	StatusIcons   map[string]*ebiten.Image
	Lib           *sprite.Library
	Tiles         map[int]Sprite
	TileShadow    map[int]Sprite
	Animations    map[string]TileAnimation
	Structures    map[string]Sprite
	Sites         map[string]Sprite // one-cell strongpoints (camp, village, castle, gates) at the characters' scale
	Rise          int
	CliffRise     int // how far a cliff block's top stands above its ground cell
	Maps          map[string]*MapArt
	Portraits     map[string]*ebiten.Image // Look.Portrait -> ink-wash portrait
	Props         map[string]Prop
	Scenery       *Scenery                 // ground decoration and pines; nil when graphics/map/scenery.json is absent
	diamonds      map[string]*ebiten.Image // game cell diamonds by colour (field_scene.go)
	SceneRules    *scenemap.Catalog        // the scene map rules (internal/scenemap) and the decoration's scatter
	Projectiles   Shots                    // thrown stones, their shadows and their landing dust
	FX            Shots                    // effects and their particles
	Outside       Shots                    // pines and stones outside the battle area
	UI            Shots                    // UI pictures: window frame, marks, icons, unit bar
	Backdrops     map[string]*ebiten.Image // far pictures behind outdoor scene maps (index.json "backdrops")
	// Anims are the looping strips by name (index.json "anims"): flame tongues and smoke puffs the code places, and
	// scene props that move (tree leaves, incense smoke), which play wherever a prop of that name stands
	Anims map[string]FrameStrip
}

// FrameStrip is an animation whose frames are all the same size, drawn with one pivot.
type FrameStrip struct {
	Frames  []Sprite
	MS      int      // per frame; 0 for a strip stepped by something else (smoke ages)
	Shadows []Sprite // the ground shadow of each frame, for a prop whose outline moves; nil keeps the prop's own
}

// Prop is scene art at the characters' pixel scale (not K times larger like map tiles).
// Ground props (floors, bridges) are baked into the ground; the rest are depth sorted.
type Prop struct {
	Sprite
	Ground   bool
	NoShadow bool   // a story scene lays no ground shadow under it (index.json "shadow": false)
	Shadow   Sprite // its silhouette cast on the ground as the units' shadows are; nil Image for a ground prop or NoShadow
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
	Faces       []MapFace
	Native      bool
	Visual      [4]int
	SmallTiles  []string
	Highland    []string
	SmallSnap   bool
	CliffRepeat bool
	Deformed    bool // battle map: forests drawn as deformed cells, like villages
	SunkWater   bool // battle map: water lies sunkDepth below the ground behind earth banks (field_water.go)
	// SceneMap is set for a scene map of format 2, drawn by internal/scenemap's rules (field_scene.go); the fields
	// above it are filled from it for the code that reads any map (size, grid, glyphs).
	SceneMap *scenemap.Map
	Scene    bool // a scenario scene: drawn without the shaded exterior, to the edge of its picture
}

// MapProp places a prop with its pivot at cell (U, V), fractions allowed. With U1, V1 it
// fills every cell from (U, V) to (U1, V1) (floors).
type MapProp struct {
	Name         string
	U, V, U1, V1 float64
	Fill         bool
}

// MapFace attaches a picture to one edge of a small ground tile; offsets use canvas pixels.
type MapFace struct {
	Name   string
	U, V   int
	Side   string
	DX, DY int
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

func patternGrey(im *image.RGBA) *image.RGBA {
	out := image.NewRGBA(im.Bounds())
	for i := 0; i < len(im.Pix); i += 4 {
		luma := (int(im.Pix[i])*299 + int(im.Pix[i+1])*587 + int(im.Pix[i+2])*114) / 1000
		v := uint8(max(0, min(255, 120+(luma-40)*3/2)))
		out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = v, v, v, im.Pix[i+3]
	}
	return out
}

// shadowed turns a black alpha-only shadow image into the shadow color.
func shadowed(im *image.RGBA, c color.RGBA) *ebiten.Image {
	for i := 0; i < len(im.Pix); i += 4 {
		a := uint32(im.Pix[i+3])
		im.Pix[i], im.Pix[i+1], im.Pix[i+2] = uint8(uint32(c.R)*a/255), uint8(uint32(c.G)*a/255), uint8(uint32(c.B)*a/255)
	}
	return ebiten.NewImageFromImage(im)
}

func LoadAssets(dir string, font *text.GoTextFaceSource, data *content.Data) (*Assets, error) {
	scripts, err := duel.FromData(data)
	if err != nil {
		return nil, err
	}
	bubble, err := duelview.LoadBubble(filepath.Join(dir, "ui", "speech-bubble"), font)
	if err != nil {
		return nil, err
	}
	dragon, err := loadRGBA(filepath.Join(dir, "ui", "dragon.png"))
	if err != nil {
		return nil, err
	}
	ruyi, err := loadRGBA(filepath.Join(dir, "ui", "ruyi-cloud-tile.png"))
	if err != nil {
		return nil, err
	}
	shou, err := loadRGBA(filepath.Join(dir, "ui", "shou-tile.png"))
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, err
	}
	var index struct {
		Tileset struct {
			Image      string
			Sprites    map[string][6]int
			Animations map[string]struct {
				Frames []int
				MS     int
			}
		}
		Maps       []string // narrative map artwork; battle maps come from content documents
		Structures struct {
			Image     string
			Cell      [2]int
			Pivot     [2]float64
			Rise      int
			CliffRise int `json:"cliff_rise"`
			Sprites   map[string]int
		}
		Sites struct {
			Image   string
			Cell    [2]int
			Pivot   [2]float64
			Sprites map[string]int
		}
		Shadows struct {
			Tileset struct {
				Image   string
				Sprites map[string][6]int
			}
		}
		ShadowRGB [3]uint8 `json:"shadow_rgb"`
		Props     struct {
			Image   string
			Sprites map[string]propEntry
		}
		Extra struct { // pictures of assets_candidate, packed by tools/build_extra.py; read as props
			Image   string
			Sprites map[string]propEntry
		}
		Terrain   string // the hills, mountains and roads of tools/worldmap/gen_terrain.py: a sheet like Extra, read as props
		Backdrops map[string]string
		Anims     map[string]struct { // packed by tools/build_extra.py (flames, smoke, incense) and tools/build_tree_sway.py
			Image   string
			Rect    [4]int // frame 0; frame i lies i widths to the right
			Frames  int
			MS      int
			Pivot   [2]float64
			Shadows bool // every frame casts its own ground shadow
		}
	}
	if err = json.Unmarshal(b, &index); err != nil {
		return nil, err
	}
	a := &Assets{Bubble: bubble, Dragon: ebiten.NewImageFromImage(dragon), Ruyi: ebiten.NewImageFromImage(patternGrey(ruyi)), Shou: ebiten.NewImageFromImage(patternGrey(shou)), Duels: scripts, Tiles: map[int]Sprite{}, TileShadow: map[int]Sprite{},
		Animations: map[string]TileAnimation{}, Structures: map[string]Sprite{}, Sites: map[string]Sprite{}, Maps: map[string]*MapArt{},
		Rise: index.Structures.Rise, CliffRise: index.Structures.CliffRise, Portraits: map[string]*ebiten.Image{}, Props: map[string]Prop{}}
	statusPixels, err := loadRGBA(filepath.Join(dir, "status", "icons.png"))
	if err != nil {
		return nil, err
	}
	statusIndex, err := os.ReadFile(filepath.Join(dir, "status", "icons.json"))
	if err != nil {
		return nil, err
	}
	var statusRects map[string][4]int
	if err := json.Unmarshal(statusIndex, &statusRects); err != nil {
		return nil, err
	}
	statusSheet := ebiten.NewImageFromImage(statusPixels)
	a.StatusIcons = make(map[string]*ebiten.Image, len(statusRects))
	for key, r := range statusRects {
		a.StatusIcons[key] = statusSheet.SubImage(image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])).(*ebiten.Image)
	}
	shade := color.RGBA{index.ShadowRGB[0], index.ShadowRGB[1], index.ShadowRGB[2], 255}
	sheets, err := sprite.LoadDir(filepath.Join(dir, "..", "sprites"))
	if err != nil {
		return nil, err
	}
	a.Lib = sprite.NewLibrary(sheets, shade)
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
	si, err := loadRGBA(filepath.Join(dir, index.Sites.Image))
	if err != nil {
		return nil, err
	}
	siSheet := ebiten.NewImageFromImage(si)
	for name, i := range index.Sites.Sprites {
		w, h := index.Sites.Cell[0], index.Sites.Cell[1]
		a.Sites[name] = Sprite{siSheet.SubImage(image.Rect(i*w, 0, (i+1)*w, h)).(*ebiten.Image), index.Sites.Pivot[0], index.Sites.Pivot[1]}
	}
	type propSheet struct {
		Image   string
		Sprites map[string]propEntry
	}
	sheetList := []propSheet{{index.Props.Image, index.Props.Sprites}, {index.Extra.Image, index.Extra.Sprites}}
	if index.Terrain != "" {
		raw, err := os.ReadFile(filepath.Join(dir, index.Terrain))
		if err != nil {
			return nil, err
		}
		var terrain propSheet
		if err := json.Unmarshal(raw, &terrain); err != nil {
			return nil, fmt.Errorf("%s: %w", index.Terrain, err)
		}
		sheetList = append(sheetList, terrain)
	}
	for _, sheet := range sheetList {
		if sheet.Image == "" {
			continue
		}
		pix, err := loadRGBA(filepath.Join(dir, sheet.Image))
		if err != nil {
			return nil, err
		}
		prSheet := ebiten.NewImageFromImage(pix)
		for name, p := range sheet.Sprites {
			r := image.Rect(p.Rect[0], p.Rect[1], p.Rect[0]+p.Rect[2], p.Rect[1]+p.Rect[3])
			pr := Prop{Sprite: Sprite{prSheet.SubImage(r).(*ebiten.Image), p.Pivot[0], p.Pivot[1]}, Ground: p.Layer == "ground",
				NoShadow: p.Shadow != nil && !*p.Shadow}
			if !pr.Ground && !pr.NoShadow {
				pr.Shadow = castShadow(pix.SubImage(r).(*image.RGBA), p.Pivot, shade)
			}
			a.Props[name] = pr
		}
	}
	// the walls of the battle map are the clean ones: no seam where one wall cell meets the next
	for i := 0; i < 16; i++ {
		if clean, ok := a.Props[fmt.Sprintf("wall_clean_%02d", i)]; ok {
			a.Structures[fmt.Sprintf("wall_%02d", i)] = clean.Sprite
		}
	}
	a.Anims = map[string]FrameStrip{}
	type animSheet struct {
		pix *image.RGBA
		img *ebiten.Image
	}
	strips := map[string]animSheet{} // the trees share one sheet
	for name, e := range index.Anims {
		sh, ok := strips[e.Image]
		if !ok {
			pix, err := loadRGBA(filepath.Join(dir, e.Image))
			if err != nil {
				return nil, err
			}
			sh = animSheet{pix, ebiten.NewImageFromImage(pix)}
			strips[e.Image] = sh
		}
		im, sheet := sh.pix, sh.img
		strip := FrameStrip{MS: e.MS}
		x, y, w, h := e.Rect[0], e.Rect[1], e.Rect[2], e.Rect[3]
		for i := range e.Frames {
			r := image.Rect(x+i*w, y, x+(i+1)*w, y+h)
			strip.Frames = append(strip.Frames, Sprite{sheet.SubImage(r).(*ebiten.Image), e.Pivot[0], e.Pivot[1]})
			if e.Shadows {
				strip.Shadows = append(strip.Shadows, castShadow(im.SubImage(r).(*image.RGBA), e.Pivot, shade))
			}
		}
		a.Anims[name] = strip
	}
	a.Backdrops = map[string]*ebiten.Image{}
	for name, path := range index.Backdrops {
		im, err := loadRGBA(filepath.Join(dir, path))
		if err != nil {
			return nil, err
		}
		a.Backdrops[name] = ebiten.NewImageFromImage(im)
	}
	if a.SceneRules, err = scenemap.LoadCatalog(dir); err != nil {
		return nil, err
	}
	for _, path := range index.Maps {
		b, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			return nil, err
		}
		if sm, ok, err := scenemap.Parse(b); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		} else if ok {
			if _, err := scenemap.Build(sm, a.SceneRules, scenemap.Options{Decor: true}); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			a.Maps[sm.ID] = sceneMapArt(sm)
			continue
		}
		var m struct {
			ID, Name    string
			Size        [2]int
			Tiles       []string
			Bounds      [4]int
			Ground      [][3]json.RawMessage
			Decorations [][4]json.RawMessage
			Props       [][]json.RawMessage
			SmallProps  [][]json.RawMessage `json:"small_props"`
			SmallFaces  [][]json.RawMessage `json:"small_faces"`
			Deformed    bool
			Native      bool
			Scene       bool
			Visual      [4]int
			SmallTiles  []string `json:"small_tiles"`
			Highland    []string `json:"highland"`
			SmallSnap   bool     `json:"small_snap"`
			CliffRepeat bool     `json:"cliff_repeat"`
		}
		if err = json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		art := &MapArt{ID: m.ID, Name: m.Name, W: m.Size[0], H: m.Size[1], Tiles: m.Tiles, Deformed: m.Deformed,
			Native: m.Native, Scene: m.Scene, Visual: m.Visual, SmallTiles: m.SmallTiles, Highland: m.Highland,
			SmallSnap: m.SmallSnap, CliffRepeat: m.CliffRepeat,
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
		if len(m.Props) > 0 && len(m.SmallProps) > 0 {
			return nil, fmt.Errorf("%s: props and small_props cannot both be set", path)
		}
		for _, p := range m.SmallProps {
			if len(p) != 3 && len(p) != 5 {
				return nil, fmt.Errorf("%s: small prop needs [name, U, V] or [name, U, V, dx, dy]", path)
			}
			var name string
			var u, v, dx, dy int
			for i, target := range []any{&name, &u, &v} {
				if err := json.Unmarshal(p[i], target); err != nil {
					return nil, fmt.Errorf("%s: small prop coordinate: %w", path, err)
				}
			}
			if len(p) == 5 {
				if err := json.Unmarshal(p[3], &dx); err != nil {
					return nil, fmt.Errorf("%s: small prop dx: %w", path, err)
				}
				if err := json.Unmarshal(p[4], &dy); err != nil {
					return nil, fmt.Errorf("%s: small prop dy: %w", path, err)
				}
			}
			if m.SmallSnap && !strings.HasPrefix(name, "tcliff_") && !strings.HasPrefix(name, "lcliff_") {
				dx, dy = 0, 0
			}
			art.Props = append(art.Props, MapProp{
				Name: name,
				U:    (float64(u)-.5)/2 + float64(dx)/64 + float64(dy)/32,
				V:    (float64(v)-.5)/2 - float64(dx)/64 + float64(dy)/32,
			})
		}
		for _, p := range m.SmallFaces {
			if len(p) != 4 && len(p) != 6 {
				return nil, fmt.Errorf("%s: small face needs [name, U, V, side] or [name, U, V, side, dx, dy]", path)
			}
			var face MapFace
			for i, target := range []any{&face.Name, &face.U, &face.V, &face.Side} {
				if err := json.Unmarshal(p[i], target); err != nil {
					return nil, fmt.Errorf("%s: small face coordinate: %w", path, err)
				}
			}
			if face.Side != "SW" && face.Side != "SE" && face.Side != "NW" && face.Side != "NE" {
				return nil, fmt.Errorf("%s: invalid small face side %q", path, face.Side)
			}
			if len(p) == 6 {
				if err := json.Unmarshal(p[4], &face.DX); err != nil {
					return nil, fmt.Errorf("%s: small face dx: %w", path, err)
				}
				if err := json.Unmarshal(p[5], &face.DY); err != nil {
					return nil, fmt.Errorf("%s: small face dy: %w", path, err)
				}
			}
			art.Faces = append(art.Faces, face)
		}
		a.Maps[m.ID] = art
	}
	if a.Scenery, err = loadScenery(dir); err != nil {
		return nil, err
	}
	if a.Projectiles, err = LoadShots(dir, "effects/projectiles"); err != nil {
		return nil, err
	}
	if a.FX, err = LoadShots(dir, "effects/fx"); err != nil {
		return nil, err
	}
	spells, err := LoadShots(dir, "effects/spells")
	if err != nil {
		return nil, err
	}
	maps.Copy(a.FX, spells)
	if a.Outside, err = LoadShots(dir, "map/outside"); err != nil {
		return nil, err
	}
	if a.UI, err = LoadShots(dir, "ui/ui"); err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(filepath.Join(dir, "portraits", "*.png"))
	for _, file := range files {
		im, err := loadRGBA(file)
		if err != nil {
			return nil, err
		}
		a.Portraits[strings.TrimSuffix(filepath.Base(file), ".png")] = ebiten.NewImageFromImage(im)
	}
	return a, nil
}

// sceneMapArt fills the common map fields from a format 2 scene map.
func sceneMapArt(m *scenemap.Map) *MapArt {
	b := m.Bounds()
	art := &MapArt{ID: m.ID, Name: m.Name, W: m.Size[0], H: m.Size[1], Native: true, Scene: true, SceneMap: m,
		Visual: [4]int{m.Origin[0], m.Origin[1], m.W(), m.H()}, SmallTiles: m.Ground, Bounds: image.Rect(b[0], b[1], b[2], b[3])}
	for v := range m.Size[1] {
		row := make([]byte, m.Size[0])
		for u := range row {
			row[u] = m.Glyph(2*u, 2*v)
		}
		art.Tiles = append(art.Tiles, string(row))
	}
	return art
}

// TileFrame is the sprite id of a tileset animation at a clock time.
func (a *Assets) TileFrame(name string, ms float64) int {
	t := a.Animations[name]
	return t.Frames[int(ms/float64(t.MS))%len(t.Frames)]
}

// castShadow is the ground shadow of a picture standing on pivot, tinted c.
func castShadow(img *image.RGBA, pivot [2]float64, c color.RGBA) Sprite {
	sd := sprite.ImageShadow(img, [2]int{int(math.Round(pivot[0])), int(math.Round(pivot[1]))})
	return Sprite{ebiten.NewImageFromImage(sprite.Tint(sd.Image, c)), float64(sd.Pivot[0]), float64(sd.Pivot[1])}
}
