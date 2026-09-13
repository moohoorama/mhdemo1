package main

import (
	"bytes"
	"demo1/internal/terrain"
	"embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/tiles/*.png assets/objects/*.png
var assets embed.FS

const screenW, screenH, panelW = 1280, 800, 280

var (
	ink    = color.NRGBA{26, 36, 42, 255}
	muted  = color.NRGBA{119, 144, 151, 255}
	accent = color.NRGBA{125, 215, 197, 255}
)

type button struct {
	y      int
	label  string
	action string
}

var buttons = []button{
	{96, "1   RIVER", "water"}, {132, "2   WASTELAND", "land"}, {168, "3   GRASSLAND", "grass"},
	{204, "4   ROCK / GRASS", "rockgrass"}, {240, "5   ROCK / SOIL", "rockland"}, {276, "6   TREES / GRASS", "forest"},
	{346, "S   SAVE MAP", "save"}, {386, "L   LOAD MAP", "load"}, {426, "Z   UNDO       Y   REDO", "undo"},
	{466, "G   GRID / SUBTILES", "grid"}, {506, "F   FIT MAP", "fit"}, {546, "R   RESTORE DEMO", "demo"},
}

func brushKind(action string) (terrain.Kind, bool) {
	switch action {
	case "water":
		return terrain.River, true
	case "land":
		return terrain.Wasteland, true
	case "grass":
		return terrain.Grass, true
	case "rockgrass":
		return terrain.RockGrass, true
	case "rockland":
		return terrain.RockWasteland, true
	case "forest":
		return terrain.Forest, true
	}
	return 0, false
}

type game struct {
	w                        *terrain.World
	tiles                    [terrain.AssetCount]*ebiten.Image
	raw                      [terrain.AssetCount]image.Image
	objectRaw                [terrain.ObjectSpriteCount]image.Image
	objectImages             [terrain.ObjectSpriteCount]*ebiten.Image
	treeTick                 int
	swaySpeed                int
	swayElapsed              time.Duration
	swayUpdated              time.Time
	zoom                     int
	panX, panY               float64
	brush                    terrain.Kind
	grid                     bool
	radius                   int
	undo, redo               []*terrain.World
	stroke                   *terrain.World
	strokeChanged            bool
	lastPaintX, lastPaintY   int
	hasLastPaint             bool
	lastMouseX, lastMouseY   int
	panning                  bool
	status, path, screenshot string
	frames                   int
	started                  time.Time
	waterFrame               int
	captured                 bool
	dirty                    bool
}

func newGame(path, screenshot string) (*game, error) {
	g := &game{started: time.Now(), swaySpeed: 2, w: terrain.Demo(), zoom: 3, brush: terrain.River, path: path, screenshot: screenshot, status: "Ready. Paint river, soil or grass."}
	for i := 0; i < terrain.AssetCount; i++ {
		b, err := assets.ReadFile("assets/tiles/" + terrain.AssetName(i))
		if err != nil {
			return nil, err
		}
		im, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		if im.Bounds() != image.Rect(0, 0, 16, 8) {
			return nil, fmt.Errorf("tile %d must be 16x8", i)
		}
		g.raw[i] = im
		g.tiles[i] = ebiten.NewImageFromImage(im)
	}
	objectFS, err := fs.Sub(assets, "assets/objects")
	if err != nil {
		return nil, err
	}
	g.objectRaw, err = terrain.LoadObjects(objectFS)
	if err != nil {
		return nil, err
	}
	for i, im := range g.objectRaw {
		g.objectImages[i] = ebiten.NewImageFromImage(im)
	}
	if w, err := terrain.Load(path); err == nil {
		g.w = w
		g.status = "Loaded " + filepath.Base(path)
	} else if !os.IsNotExist(err) {
		g.status = "Load failed; showing demo: " + err.Error()
	}
	g.fit()
	return g, nil
}
func (g *game) fit() {
	w, h := (g.w.Width+g.w.Height)*16, (g.w.Width+g.w.Height)*8
	g.zoom = max(1, min(8, min((screenW-panelW-70)/w, (screenH-120)/h)))
	g.panX = float64(panelW + (screenW-panelW-w*g.zoom)/2 + g.w.Height*16*g.zoom)
	g.panY = float64(65 + (screenH-120-h*g.zoom)/2)
}
func (g *game) pos(u, v float64) (float32, float32) {
	x, y := terrain.Project(u, v)
	return float32(g.panX + x*float64(g.zoom)), float32(g.panY + y*float64(g.zoom))
}
func (g *game) mouseCell() (int, int) {
	x, y := ebiten.CursorPosition()
	return terrain.CellAt((float64(x)-g.panX)/float64(g.zoom), (float64(y)-g.panY)/float64(g.zoom))
}
func (g *game) pushUndo(w *terrain.World) {
	g.undo = append(g.undo, w)
	if len(g.undo) > 100 {
		g.undo = g.undo[1:]
	}
	g.redo = nil
}
func (g *game) finishStroke() {
	if g.stroke != nil && g.strokeChanged {
		g.pushUndo(g.stroke)
		g.dirty = true
	}
	g.stroke = nil
	g.strokeChanged = false
	g.hasLastPaint = false
}
func (g *game) action(a string) {
	g.finishStroke()
	if kind, ok := brushKind(a); ok {
		g.brush = kind
		return
	}
	switch a {
	case "water":
		g.brush = terrain.River
	case "land":
		g.brush = terrain.Wasteland
	case "grass":
		g.brush = terrain.Grass
	case "grid":
		g.grid = !g.grid
	case "fit":
		g.fit()
	case "undo":
		if len(g.undo) > 0 {
			g.redo = append(g.redo, g.w.Clone())
			g.w = g.undo[len(g.undo)-1]
			g.undo = g.undo[:len(g.undo)-1]
			g.dirty = true
			g.status = "Undid one stroke."
		}
	case "redo":
		if len(g.redo) > 0 {
			g.undo = append(g.undo, g.w.Clone())
			g.w = g.redo[len(g.redo)-1]
			g.redo = g.redo[:len(g.redo)-1]
			g.dirty = true
			g.status = "Redid one stroke."
		}
	case "save":
		if err := g.w.Save(g.path); err != nil {
			g.status = "Save failed: " + err.Error()
		} else {
			g.dirty = false
			g.status = "Saved " + filepath.Base(g.path)
		}
	case "load":
		if w, err := terrain.Load(g.path); err != nil {
			g.status = "Load failed: " + err.Error()
		} else {
			g.pushUndo(g.w.Clone())
			g.w = w
			g.dirty = false
			g.fit()
			g.status = "Loaded " + filepath.Base(g.path)
		}
	case "demo":
		g.pushUndo(g.w.Clone())
		g.w = terrain.Demo()
		g.dirty = true
		g.fit()
		g.status = "Restored demo. Z to undo."
	case "export":
		if err := writePNG("map-export.png", terrain.RenderScene(g.w, g.raw, g.objectRaw, 4, g.waterFrame, g.treeTick)); err != nil {
			g.status = "Export failed: " + err.Error()
		} else {
			g.status = "Exported map-export.png (4x)."
		}
	}
}
func (g *game) paint(x, y int, value terrain.Kind) {
	for dy := -g.radius; dy <= g.radius; dy++ {
		for dx := -g.radius; dx <= g.radius; dx++ {
			if dx*dx+dy*dy <= g.radius*g.radius && g.w.SetTerrain(x+dx, y+dy, value) {
				g.strokeChanged = true
			}
		}
	}
}
func (g *game) animate() {
	g.waterFrame = int(time.Since(g.started)/(time.Second/8)) % 8
	g.advanceSway(time.Now())
}
func (g *game) advanceSway(now time.Time) {
	if g.swayUpdated.IsZero() {
		g.swayUpdated = g.started
	}
	g.swayElapsed = (g.swayElapsed + now.Sub(g.swayUpdated)*time.Duration(g.swaySpeed)) % (8 * time.Second)
	g.swayUpdated = now
	g.treeTick = int(g.swayElapsed/(time.Second/8)) % terrain.TreeCycleTicks
}
func (g *game) Update() error {
	if g.captured {
		return ebiten.Termination
	}
	g.animate()
	mx, my := ebiten.CursorPosition()
	for _, k := range []struct {
		key ebiten.Key
		act string
	}{{ebiten.Key1, "water"}, {ebiten.Key2, "land"}, {ebiten.Key3, "grass"}, {ebiten.Key4, "rockgrass"}, {ebiten.Key5, "rockland"}, {ebiten.Key6, "forest"}, {ebiten.KeyS, "save"}, {ebiten.KeyL, "load"}, {ebiten.KeyZ, "undo"}, {ebiten.KeyY, "redo"}, {ebiten.KeyG, "grid"}, {ebiten.KeyF, "fit"}, {ebiten.KeyR, "demo"}, {ebiten.KeyP, "export"}} {
		if inpututil.IsKeyJustPressed(k.key) {
			g.action(k.act)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		g.radius = max(0, g.radius-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		g.radius = min(4, g.radius+1)
	}
	left := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	right := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	pan := ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) || (left && ebiten.IsKeyPressed(ebiten.KeySpace))
	if mx < panelW {
		g.finishStroke()
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			for i, speed := range []int{1, 2, 4, 8} {
				if mx >= 18+i*62 && mx < 76+i*62 && my >= 626 && my < 660 {
					g.advanceSway(time.Now())
					g.swaySpeed = speed
					g.status = fmt.Sprintf("Tree / grass sway: %dx", speed)
				}
			}
			for _, b := range buttons {
				if mx >= 18 && mx < panelW-18 && my >= b.y && my < b.y+34 {
					if b.action == "undo" && mx > 154 {
						g.action("redo")
					} else {
						g.action(b.action)
					}
				}
			}
		}
	} else {
		_, wheel := ebiten.Wheel()
		if wheel != 0 {
			old := g.zoom
			if wheel > 0 {
				g.zoom++
			} else {
				g.zoom--
			}
			g.zoom = max(1, min(12, g.zoom))
			g.panX = math.Round(float64(mx) - (float64(mx)-g.panX)*float64(g.zoom)/float64(old))
			g.panY = math.Round(float64(my) - (float64(my)-g.panY)*float64(g.zoom)/float64(old))
		}
		if pan {
			g.finishStroke()
			if g.panning {
				g.panX += float64(mx - g.lastMouseX)
				g.panY += float64(my - g.lastMouseY)
			}
		} else if left || right {
			x, y := g.mouseCell()
			if g.w.Inside(x, y) {
				if g.stroke == nil {
					g.stroke = g.w.Clone()
				}
				value := g.brush
				if right {
					value = terrain.Wasteland
				}
				if g.hasLastPaint {
					dx, dy := x-g.lastPaintX, y-g.lastPaintY
					n := max(abs(dx), abs(dy))
					for i := 1; i <= n; i++ {
						g.paint(g.lastPaintX+int(math.Round(float64(dx*i)/float64(n))), g.lastPaintY+int(math.Round(float64(dy*i)/float64(n))), value)
					}
				}
				g.paint(x, y, value)
				g.lastPaintX = x
				g.lastPaintY = y
				g.hasLastPaint = true
			} else {
				g.hasLastPaint = false
			}
		} else {
			g.finishStroke()
		}
	}
	g.panning = pan && mx >= panelW
	g.lastMouseX = mx
	g.lastMouseY = my
	return nil
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func rect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	vector.FillRect(dst, x, y, w, h, c, false)
}
func label(dst *ebiten.Image, s string, x, y int) { ebitenutil.DebugPrintAt(dst, s, x, y) }
func (g *game) diamond(dst *ebiten.Image, u, v, r float64, c color.Color) {
	pts := [][2]float64{{u - r, v - r}, {u + r, v - r}, {u + r, v + r}, {u - r, v + r}}
	for i, p := range pts {
		q := pts[(i+1)%4]
		x, y := g.pos(p[0], p[1])
		xx, yy := g.pos(q[0], q[1])
		vector.StrokeLine(dst, x, y, xx, yy, 1, c, false)
	}
}
func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{17, 25, 30, 255})
	for y := 0; y < g.w.Height; y++ {
		for x := 0; x < g.w.Width; x++ {
			ground := g.w.MasksFor(x, y, terrain.Grass)
			for k, m := range g.w.Masks(x, y) {
				px, py := g.pos(float64(x)+float64(k%2)/2, float64(y)+float64(k/2)/2)
				px -= float32(8 * g.zoom)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.zoom), float64(g.zoom))
				op.GeoM.Translate(float64(px), float64(py))
				for _, i := range terrain.SurfaceLayers(ground[k], m, g.waterFrame) {
					screen.DrawImage(g.tiles[i], op)
				}
			}
		}
	}
	for _, p := range g.w.Placements() {
		index := p.Sprite(g.treeTick)
		sprite := g.objectImages[index]
		x, y := p.Position()
		b := sprite.Bounds()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(g.zoom), float64(g.zoom))
		op.GeoM.Translate(g.panX+(x-float64(b.Dx()/2))*float64(g.zoom), g.panY+(y-float64(b.Dy())+2)*float64(g.zoom))
		screen.DrawImage(sprite, op)
	}
	if g.grid {
		for axis := 0; axis < 2; axis++ {
			n := g.w.Width
			if axis == 1 {
				n = g.w.Height
			}
			for i := 0; i <= n*2; i++ {
				t := float64(i) / 2
				a, b := g.pos(t, 0)
				c, d := g.pos(t, float64(g.w.Height))
				if axis == 1 {
					a, b = g.pos(0, t)
					c, d = g.pos(float64(g.w.Width), t)
				}
				width := float32(.5)
				alpha := uint8(50)
				if i%2 == 0 {
					width = 1
					alpha = 130
				}
				vector.StrokeLine(screen, a, b, c, d, width, color.NRGBA{232, 229, 197, alpha}, false)
			}
		}
	}
	mx, my := ebiten.CursorPosition()
	vx, vy := g.mouseCell()
	selected := mx >= panelW && g.w.Inside(vx, vy)
	if selected {
		for dy := -g.radius; dy <= g.radius; dy++ {
			for dx := -g.radius; dx <= g.radius; dx++ {
				if dx*dx+dy*dy <= g.radius*g.radius && g.w.Inside(vx+dx, vy+dy) {
					g.diamond(screen, float64(vx+dx)+.5, float64(vy+dy)+.5, .5, accent)
				}
			}
		}
	}
	rect(screen, 0, 0, panelW, screenH, ink)
	rect(screen, panelW-1, 0, 1, screenH, color.NRGBA{59, 80, 87, 255})
	label(screen, "RIVER / SOIL / GRASS", 20, 21)
	label(screen, "ISOMETRIC AUTOTILE LAB", 20, 43)
	rect(screen, 20, 76, 240, 1, muted)
	for _, b := range buttons {
		c := color.NRGBA{38, 51, 58, 255}
		kind, isBrush := brushKind(b.action)
		selected := (isBrush && g.brush == kind) || (b.action == "grid" && g.grid)
		if selected {
			c = color.NRGBA{46, 95, 95, 255}
		}
		if mx >= 18 && mx < 262 && my >= b.y && my < b.y+34 {
			c = color.NRGBA{61, 81, 86, 255}
		}
		rect(screen, 18, float32(b.y), 244, 34, c)
		if selected {
			rect(screen, 18, float32(b.y), 3, 34, accent)
		}
		label(screen, b.label, 29, b.y+9)
	}
	label(screen, fmt.Sprintf("BRUSH %d   [ / ] to resize", g.radius+1), 20, 318)
	label(screen, "TREE / GRASS SWAY SPEED", 20, 602)
	for i, speed := range []int{1, 2, 4, 8} {
		x := 18 + i*62
		c := color.NRGBA{38, 51, 58, 255}
		if g.swaySpeed == speed {
			c = color.NRGBA{46, 95, 95, 255}
		}
		rect(screen, float32(x), 626, 58, 34, c)
		if g.swaySpeed == speed {
			rect(screen, float32(x), 657, 58, 3, accent)
		}
		label(screen, fmt.Sprintf("%dx", speed), x+20, 635)
	}
	label(screen, "LMB paint  /  RMB erase\nWheel zoom / Space-drag pan\nP export PNG / Z-Y history", 20, 690)
	modified := ""
	if g.dirty || g.strokeChanged {
		modified = " *"
	}
	rect(screen, panelW, 0, screenW-panelW, 43, ink)
	label(screen, fmt.Sprintf("%dx%d diamonds  |  zoom %dx  |  %s%s", g.w.Width, g.w.Height, g.zoom, filepath.Base(g.path), modified), panelW+20, 13)
	rect(screen, panelW, screenH-33, screenW-panelW, 33, ink)
	status := g.status
	if len(status) > 115 {
		status = status[:112] + "..."
	}
	label(screen, status, panelW+20, screenH-24)
	if selected {
		label(screen, fmt.Sprintf("Cell %d,%d %s\nSoil %v\nDry  %v", vx, vy, g.w.Terrain(vx, vy), g.w.Masks(vx, vy), g.w.MasksFor(vx, vy, terrain.Grass)), 20, 746)
	}
	g.frames++
	if g.screenshot != "" && g.frames >= 3 && !g.captured {
		if err := writePNG(g.screenshot, screen); err != nil {
			log.Printf("screenshot: %v", err)
		}
		g.captured = true
	}
}
func (*game) Layout(int, int) (int, int) { return screenW, screenH }
func writePNG(path string, im image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err = png.Encode(f, im); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
func main() {
	path := flag.String("map", "map.json", "map JSON to load/save")
	shot := flag.String("screenshot", "", "save one rendered editor frame and exit")
	flag.Parse()
	g, err := newGame(*path, *shot)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("demo1 - River / Wasteland Autotile Editor")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err = ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
