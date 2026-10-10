package gui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// Cocoa requires the application run loop on the process main thread.
var rendererMain = make(chan func())

func TestMain(m *testing.M) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		os.Exit(m.Run())
	}
	done := make(chan int, 1)
	go func() { done <- m.Run() }()
	for {
		select {
		case run := <-rendererMain:
			run()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func TestRenderedEditorAndStatuses(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for the native renderer")
	}
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.New(d, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	for _, scene := range g.scenes.Scenes {
		if g.assets.Maps[scene.Map] == nil {
			t.Fatalf("missing narrative map %s", scene.Map)
		}
	}
	out := "../../reports/editor-regression"
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	old := cursorPos
	cursorPos = func() (int, int) { return 0, 0 }
	defer func() { cursorPos = old }()
	test := &editorRenderTest{g: g, out: out, step: -1}
	ebiten.SetWindowSize(Width, Height)
	result := make(chan error, 1)
	rendererMain <- func() { result <- ebiten.RunGame(test) }
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if test.err != nil {
		t.Fatal(test.err)
	}
}

type editorRenderTest struct {
	g     *Game
	out   string
	step  int
	drawn bool
	err   error
	water *image.RGBA
}

func (p *editorRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *editorRenderTest) Update() error {
	if p.step >= 0 && !p.drawn {
		return nil
	}
	p.drawn = false
	p.step++
	g := p.g
	switch p.step {
	case 0:
		g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
		g.editorOpen(0)
		g.editor.doc.PaintSquare(16, 8, 4, "~")
		g.editorRebuild(false)
	case 1:
		g.editor.doc.PaintSquare(16, 8, 4, "b")
		g.editorRebuild(false)
	case 2:
		g.editor.popup = "officers"
		g.editor.family = 1
		g.editor.army = 4
	case 3:
		e := g.editor
		e.popup = "traits"
		for _, id := range sorted(g.S.Data.Characters) {
			c := g.S.Data.Characters[id]
			if c.Template || len(c.Traits) == 0 {
				continue
			}
			traits := append([]string{}, c.Traits...)
			e.draft = &content.Spawn{Character: id, Facing: "SW", Traits: &traits}
			break
		}
		if e.draft == nil {
			p.err = fmt.Errorf("no officer with owned traits")
			return ebiten.Termination
		}
		cursorPos = func() (int, int) { return 250, 210 }
	case 4:
		cursorPos = func() (int, int) { return 0, 0 }
		g.editor.popup = ""
		g.editor.draft = nil
		g.editorTrial()
		if g.editorReturn == nil {
			p.err = fmt.Errorf("trial rejected: %s", g.editor.message)
			return ebiten.Termination
		}
		if _, err := g.S.Engine.Apply(core.Command{Kind: "start"}); err != nil {
			p.err = err
			return ebiten.Termination
		}
		commands, err := g.S.Engine.DeploymentPlan()
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		for _, c := range commands {
			if _, err = g.S.Engine.Apply(c); err != nil {
				p.err = err
				return ebiten.Termination
			}
		}
		snap := g.S.Engine.Snapshot()
		n := 0
		for i := range snap.Units {
			u := &snap.Units[i]
			if u.Faction != "ally" {
				continue
			}
			if n == 0 {
				u.Status = map[string]core.Status{"poison": {Turns: 2}, "root": {Turns: 2}, "disarm": {Turns: 2}, "attack": {Turns: 2, Value: 20}, "defense": {Turns: 2, Value: -20}, "speed": {Turns: 2, Value: 1}}
			}
			if n == 1 {
				u.Status = map[string]core.Status{"burn": {Turns: 2}, "confusion": {Turns: 2}, "seal": {Turns: 2}, "agility": {Turns: 2, Value: 20}, "mind": {Turns: 2, Value: 20}, "morale": {Turns: 2, Value: -20}}
			}
			n++
		}
		restored, err := core.Restore(g.S.Data, snap)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		g.S.Engine = restored
		g.ensureField(g.S.Engine.Observe())
		g.modals = nil
		for i := 0; i < 30; i++ {
			g.history = append(g.history, fmt.Sprintf("전투 기록 %d", i+1))
		}
	case 5:
		cp := g.S.Engine.Snapshot().Checkpoint
		restored, err := core.Restore(g.S.Data, *cp)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		g.S.Engine = restored
		g.field.dispose()
		g.field = nil
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "start"}})
		g.ensureField(g.S.Engine.Observe())
		g.modals = nil
	case 6:
		cursorPos = func() (int, int) { return 120, deploymentTop + 65 }
		g.leftClick()
		cursorPos = func() (int, int) { return 0, 0 }
		if g.deploymentSelected == "" || len(g.deploymentMarks()) == 0 {
			p.err = fmt.Errorf("card click did not expose placements")
			return ebiten.Termination
		}
		g.uiSeconds = .25
	case 7:
		commands, err := g.S.Engine.DeploymentPlan()
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		c := commands[0]
		g.deploymentSelected = c.Actor
		g.deploymentMapClick(c.X, c.Y)
		snap := g.S.Engine.Snapshot()
		restored, err := core.Restore(g.S.Data, snap)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		g.S.Engine = restored
		g.selectDeployment(g.S.Engine.Observe().Deployment[1])
		g.zoom = 1.5
	case 8:
		commands, err := g.S.Engine.DeploymentPlan()
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		for _, c := range commands {
			if !g.request(session.Request{Op: "command", Command: c}).OK {
				p.err = fmt.Errorf("deployment command failed")
				return ebiten.Termination
			}
		}
		g.zoom = 1
	case 9:
		cp := g.S.Engine.Snapshot().Checkpoint
		restored, err := core.Restore(g.S.Data, *cp)
		if err != nil {
			p.err = err
			return ebiten.Termination
		}
		g.S.Engine = restored
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "start"}})
		st := g.S.Data.Stages[g.S.Engine.Observe().Stage]
		legacy := &MapArt{ID: st.ID, W: st.Width, H: st.Height, Tiles: st.Tiles, Deformed: true, Bounds: image.Rect(-st.Height*16, -16, st.Width*16, (st.Width+st.Height)*8)}
		for y := 0; y < st.Height; y++ {
			for x := 0; x < st.Width; x++ {
				for _, xy := range [][2]int{{0, 0}, {8, 4}, {-8, 4}, {0, 8}} {
					legacy.Ground = append(legacy.Ground, GroundTile{X: (x-y)*16 + xy[0], Y: (x+y)*8 + xy[1], Layers: []int{23}})
				}
			}
		}
		g.field.dispose()
		g.field = nil
		g.assets.Maps[st.ID] = legacy
		g.ensureField(g.S.Engine.Observe())
		g.selectDeployment(g.S.Engine.Observe().Deployment[0])
		g.zoom = 1.5
	default:
		return ebiten.Termination
	}
	return nil
}

