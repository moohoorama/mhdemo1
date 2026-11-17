package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"log"
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
	// ShadowMask is where a 3D prop's shadow falls, laid from its model (tools/build_extra.py), in place of its cast silhouette
	ShadowMask *struct {
		Rect  [4]int
		Pivot [2]int
	} `json:"shadow_mask"`
}

type Assets struct {
	Duels         duel.Scripts
	Bubble        *duelview.Bubble
	Dragon        *ebiten.Image // grey dragon pattern for narration
	Ruyi          *ebiten.Image // grey ruyi pattern for dialogue
	RuyiMean      float64       // the ruyi pattern's mean grey, 0–1, so a tint can set its mean colour (see ruyiPanel)
	Shou          *ebiten.Image // grey shou pattern for central windows
	groundPattern *ebiten.Image
	grassTints    map[content.GrassTint]*Assets // the tiles and ground decoration with a map's grass, see grassAssets
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
	portraits     map[string]*ebiten.Image // Look.Portrait -> ink-wash portrait, read through portrait on first use
	Props         map[string]Prop
	Scenery       *Scenery                 // ground decoration and pines; nil when graphics/map/scenery.json is absent
	diamonds      map[string]*ebiten.Image // game cell diamonds by colour (field_scene.go)
	floods        map[int]*ebiten.Image    // flood water over a small tile by its scenemap.DrawFlood tile (field_flood.go)
	waterlines    map[[4]int]*ebiten.Image // foam lines round what stands in a flood, by shape (field_flood.go)
	SceneRules    *scenemap.Catalog        // the scene map rules (internal/scenemap) and the decoration's scatter
	Projectiles   Shots                    // thrown stones, their shadows and their landing dust
	effects       Shots                    // effects and their particles; read through effect, which adds a spell on first use
	spells        map[string]bool          // spell effects on disk under effects/spells, read the first time one plays
	dir           string                   // the graphics directory
	Outside       Shots                    // pines and stones outside the battle area
	UI            Shots                    // UI pictures: window frame, marks, icons, unit bar
	Backdrops     map[string]backdrop      // far pictures behind outdoor scene maps (index.json "backdrops")
	// Anims are the looping strips by name (index.json "anims"): flame tongues and smoke puffs the code places, and
	// scene props that move (tree leaves, incense smoke), which play wherever a prop of that name stands
	Anims map[string]FrameStrip
}

