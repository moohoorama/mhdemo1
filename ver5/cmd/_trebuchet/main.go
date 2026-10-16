// Temporary tool: a 발석거 on a real battle map lobs stones at cells 2–5 away (gui.Lob).
//
//	go run ./cmd/_trebuchet                       # window: fires in turn; Space = next target, click = fire at a cell
//	go run ./cmd/_trebuchet -capture DIR          # writes one flight's frames, a contact sheet and all arcs, then quits
//	go run ./cmd/_trebuchet -sheet X.spr          # another catapult sheet (e.g. a scratch bake of a new motion)
//	go run ./cmd/_trebuchet -launch end           # the old sheet (stone drawn in frame 2): fly when frame 2 ends
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/sprite"
)

const (
	W, H     = 1024, 720
	releaseF = 2 // attack frame with the arm about upright: the stone leaves its bucket
	tick     = 1.0 / 60
)

// targets are offsets in cells from the catapult (its range is 2–5).
var targets = [][2]int{{2, 0}, {0, 3}, {-4, 0}, {0, -5}, {3, 2}, {-2, -3}, {5, 0}, {-3, 2}}

type demo struct {
	field    *gui.Field
	art      *sprite.Art
	proj     gui.Projectiles
	release  map[string][2]float64
	launchAt float64 // seconds into the attack animation when the stone leaves

	cu, cv     int
	dir, anim  string
	animT      float64
	shotT      float64 // seconds since the attack started; < 0 while idle
	aim        [2]int
	next       int
	lob        *gui.Lob
	pause      float64
	auto       bool
	zoom       float64
	camX, camY float64
	clockMS    float64
	capture    string
}

func main() {
	capture := flag.String("capture", "", "write PNG frames, a contact sheet and an arc overlay of the flights to this directory and exit")
	mapID := flag.String("map", "", "battle map id (default: the first one)")
	zoom := flag.Float64("zoom", 2, "canvas scale")
	sheet := flag.String("sheet", "assets/sprites/catapult.spr", "catapult sprite sheet")
	launch := flag.String("launch", "start", "start or end: the stone flies from the start or the end of attack frame 2")
	flag.Parse()
	d, err := setup(*mapID, *zoom, *sheet, *launch == "end")
	check(err)
	d.capture = *capture
	ebiten.SetWindowSize(W, H)
	ebiten.SetWindowTitle("발석거 투석 시연")
	if d.capture != "" {
		ebiten.SetWindowSize(320, 200)
	}
	check(ebiten.RunGame(d))
}

func setup(mapID string, zoom float64, sheetPath string, launchAtEnd bool) (*demo, error) {
	data, err := content.Load("assets/data")
	if err != nil {
		return nil, err
	}
	if mapID == "" {
		ids := slices.Sorted(maps.Keys(data.Maps))
		if len(ids) == 0 {
			return nil, fmt.Errorf("no battle maps")
		}
		mapID = ids[0]
	}
	doc := data.Maps[mapID]
	if doc == nil {
		return nil, fmt.Errorf("no map %q", mapID)
	}
	f, err := os.Open("assets/fonts/NotoSansKR.ttf")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	font, err := text.NewGoTextFaceSource(f)
	if err != nil {
		return nil, err
	}
	assets, err := gui.LoadAssets("assets/graphics", font)
	if err != nil {
		return nil, err
	}
	proj, err := gui.LoadProjectiles("assets/graphics")
	if err != nil {
		return nil, err
	}
	sheet, err := sprite.Load(sheetPath)
	if err != nil {
		return nil, err
	}
	field := gui.NewField(assets, mapArt(doc, data))
	field.ShowExterior = true
	art := sprite.NewLibrary(map[string]*sprite.Sheet{sheet.ID: sheet}, shadowRGB).Art(sheet.ID, nil)
	ms := art.Animations["attack"].MS
	n := releaseF
	if launchAtEnd {
		n++
	}
	launch := 0
	for _, v := range ms[:n] {
		launch += v
	}
	d := &demo{field: field, art: art, proj: proj, release: gui.ReleasePoints(sheet, "attack", releaseF),
		launchAt: float64(launch) / 1000, dir: "SE", anim: "idle", shotT: -1, auto: true, zoom: zoom, pause: .6}
	d.cu, d.cv = openCell(field)
	d.camX, d.camY = field.Center(float64(d.cu), float64(d.cv))
	d.camY -= 30
	return d, nil
}