func (p *editorRenderTest) Draw(dst *ebiten.Image) {
	if p.drawn {
		return
	}
	p.g.Draw(dst)
	img := image.NewRGBA(dst.Bounds())
	dst.ReadPixels(img.Pix)
	names := []string{"water", "bridge", "officers", "traits", "statuses", "deployment-empty", "deployment-selected", "deployment-partial", "deployment-finished", "deployment-legacy"}
	path := filepath.Join(p.out, names[p.step]+".png")
	f, err := os.Create(path)
	if err == nil {
		err = png.Encode(f, img)
		f.Close()
	}
	if err != nil {
		p.err = err
	}
	if p.step == 4 {
		oldScale := scale
		var baseline []byte
		for _, factor := range []float64{1, 2} {
			scale = factor
			icon := ebiten.NewImage(40, 32)
			drawStatusIcon(icon, statusIcon{kind: "attack", tint: statusIconColors["attack"]}, 16, 8, 1)
			pix := make([]byte, 40*32*4)
			icon.ReadPixels(pix)
			if factor == 1 {
				colored := 0
				for y := 3; y < 21; y++ {
					for x := 10; x < 23; x++ {
						i := (y*40 + x) * 4
						if pix[i] > 180 && pix[i+1] < 130 && pix[i+2] < 130 {
							colored++
						}
					}
				}
				if colored < 8 {
					p.err = fmt.Errorf("status icon has no visible red arrow")
				}
				baseline = pix
			} else if !bytes.Equal(baseline, pix) {
				p.err = fmt.Errorf("status icon shifted with display scale")
			}
			icon.Deallocate()
		}
		scale = 2
		high := ebiten.NewImage(Width*2, Height*2)
		p.g.Draw(high)
		highImg := image.NewRGBA(high.Bounds())
		high.ReadPixels(highImg.Pix)
		if file, err := os.Create(filepath.Join(p.out, "statuses-hidpi.png")); err == nil {
			if err = png.Encode(file, highImg); err != nil {
				p.err = err
			}
			file.Close()
		} else {
			p.err = err
		}
		high.Deallocate()
		scale = 1
		gallery := ebiten.NewImage(372, 156)
		gallery.Fill(color.NRGBA{R: 40, G: 45, B: 50, A: 255})
		for i, kind := range []string{"attack", "defense", "agility", "mind", "morale", "speed"} {
			for row := 0; row < 2; row++ {
				x, y := float32(31+i*62), float32(20+row*78)
				drawStatusIcon(gallery, statusIcon{kind: kind, down: row == 1, tint: statusIconColors[kind]}, x, y, 2)
				p.g.label(gallery, map[string]string{"attack": "공격", "defense": "방어", "agility": "순발", "mind": "정신", "morale": "사기", "speed": "이동"}[kind], float64(x-13), float64(y+30), 12, ink)
			}
		}
		preview := image.NewRGBA(gallery.Bounds())
		gallery.ReadPixels(preview.Pix)
		if file, err := os.Create(filepath.Join(p.out, "status-icons.png")); err == nil {
			if err = png.Encode(file, preview); err != nil {
				p.err = err
			}
			file.Close()
		} else {
			p.err = err
		}
		gallery.Deallocate()
		scale = oldScale
	}
	if p.step == 0 {
		p.water = img
	}
	if p.step == 1 {
		e := p.g.editor
		x, y := e.field.Center(8, 4)
		x, y = e.screen(x, y)
		differences := 0
		for py := int(y) - 8; py < int(y)+8; py++ {
			for px := int(x) - 15; px < int(x)+15; px++ {
				if p.water.RGBAAt(px, py) != img.RGBAAt(px, py) {
					differences++
				}
			}
		}
		if differences < 20 {
			p.err = fmt.Errorf("bridge remained hidden: changed pixels %d", differences)
		}
	}
	if p.step == 7 {
		g := p.g
		units := make([]*unitVis, 0, len(g.vis))
		for _, u := range g.S.Engine.Observe().UnitViews {
			units = append(units, g.vis[u.ID])
		}
		pixels := make([]byte, g.field.Canvas.Bounds().Dx()*g.field.Canvas.Bounds().Dy()*4)
		for _, enabled := range []bool{false, true} {
			g.field.setDeployment(enabled, g.S.Engine.Observe().Map.Allies)
			marks := []Highlight(nil)
			if enabled {
				marks = g.deploymentMarks()
			}
			start := time.Now()
			allocs := testing.AllocsPerRun(100, func() { g.field.Draw(0, units, marks); g.field.Canvas.ReadPixels(pixels) })
			fmt.Printf("Field.Draw native B01 1.5x 1280x637 deployment=%t: %.0f ns/frame, %.0f allocs/frame (100 draws plus warmup, GPU readback included)\n", enabled, float64(time.Since(start).Nanoseconds())/101, allocs)
		}
	}
	p.drawn = true
}
