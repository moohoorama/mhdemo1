package main

import (
	"demo1/internal/graphics"
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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed assets/generated/terrain.png assets/generated/objects.png assets/generated/catalog.json
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
	catalog                  *graphics.Catalog
	renderer                 *screenRenderer
	sceneCache               terrain.SceneCache
	drawList                 []graphics.DrawItem
	camera                   Camera
	treeTick                 int
	swaySpeed                int
	swayElapsed              time.Duration
	swayUpdated              time.Time
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
	g := &game{started: time.Now(), swaySpeed: 2, w: terrain.Demo(), camera: Camera{Zoom: 3}, brush: terrain.River, path: path, screenshot: screenshot, status: "Ready. Paint river, soil or grass."}
	files, err := fs.Sub(assets, "assets/generated")
	if err != nil {
		return nil, err
	}
	g.catalog, err = graphics.Load(files)
	if err != nil {
		return nil, err
	}
	if err = terrain.ValidateAssets(g.catalog); err != nil {
		return nil, err
	}
	g.renderer = newScreenRenderer(g.catalog)
	if w, err := terrain.Load(path); err == nil {
		g.w = w
		g.status = "Loaded " + filepath.Base(path)
	} else if !os.IsNotExist(err) {
		g.status = "Load failed; showing demo: " + err.Error()
	}
	g.fit()
	g.sceneCache.Ensure(g.w)
	return g, nil
}
func (g *game) fit() {
	w, h := (g.w.Width+g.w.Height)*16, (g.w.Width+g.w.Height)*8
	g.camera.Zoom = max(1, min(8, min((screenW-panelW-70)/w, (screenH-120)/h)))
	g.camera.PanX = float64(panelW + (screenW-panelW-w*g.camera.Zoom)/2 + g.w.Height*16*g.camera.Zoom)
	g.camera.PanY = float64(65 + (screenH-120-h*g.camera.Zoom)/2)
}
func (g *game) pos(u, v float64) (float32, float32) {
	x, y := terrain.Project(u, v)
	return float32(g.camera.PanX + x*float64(g.camera.Zoom)), float32(g.camera.PanY + y*float64(g.camera.Zoom))
}
func (g *game) mouseCell() (int, int) {
	x, y := ebiten.CursorPosition()
	return terrain.CellAt((float64(x)-g.camera.PanX)/float64(g.camera.Zoom), (float64(y)-g.camera.PanY)/float64(g.camera.Zoom))
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
		if err := writePNG("map-export.png", g.exportImage(4)); err != nil {
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
			old := g.camera.Zoom
			if wheel > 0 {
				g.camera.Zoom++
			} else {
				g.camera.Zoom--
			}
			g.camera.Zoom = max(1, min(12, g.camera.Zoom))
			g.camera.PanX = math.Round(float64(mx) - (float64(mx)-g.camera.PanX)*float64(g.camera.Zoom)/float64(old))
			g.camera.PanY = math.Round(float64(my) - (float64(my)-g.camera.PanY)*float64(g.camera.Zoom)/float64(old))
		}
		if pan {
			g.finishStroke()
			if g.panning {
				g.camera.PanX += float64(mx - g.lastMouseX)
				g.camera.PanY += float64(my - g.lastMouseY)
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
func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{17, 25, 30, 255})
	g.renderer.Draw(screen, g.buildDrawList(), g.camera)
	g.drawEditorUI(screen)
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