// shadowRGB is assets/graphics/index.json's shadow_rgb (LoadAssets keeps it inside its library).
var shadowRGB = color.RGBA{18, 26, 14, 255}

// mapArt mirrors gui.artFromDocument (unexported) for a battle map document.
func mapArt(doc *content.MapDocument, data *content.Data) *gui.MapArt {
	a := doc.Graphics(data)
	m := &gui.MapArt{ID: a.ID, Name: a.Name, W: a.Size[0], H: a.Size[1], Tiles: a.Tiles, Native: true, Deformed: true,
		Bounds: image.Rect(a.Bounds[0], a.Bounds[1], a.Bounds[2], a.Bounds[3]), Visual: a.Visual, SmallTiles: a.SmallTiles}
	num := func(v any) float64 {
		switch n := v.(type) {
		case int:
			return float64(n)
		case float64:
			return n
		}
		return 0
	}
	for _, t := range a.Ground {
		m.Ground = append(m.Ground, gui.GroundTile{X: t.X, Y: t.Y, Layers: t.Layers})
	}
	for _, d := range a.Decorations {
		name, _ := d[2].(string)
		m.Decorations = append(m.Decorations, gui.Decoration{X: int(num(d[0])), Y: int(num(d[1])), Animation: name, Phase: int(num(d[3]))})
	}
	for _, p := range a.Props {
		name, _ := p[0].(string)
		v := gui.MapProp{Name: name, U: num(p[1]), V: num(p[2])}
		if len(p) >= 5 {
			v.Fill, v.U1, v.V1 = true, num(p[3]), num(p[4])
		}
		m.Props = append(m.Props, v)
	}
	return m
}

// openCell is the cell whose targets, then most cells within five, are open ground (plain or barren).
func openCell(f *gui.Field) (int, int) {
	bu, bv, best := f.Map.W/2, f.Map.H/2, -1
	for v := 0; v < f.Map.H; v++ {
		for u := 0; u < f.Map.W; u++ {
			n := 0
			for _, t := range targets { // every target on open ground counts most
				if c := f.Tile(u+t[0], v+t[1]); c == '.' || c == 'd' {
					n += 1000
				}
			}
			for dv := -5; dv <= 5; dv++ {
				for du := -5; du <= 5; du++ {
					if t := f.Tile(u+du, v+dv); t == '.' || t == 'd' {
						n++
					}
				}
			}
			if n > best {
				bu, bv, best = u, v, n
			}
		}
	}
	return bu, bv
}

var directions = []string{"E", "SE", "S", "SW", "W", "NW", "N", "NE"} // by screen angle from +x, as gui.facing

func facing(du, dv float64) string {
	sx, sy := (du-dv)*16, (du+dv)*8
	a := math.Mod(math.Atan2(sy, sx)*180/math.Pi+360, 360)
	return directions[int((a+22.5)/45)%8]
}

// fire starts the attack at a cell offset from the catapult.
func (d *demo) fire(du, dv int) {
	d.aim = [2]int{du, dv}
	d.dir = facing(float64(du), float64(dv))
	d.anim, d.animT, d.shotT, d.lob = "attack", 0, 0, nil
}

func (d *demo) feet() (float64, float64) {
	x, y := d.field.Center(float64(d.cu), float64(d.cv))
	return x, y - d.field.Lift(d.cu, d.cv)
}