// backdrop is a far picture and the colour of its near ground, which runs on down behind the map.
type backdrop struct {
	*ebiten.Image
	ground color.NRGBA
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
	Native      bool
	Visual      [4]int
	SmallTiles  []string
	Deformed    bool // battle map: forests drawn as deformed cells, like villages
	SunkWater   bool // battle map: water lies sunkDepth below the ground behind earth banks (field_water.go)
	// SceneMap is set for a scene map of format 2, drawn by internal/scenemap's rules (field_scene.go); the fields
	// above it are filled from it for the code that reads any map (size, grid, glyphs).
	SceneMap *scenemap.Map
	Scene    bool               // a scenario scene: drawn without the shaded exterior, to the edge of its picture
	Grass    *content.GrassTint // the map's shade of grass, see grassAssets
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

// meanGrey is the mean of a grey picture's red channel, 0–1.
func meanGrey(im *image.RGBA) float64 {
	sum, n := 0, 0
	for i := 0; i < len(im.Pix); i += 4 {
		sum += int(im.Pix[i])
		n++
	}
	return float64(sum) / float64(max(1, n)) / 255
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
	ruyiGrey := patternGrey(ruyi)
	a := &Assets{Bubble: bubble, Dragon: ebiten.NewImageFromImage(dragon), Ruyi: ebiten.NewImageFromImage(ruyiGrey), RuyiMean: meanGrey(ruyiGrey), Shou: ebiten.NewImageFromImage(patternGrey(shou)), Duels: scripts, Tiles: map[int]Sprite{}, TileShadow: map[int]Sprite{},
		Animations: map[string]TileAnimation{}, Structures: map[string]Sprite{}, Sites: map[string]Sprite{}, Maps: map[string]*MapArt{},
		Rise: index.Structures.Rise, CliffRise: index.Structures.CliffRise, portraits: map[string]*ebiten.Image{}, Props: map[string]Prop{}}
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
		a.Tiles[n] = Sprite{cut(sheet, image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])), float64(r[4]), float64(r[5])}
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
		a.TileShadow[n] = Sprite{cut(shadowSheet, image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])), float64(r[4]), float64(r[5])}
	}
	st, err := loadRGBA(filepath.Join(dir, index.Structures.Image))
	if err != nil {
		return nil, err
	}
	stSheet := ebiten.NewImageFromImage(st)
	w, h := index.Structures.Cell[0], index.Structures.Cell[1]
	for name, i := range index.Structures.Sprites {
		a.Structures[name] = Sprite{cut(stSheet, image.Rect(i*w, 0, (i+1)*w, h)),
			index.Structures.Pivot[0], index.Structures.Pivot[1]}
	}
	si, err := loadRGBA(filepath.Join(dir, index.Sites.Image))
	if err != nil {
		return nil, err
	}
	siSheet := ebiten.NewImageFromImage(si)
	for name, i := range index.Sites.Sprites {
		w, h := index.Sites.Cell[0], index.Sites.Cell[1]
		a.Sites[name] = Sprite{cut(siSheet, image.Rect(i*w, 0, (i+1)*w, h)), index.Sites.Pivot[0], index.Sites.Pivot[1]}
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
			pr := Prop{Sprite: Sprite{cut(prSheet, r), p.Pivot[0], p.Pivot[1]}, Ground: p.Layer == "ground",
				NoShadow: p.Shadow != nil && !*p.Shadow}
			if m := p.ShadowMask; m != nil && !pr.Ground && !pr.NoShadow {
				mr := image.Rect(m.Rect[0], m.Rect[1], m.Rect[0]+m.Rect[2], m.Rect[1]+m.Rect[3])
				sd := sprite.MaskShadow(pix.SubImage(mr).(*image.RGBA), m.Pivot)
				pr.Shadow = Sprite{ebiten.NewImageFromImage(sprite.Tint(sd.Image, shade)), float64(sd.Pivot[0]), float64(sd.Pivot[1])}
			} else if !pr.Ground && !pr.NoShadow {
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
			strip.Frames = append(strip.Frames, Sprite{cut(sheet, r), e.Pivot[0], e.Pivot[1]})
			if e.Shadows {
				strip.Shadows = append(strip.Shadows, castShadow(im.SubImage(r).(*image.RGBA), e.Pivot, shade))
			}
		}
		a.Anims[name] = strip
	}
	a.Backdrops = map[string]backdrop{}
	for name, path := range index.Backdrops {
		im, err := loadRGBA(filepath.Join(dir, path))
		if err != nil {
			return nil, err
		}
		b := im.Bounds()
		a.Backdrops[name] = backdrop{ebiten.NewImageFromImage(im), color.NRGBAModel.Convert(im.At(b.Min.X+b.Dx()/2, b.Max.Y-1)).(color.NRGBA)}
	}
	if a.SceneRules, err = scenemap.LoadCatalog(dir); err != nil {
		return nil, err
	}
	for _, path := range index.Maps {
		b, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			return nil, err
		}
		sm, ok, err := scenemap.Parse(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if !ok {
			return nil, fmt.Errorf("%s: a scene map is of format %d", path, scenemap.Format)
		}
		a.Maps[sm.ID] = sceneMapArt(sm)
	}
	if a.Scenery, err = loadScenery(dir); err != nil {
		return nil, err
	}
	if a.Projectiles, err = LoadShots(dir, "effects/projectiles"); err != nil {
		return nil, err
	}
	if a.effects, err = LoadShots(dir, "effects/fx"); err != nil {
		return nil, err
	}
	spellFiles, _ := filepath.Glob(filepath.Join(dir, "effects", "spells", "*.json"))
	a.spells = map[string]bool{}
	for _, f := range spellFiles {
		a.spells[strings.TrimSuffix(filepath.Base(f), ".json")] = true
	}
	a.dir = dir
	if a.Outside, err = LoadShots(dir, "map/outside"); err != nil {
		return nil, err
	}
	if a.UI, err = LoadShots(dir, "ui/ui"); err != nil {
		return nil, err
	}
	return a, nil
}

// portrait is the ink-wash portrait of that name (graphics/portraits/<name>.png), read the first time it is shown:
// there are more than a hundred and a scene shows a few. nil when there is none.
func (a *Assets) portrait(name string) *ebiten.Image {
	if im, ok := a.portraits[name]; ok || name == "" {
		return im
	}
	var im *ebiten.Image
	if pix, err := loadRGBA(filepath.Join(a.dir, "portraits", name+".png")); err == nil {
		im = ebiten.NewImageFromImage(pix)
	} else if !os.IsNotExist(err) {
		log.Printf("portrait %s: %v", name, err)
	}
	a.portraits[name] = im
	return im
}

// lookupEffect is the effect of that name. A spell's sheet (effects/spells/<name>) is read the first time it is asked
// for: the spells together are larger than every other picture, and a battle plays only a few of them.
func (a *Assets) lookupEffect(name string) (Shot, bool) {
	if s, ok := a.effects[name]; ok || !a.spells[name] {
		return s, ok
	}
	shots, err := LoadShots(a.dir, "effects/spells/"+name)
	if err != nil {
		log.Printf("spell effect %s: %v", name, err)
		delete(a.spells, name)
		return Shot{}, false
	}
	maps.Copy(a.effects, shots)
	s, ok := a.effects[name]
	return s, ok
}

// effect is lookupEffect without the found flag; a missing effect has no frames.
func (a *Assets) effect(name string) Shot {
	s, _ := a.lookupEffect(name)
	return s
}

// sceneMapArt fills the common map fields from a format 2 scene map.
func sceneMapArt(m *scenemap.Map) *MapArt {
	b := m.Bounds()
	art := &MapArt{ID: m.ID, Name: m.Name, W: m.Size[0], H: m.Size[1], Native: true, Scene: true, SceneMap: m, Grass: m.Grass,
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