func (d *demo) step(dt float64) {
	d.clockMS += dt * 1000
	d.animT += dt
	if d.anim == "attack" && d.animT >= d.art.Length("attack") {
		d.anim, d.animT = "idle", 0
	}
	if d.shotT >= 0 {
		before := d.shotT
		d.shotT += dt
		if before < d.launchAt && d.shotT >= d.launchAt {
			fx, fy := d.feet()
			r := d.release[d.dir]
			tu, tv := d.cu+d.aim[0], d.cv+d.aim[1]
			tx, ty := d.field.Center(float64(tu), float64(tv))
			d.lob = gui.NewLob("stone", fx+r[0], fy, -r[1], tx, ty-d.field.Lift(tu, tv))
			d.lob.Update(d.shotT - d.launchAt)
		} else if d.lob != nil {
			d.lob.Update(dt)
		}
		if d.lob != nil && d.lob.Done(d.proj) {
			d.shotT, d.lob, d.pause = -1, nil, .5
		}
		return
	}
	if d.auto {
		if d.pause -= dt; d.pause <= 0 {
			t := targets[d.next%len(targets)]
			d.next++
			d.fire(t[0], t[1])
		}
	}
}

func (d *demo) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		t := targets[d.next%len(targets)]
		d.next++
		d.fire(t[0], t[1])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		d.auto = !d.auto
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		u, v := d.field.Cell((float64(x)-W/2)/d.zoom+d.camX, (float64(y)-H/2)/d.zoom+d.camY)
		if u != d.cu || v != d.cv {
			d.fire(u-d.cu, v-d.cv)
		}
	}
	speed := 1.0
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		speed = .25 // hold S for slow motion
	}
	d.step(tick * speed)
	return nil
}

func (d *demo) toScreen(x, y float64) (float64, float64) {
	return (x-d.camX)*d.zoom + W/2, (y-d.camY)*d.zoom + H/2
}

// scene draws the field, the target marks and the catapult; the stone goes on top.
func (d *demo) scene(dst *ebiten.Image) {
	var marks []gui.Highlight
	for _, t := range targets {
		c := color.RGBA{240, 200, 90, 70}
		if d.shotT >= 0 && t == d.aim {
			c = color.RGBA{255, 90, 60, 150}
		}
		marks = append(marks, gui.Highlight{U: d.cu + t[0], V: d.cv + t[1], Color: c})
	}
	d.field.Draw(d.clockMS, nil, marks)
	c := d.field.Canvas
	x, y := d.feet()
	i := d.art.FrameIndex(d.anim, d.animT*1000)
	sop := &ebiten.DrawImageOptions{}
	sop.GeoM.Translate(math.Round(x-d.art.ShadowPivot[0]), math.Round(y-d.art.ShadowPivot[1]))
	c.DrawImage(d.art.Shadow(d.dir, d.anim, i), sop)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x-d.art.Pivot[0]), math.Round(y-d.art.Pivot[1]))
	c.DrawImage(d.art.Frame(d.dir, d.anim, i), op)
	dst.Fill(color.RGBA{30, 38, 46, 255})
	fop := &ebiten.DrawImageOptions{}
	fop.GeoM.Translate(-d.camX, -d.camY)
	fop.GeoM.Scale(d.zoom, d.zoom)
	fop.GeoM.Translate(W/2, H/2)
	dst.DrawImage(c, fop)
}

func (d *demo) Draw(screen *ebiten.Image) {
	if d.capture != "" {
		check(d.captureAll())
		fmt.Println(d.capture)
		os.Exit(0)
	}
	d.scene(screen)
	if d.lob != nil {
		d.lob.Draw(screen, d.proj, d.toScreen, d.zoom)
	}
	ebitenutil.DebugPrint(screen, "Space: next target  click: fire at cell  A: auto on/off  hold S: slow motion  Esc: quit")
}

func (d *demo) Layout(int, int) (int, int) { return W, H }

// captureAll replays shots with a fixed clock: every frame of the longest one, a contact
// sheet of it, and one image with every target's arc traced by the stone.
func (d *demo) captureAll() error {
	frames := filepath.Join(d.capture, "frames")
	if err := os.MkdirAll(frames, 0755); err != nil {
		return err
	}
	screen := ebiten.NewImage(W, H)
	read := func() *image.RGBA {
		im := image.NewRGBA(image.Rect(0, 0, W, H))
		screen.ReadPixels(im.Pix)
		return im
	}
	// the longest shot, frame by frame from the start of the attack
	t := targets[6]
	d.fire(t[0], t[1])
	var shots, lands []*image.RGBA
	fx, fy := d.toScreen(d.feet())
	box := image.Rect(int(fx)-70, int(fy)-120, int(fx)+70, int(fy)+30)
	for n := 0; d.shotT >= 0; n++ {
		if n%2 == 0 {
			d.scene(screen)
			if d.lob != nil {
				d.lob.Draw(screen, d.proj, d.toScreen, d.zoom)
				x, y, _ := d.lob.Pos()
				sx, sy := d.toScreen(x, y)
				box = box.Union(image.Rect(int(sx)-40, int(sy)-60, int(sx)+40, int(sy)+50))
			}
			ebitenutil.DebugPrint(screen, fmt.Sprintf("t=%.3fs", d.shotT))
			im := read()
			if d.shotT >= d.launchAt-.3 { // the contact sheet starts just before the release
				shots = append(shots, im)
			}
			if d.lob != nil && d.lob.T >= d.lob.Dur-2*tick {
				lx, ly := d.toScreen(d.lob.TX, d.lob.TY)
				lands = append(lands, im.SubImage(image.Rect(int(lx)-48, int(ly)-40, int(lx)+48, int(ly)+24)).(*image.RGBA))
			}
			if err := save(filepath.Join(frames, fmt.Sprintf("f%03d.png", n/2)), im); err != nil {
				return err
			}
		}
		d.step(tick)
	}
	// landing close-up: each frame as shown (top row) and three times larger (below)
	const big = 3
	landing := image.NewRGBA(image.Rect(0, 0, len(lands)*(96*big+4), 64+4+64*big))
	for i, im := range lands {
		x0 := i * (96*big + 4)
		b := im.Bounds()
		draw.Draw(landing, image.Rect(x0, 0, x0+96, 64), im, b.Min, draw.Src)
		for y := 0; y < 64*big; y++ {
			for x := 0; x < 96*big; x++ {
				landing.Set(x0+x, 68+y, im.At(b.Min.X+x/big, b.Min.Y+y/big))
			}
		}
	}
	// contact sheet: the region the shot covers, five across, a timestamp in each corner
	crop := box.Intersect(image.Rect(0, 0, W, H))
	cols := 5
	rows := (len(shots) + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*crop.Dx(), rows*crop.Dy()))
	for i, im := range shots {
		at := image.Pt(i%cols*crop.Dx(), i/cols*crop.Dy())
		draw.Draw(sheet, image.Rectangle{at, at.Add(crop.Size())}, im, crop.Min, draw.Src)
		draw.Draw(sheet, image.Rect(at.X, at.Y, at.X+70, at.Y+16), im, image.Pt(0, 0), draw.Src)
	}
	if err := save(filepath.Join(d.capture, "contact.png"), sheet); err != nil {
		return err
	}
	if err := save(filepath.Join(d.capture, "landing.png"), landing); err != nil {
		return err
	}
	// every arc: the scene once, then the stone and its shadow every third tick of each flight
	d.anim, d.animT, d.dir = "idle", 0, "SE"
	d.scene(screen)
	for _, t := range targets {
		d.fire(t[0], t[1])
		for n := 0; d.shotT >= 0; n++ {
			if d.lob != nil && !d.lob.Landed() && n%3 == 0 {
				d.lob.Draw(screen, d.proj, d.toScreen, d.zoom)
			}
			d.step(tick)
		}
	}
	return save(filepath.Join(d.capture, "arcs.png"), read())
}

func save(path string, im image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, im); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
