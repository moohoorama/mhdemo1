package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/mapedit"
	"srpg/internal/session"
)

type mapEditor struct {
	previews                                   map[string]core.UnitView
	units                                      []*unitVis
	unitsKey                                   string
	brushOffset                                int
	brushSize                                  int
	officerClass                               string
	trialArt                                   *MapArt
	trialID                                    string
	doc                                        *mapedit.Document
	history                                    mapedit.History
	field                                      *Field
	dirty, rebuild                             bool
	tab, brush, family, classIndex, army, page int
	ally, officer                              bool
	pick                                       string
	level                                      int
	facing                                     string
	selected                                   int
	draft                                      *content.Spawn
	popup, entry, editing, message, query      string
	cx, cy, zoom                               float64
	stroke                                     bool
	pan                                        bool
	rightPending, rightMoved                   bool
	rightStart, rightLast                      image.Point
	lastX, lastY                               int
	lastSmall                                  image.Point
	corner                                     int
	resizing                                   bool
	proposed                                   mapedit.Rect
}

var editorViewport = image.Rect(16, 64, 992, 688)
var editorKeys = []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9, ebiten.Key0, ebiten.KeyQ, ebiten.KeyW, ebiten.KeyE, ebiten.KeyR, ebiten.KeyT, ebiten.KeyY, ebiten.KeyU, ebiten.KeyI, ebiten.KeyO, ebiten.KeyP, ebiten.KeyA, ebiten.KeyS, ebiten.KeyD, ebiten.KeyF, ebiten.KeyG, ebiten.KeyH, ebiten.KeyJ, ebiten.KeyK, ebiten.KeyL, ebiten.KeyZ, ebiten.KeyX, ebiten.KeyC, ebiten.KeyV, ebiten.KeyB, ebiten.KeyN, ebiten.KeyM}

const editorKeyNames = "1234567890qwertyuiopasdfghjklzxcvbnm"

func (g *Game) editorOpen(index int) {
	e := g.editor
	st := g.S.Data.Stages[index]
	original := g.S.Data.Maps[st.ID]
	if original == nil {
		e.message = "맵 문서가 없습니다"
		return
	}
	copy := mapedit.Clone(*original)
	doc := &copy
	e.doc = doc
	e.history = mapedit.History{}
	e.selected = -1
	e.draft = nil
	e.dirty = false
	e.popup = ""
	e.pick = ""
	g.editorRebuild(true)
}
func artFromDocument(d *mapedit.Document, data *content.Data) *MapArt {
	a := d.Graphics(data)
	m := &MapArt{ID: a.ID, Name: a.Name, W: a.Size[0], H: a.Size[1], Tiles: a.Tiles, Bounds: image.Rect(a.Bounds[0], a.Bounds[1], a.Bounds[2], a.Bounds[3]), Native: true, Deformed: true, Visual: a.Visual, SmallTiles: a.SmallTiles}
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
		m.Ground = append(m.Ground, GroundTile{t.X, t.Y, t.Layers})
	}
	for _, d := range a.Decorations {
		name, _ := d[2].(string)
		m.Decorations = append(m.Decorations, Decoration{int(num(d[0])), int(num(d[1])), name, int(num(d[3]))})
	}
	for _, p := range a.Props {
		name, _ := p[0].(string)
		v := MapProp{Name: name, U: num(p[1]), V: num(p[2])}
		if len(p) >= 5 {
			v.Fill = true
			v.U1 = num(p[3])
			v.V1 = num(p[4])
		}
		m.Props = append(m.Props, v)
	}
	return m
}
func (f *Field) dispose() {
	seen := map[*ebiten.Image]bool{}
	for _, im := range append(f.ground[:], f.Canvas, f.clip, f.shadows, f.diamond, f.exterior, f.deploymentMask) {
		if im != nil && !seen[im] {
			im.Deallocate()
			seen[im] = true
		}
	}
}
func (g *Game) editorRebuild(fit bool) {
	e := g.editor
	if e.doc == nil {
		return
	}
	var ox, oy float64
	showExterior := false
	if e.field != nil {
		ox, oy = e.field.OriginX*K, e.field.OriginY*K
		showExterior = e.field.ShowExterior
		e.field.dispose()
	}
	e.field = NewField(g.assets, artFromDocument(e.doc, g.S.Data))
	e.field.ShowExterior = showExterior
	e.rebuild = false
	e.unitsKey = ""
	if fit {
		r := e.doc.Play
		e.cx, e.cy = e.field.Center(float64(r.W-1)/2, float64(r.H-1)/2)
		e.zoom = math.Min(1, math.Min(float64(editorViewport.Dx())/float64((r.W+r.H)*32+128), float64(editorViewport.Dy())/float64((r.W+r.H)*16+128)))
	} else {
		e.cx += e.field.OriginX*K - ox
		e.cy += e.field.OriginY*K - oy
	}
}
func (g *Game) editorChanged() { e := g.editor; e.dirty, e.rebuild = true, true }
func (e *mapEditor) canvas(x, y int) (float64, float64) {
	return (float64(x-editorViewport.Min.X)-float64(editorViewport.Dx())/2)/e.zoom + e.cx, (float64(y-editorViewport.Min.Y)-float64(editorViewport.Dy())/2)/e.zoom + e.cy
}
func (e *mapEditor) screen(x, y float64) (float64, float64) {
	return (x-e.cx)*e.zoom + float64(editorViewport.Min.X+editorViewport.Dx()/2), (y-e.cy)*e.zoom + float64(editorViewport.Min.Y+editorViewport.Dy()/2)
}
func (e *mapEditor) smallAt(x, y int) image.Point {
	px, py := e.canvas(x, y)
	a, b := (px/K-e.field.OriginX)/16, (py/K-e.field.OriginY)/8
	return image.Pt(int(math.Floor(a+b))+e.doc.Play.X*2, int(math.Floor(b-a))+e.doc.Play.Y*2)
}
func (e *mapEditor) cornerAt(r mapedit.Rect, i int) (float64, float64) {
	u, v := r.X, r.Y
	if i == 1 || i == 2 {
		u += r.W
	}
	if i == 2 || i == 3 {
		v += r.H
	}
	x, y := e.field.Center(float64(u-e.doc.Play.X)-.5, float64(v-e.doc.Play.Y)-.5)
	return e.screen(x, y)
}
func (g *Game) updateEditor() error {
	e := g.editor
	g.clockMS += 1000.0 / 60
	pressed := keyJustPressed
	x, y := cursorPos()
	left := mouseJustPressed(ebiten.MouseButtonLeft)
	if pressed(ebiten.KeyF12) {
		g.capture = true
	}
	if e.editing != "" {
		e.entry = string(ebiten.AppendInputChars([]rune(e.entry)))
		if pressed(ebiten.KeyBackspace) {
			r := []rune(e.entry)
			if len(r) > 0 {
				e.entry = string(r[:len(r)-1])
			}
		}
		if pressed(ebiten.KeyEnter) {
			g.editorFinishEntry()
		}
		if pressed(ebiten.KeyEscape) {
			e.editing = ""
		}
		return nil
	}
	if left {
		for i := len(g.buttons) - 1; i >= 0; i-- {
			b := g.buttons[i]
			if image.Pt(x, y).In(b.rect) {
				b.click()
				return nil
			}
		}
	}
	if g.overlay != "" {
		if pressed(ebiten.KeyEscape) {
			g.overlay = ""
		}
		return nil
	}
	if e.popup != "" {
		if pressed(ebiten.KeyEscape) {
			e.popup = ""
		}
		_, wheel := ebiten.Wheel()
		if wheel != 0 {
			e.page = max(0, e.page-int(wheel))
		}
		return nil
	}
	if e.doc == nil {
		if pressed(ebiten.KeyEscape) {
			g.editor = nil
		}
		return nil
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if ctrl {
		if pressed(ebiten.KeyS) {
			g.editorSave(false)
		}
		if pressed(ebiten.KeyZ) {
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				e.history.Redo(&e.doc)
			} else {
				e.history.Undo(&e.doc)
			}
			e.selected = -1
			e.draft = nil
			g.editorChanged()
		}
		return nil
	}
	if e.tab == 0 && y >= 700 {
		_, wheel := ebiten.Wheel()
		if wheel != 0 {
			e.brushOffset = max(0, min(max(0, len(g.S.Data.MapEditor.Brushes)-11), e.brushOffset-int(wheel)))
			return nil
		}
	}
	if pressed(ebiten.KeyTab) {
		e.tab = 1 - e.tab
	}
	if pressed(ebiten.KeyEscape) {
		g.editorLeave()
		return nil
	}
	if e.tab == 0 {
		for i, k := range editorKeys {
			if i < len(g.S.Data.MapEditor.Brushes) && pressed(k) {
				e.brush = i
				e.brushOffset = max(0, i-10)
			}
		}
	}
	if e.rebuild && (!e.stroke || g.ticks%6 == 0) {
		g.editorRebuild(false)
	}
	if e.resizing {
		if mousePressed(ebiten.MouseButtonLeft) {
			px, py := e.canvas(x, y)
			a, b := (px/K-e.field.OriginX)/16, (py/K-e.field.OriginY)/8
			u, v := int(math.Round((a+b)/2))+e.doc.Play.X, int(math.Round((b-a)/2))+e.doc.Play.Y
			r := e.doc.Play
			x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
			if e.corner == 0 || e.corner == 3 {
				x0 = min(u, x1-4)
				x0 = max(x0, x1-mapedit.MaxSize)
			} else {
				x1 = max(u, x0+4)
				x1 = min(x1, x0+mapedit.MaxSize)
			}
			if e.corner == 0 || e.corner == 1 {
				y0 = min(v, y1-4)
				y0 = max(y0, y1-mapedit.MaxSize)
			} else {
				y1 = max(v, y0+4)
				y1 = min(y1, y0+mapedit.MaxSize)
			}
			e.proposed = mapedit.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
			return nil
		}
		e.resizing = false
		r := e.proposed
		n := 0
		for _, sp := range append(append([]content.Spawn{}, e.doc.Stage.Enemies...), e.doc.Stage.Party...) {
			if !r.Contains(sp.X, sp.Y) {
				n++
			}
		}
		for _, p := range e.doc.Stage.Allies {
			if !r.Contains(p.X, p.Y) {
				n++
			}
		}
		g.askWith(fmt.Sprintf("맵 크기 %d×%d → %d×%d\n영역 밖 배치 %d개를 삭제하고 적용할까요?", e.doc.Play.W, e.doc.Play.H, r.W, r.H, n), "예", "아니오", func() {
			e.history.Push(e.doc)
			if err := e.doc.Resize(r); err != nil {
				e.message = err.Error()
				return
			}
			e.selected = -1
			e.draft = nil
			g.editorChanged()
			g.editorRebuild(true)
		})
		return nil
	}
	if left && image.Pt(x, y).In(editorViewport) {
		for i := 0; i < 4; i++ {
			hx, hy := e.cornerAt(e.doc.Play, i)
			if math.Hypot(float64(x)-hx, float64(y)-hy) < 14 {
				e.resizing = true
				e.corner = i
				e.proposed = e.doc.Play
				return nil
			}
		}
	}
	if mouseJustPressed(ebiten.MouseButtonRight) && image.Pt(x, y).In(editorViewport) {
		e.rightPending = true
		e.rightMoved = false
		e.rightStart = image.Pt(x, y)
		e.rightLast = e.rightStart
	}
	if e.rightPending {
		if mousePressed(ebiten.MouseButtonRight) {
			if e.rightMoved || abs(x-e.rightStart.X)+abs(y-e.rightStart.Y) > 4 {
				e.rightMoved = true
				e.cx -= float64(x-e.rightLast.X) / e.zoom
				e.cy -= float64(y-e.rightLast.Y) / e.zoom
				e.rightLast = image.Pt(x, y)
				e.clampCamera()
			}
		} else {
			if !e.rightMoved && image.Pt(x, y).In(editorViewport) {
				p := e.smallAt(x, y)
				if e.tab == 0 {
					for i, t := range g.S.Data.MapEditor.Brushes {
						if t == e.doc.Small(p.X, p.Y) {
							e.brush = i
						}
					}
				} else {
					e.history.Push(e.doc)
					g.editorStamp(p, true)
				}
			}
			e.rightPending = false
		}
		return nil
	}
	pan := mousePressed(ebiten.MouseButtonMiddle) || ebiten.IsKeyPressed(ebiten.KeySpace) && mousePressed(ebiten.MouseButtonLeft)
	if pan {
		if e.pan {
			e.cx -= float64(x-e.lastX) / e.zoom
			e.cy -= float64(y-e.lastY) / e.zoom
		}
		e.pan = true
		e.lastX, e.lastY = x, y
		e.clampCamera()
		return nil
	}
	e.pan = false
	if !pointer.used && image.Pt(x, y).In(editorViewport) && ebiten.IsFocused() && len(g.Shots) == 0 {
		dx, dy := editorEdgeScroll(image.Pt(x, y), editorViewport)
		e.cx += float64(dx) * 8 / e.zoom
		e.cy += float64(dy) * 8 / e.zoom
		e.clampCamera()
	}
	if image.Pt(x, y).In(editorViewport) {
		_, wheel := ebiten.Wheel()
		if wheel != 0 {
			e.zoom = math.Max(.1, math.Min(3, e.zoom*math.Pow(1.15, wheel)))
		}
	}
	held := mousePressed(ebiten.MouseButtonLeft)
	if !held {
		if e.stroke {
			e.stroke = false
			if e.rebuild {
				g.editorRebuild(false)
			}
		}
		return nil
	}
	if !image.Pt(x, y).In(editorViewport) {
		return nil
	}
	point := e.smallAt(x, y)
	big := image.Pt(mapedit.FloorHalf(point.X), mapedit.FloorHalf(point.Y))
	if left {
		if e.tab == 1 && left && !e.ally {
			for i, sp := range e.doc.Stage.Enemies {
				if sp.X == big.X && sp.Y == big.Y {
					e.selected = i
					s := mapedit.Clone(sp)
					e.draft = &s
					e.stroke = false
					return nil
				}
			}
		}
		e.history.Push(e.doc)
		e.stroke = true
		e.lastSmall = point
		g.editorStamp(point, false)
	}
	if e.stroke && point != e.lastSmall {
		dx, dy := point.X-e.lastSmall.X, point.Y-e.lastSmall.Y
		n := max(abs(dx), abs(dy))
		for i := 1; i <= n; i++ {
			p := image.Pt(e.lastSmall.X+dx*i/n, e.lastSmall.Y+dy*i/n)
			g.editorStamp(p, false)
		}
		e.lastSmall = point
	}
	return nil
}
func (g *Game) editorStamp(p image.Point, erase bool) {
	e := g.editor
	d := e.doc
	x, y := mapedit.FloorHalf(p.X), mapedit.FloorHalf(p.Y)
	if e.tab == 0 {
		if d.Visual.Contains(x, y) {
			d.PaintSquare(p.X, p.Y, max(1, e.brushSize), g.S.Data.MapEditor.Brushes[e.brush])
			g.editorChanged()
		}
		return
	}
	if !d.Play.Contains(x, y) {
		e.message = "외곽에는 병력을 배치할 수 없습니다"
		return
	}
	if e.ally {
		for i, q := range d.Stage.Allies {
			if q.X == x && q.Y == y {
				if erase {
					d.Stage.Allies = append(d.Stage.Allies[:i], d.Stage.Allies[i+1:]...)
					e.dirty = true
				}
				return
			}
		}
		if !erase {
			usable := false
			for _, id := range g.S.Data.Campaign.Roster {
				class := g.S.Data.Classes[g.S.Data.Characters[id].Class]
				usable = usable || g.S.Data.Terrain[class.Terrain][d.Tile(x, y)].Cost > 0
			}
			if !usable {
				e.message = "아군이 배치할 수 없는 지형입니다"
				return
			}
			for _, q := range append(append([]content.Spawn{}, d.Stage.Enemies...), d.Stage.Party...) {
				if q.X == x && q.Y == y {
					return
				}
			}
			d.Stage.Allies = append(d.Stage.Allies, content.Point{X: x, Y: y})
			e.dirty = true
		}
		return
	}
	for i, sp := range d.Stage.Enemies {
		if sp.X == x && sp.Y == y {
			if erase {
				d.Stage.Enemies = append(d.Stage.Enemies[:i], d.Stage.Enemies[i+1:]...)
				e.selected = -1
				e.draft = nil
				e.dirty = true
			}
			return
		}
	}
	if erase {
		return
	}
	for _, q := range d.Stage.Allies {
		if q.X == x && q.Y == y {
			e.message = "아군 배치 칸과 겹칩니다"
			return
		}
	}
	for _, q := range d.Stage.Party {
		if q.X == x && q.Y == y {
			return
		}
	}
	class := g.editorClass()
	if _, ok := g.S.Data.Classes[class]; !ok {
		e.message = "미구현 병종입니다"
		return
	}
	pick := e.pick
	if !e.officer {
		pick = ""
		for _, id := range sorted(g.S.Data.Characters) {
			c := g.S.Data.Characters[id]
			if c.Template && g.S.Data.ClassFamily(c.Class) == g.S.Data.ClassFamily(class) {
				pick = id
				break
			}
		}
	}
	if pick == "" {
		e.message = "병력을 먼저 선택하세요"
		return
	}
	c := g.S.Data.Characters[pick]
	if !c.Template {
		for _, s := range append(append([]content.Spawn{}, d.Stage.Enemies...), d.Stage.Party...) {
			if s.Character == pick {
				e.message = "장수는 중복 배치할 수 없습니다"
				return
			}
		}
		for _, id := range g.S.Data.Campaign.Roster {
			if id == pick {
				e.message = "아군 출전 장수와 중복됩니다"
				return
			}
		}
	}
	equipment := []string{}
	cl := g.S.Data.Classes[class]
	for _, id := range c.Equipment {
		eq := g.S.Data.Equipment[id]
		if eq.Slot == "weapon" && eq.Type != cl.Weapon || eq.Slot == "armor" && eq.Type != cl.Armor {
			continue
		}
		equipment = append(equipment, id)
	}
	sp := content.Spawn{Character: pick, X: x, Y: y, Class: class, Level: e.level, Facing: e.facing, Equipment: &equipment}
	d.Stage.Enemies = append(d.Stage.Enemies, sp)
	e.selected = len(d.Stage.Enemies) - 1
	s := mapedit.Clone(sp)
	e.draft = &s
	e.dirty = true
	e.message = "배치됨: " + c.Name
}
func (g *Game) editorClass() string {
	e := g.editor
	fs := g.S.Data.MapEditor.Families
	if len(fs) == 0 {
		return ""
	}
	f := fs[e.family%len(fs)]
	if len(f.Classes) == 0 {
		return ""
	}
	return f.Classes[e.classIndex%len(f.Classes)].ID
}
func (g *Game) editorSave(copyID bool) {
	e := g.editor
	if e.doc == nil {
		return
	}
	if copyID {
		e.editing = "id"
		e.entry = e.doc.Stage.ID + "_copy"
		return
	}
	if err := g.editorValidate(); err != nil {
		e.message = err.Error()
		return
	}
	if err := mapedit.Save(g.root, e.doc, g.S.Data); err != nil {
		e.message = err.Error()
		return
	}
	data, err := content.Load(filepath.Join(g.root, "assets/data"))
	if err != nil {
		e.message = err.Error()
		return
	}
	g.S.Data = data
	g.assets.Maps[e.doc.Stage.ID] = artFromDocument(e.doc, data)
	e.dirty = false
	e.message = "저장 완료 · 캠페인에 적용됨"
}
func (g *Game) editorValidate() error {
	e := g.editor
	d := g.S.Data
	if err := e.doc.Validate(d); err != nil {
		return err
	}
	for _, sp := range append(append([]content.Spawn{}, e.doc.Stage.Enemies...), e.doc.Stage.Party...) {
		v := g.editorPreview(sp)
		if sp.HP != nil && *sp.HP > v.Stats.MaxHP || sp.MP != nil && *sp.MP > v.Stats.MaxMP {
			return fmt.Errorf("%s: HP/MP가 최대값을 초과합니다", v.Name)
		}
	}
	return nil
}
func (g *Game) editorLeave() {
	e := g.editor
	if e.dirty {
		e.popup = "leave"
		return
	}
	if e.field != nil {
		e.field.dispose()
	}
	g.editor = nil
}
func (g *Game) editorTrial() {
	e := g.editor
	if err := g.editorValidate(); err != nil {
		e.message = err.Error()
		return
	}
	d := *g.S.Data
	d.Stages = append([]content.Stage{}, d.Stages...)
	d.Nodes = append([]content.Node{}, d.Nodes...)
	st := e.doc.Compile()
	i := d.StageIndex(st.ID)
	if i < 0 {
		d.Stages = append(d.Stages, st)
	} else {
		d.Stages[i] = st
	}
	node := "editor_trial"
	d.Nodes = append(d.Nodes, content.Node{ID: node, Kind: "preparation", Stage: st.ID})
	d.Campaign.Start = node
	for _, id := range sorted(d.Characters) {
		c := d.Characters[id]
		if c.Playable {
			fixed := false
			for _, sp := range append(append([]content.Spawn{}, st.Enemies...), st.Party...) {
				fixed = fixed || sp.Character == id
			}
			if fixed {
				continue
			}
			found := false
			for _, old := range d.Campaign.Roster {
				found = found || old == id
			}
			if !found {
				d.Campaign.Roster = append(append([]string{}, d.Campaign.Roster...), id)
			}
		}
	}
	d.Campaign.Deployment = d.Campaign.Roster[:min(st.Limit, len(d.Campaign.Roster))]
	trialDir, err := os.MkdirTemp("", "srpg-editor-trial-")
	if err != nil {
		e.message = err.Error()
		return
	}
	e.trialArt = g.assets.Maps[st.ID]
	e.trialID = st.ID
	g.editorSession = g.S
	g.editorReturn = e
	g.editor = nil
	g.S = &session.Session{Data: &d, Engine: core.New(&d, 1), Screen: "playing", Settings: g.editorSession.Settings, Store: g.editorSession.Store}
	g.S.Store.Dir = trialDir
	g.assets.Maps[st.ID] = artFromDocument(e.doc, &d)
	g.field = nil
	g.vis = map[string]*unitVis{}
	g.modals = nil
	g.overlay = ""
	g.resetBattleLog()
	g.resetDeployment()
}
func (g *Game) endEditorTrial() {
	if g.editorReturn == nil {
		return
	}
	g.player.Skip()
	if g.field != nil {
		g.field.dispose()
	}
	g.S = g.editorSession
	e := g.editorReturn
	if e.trialArt != nil {
		g.assets.Maps[e.trialID] = e.trialArt
	} else {
		delete(g.assets.Maps, e.trialID)
	}
	g.editor = g.editorReturn
	g.editorReturn = nil
	g.editorSession = nil
	g.field = nil
	g.vis = map[string]*unitVis{}
	g.modals = nil
	g.overlay = ""
	g.resetBattleLog()
	g.resetDeployment()
}
func (g *Game) editorFinishEntry() {
	e := g.editor
	kind := e.editing
	e.editing = ""
	if kind == "search" {
		e.query = e.entry
		return
	}
	if kind == "id" {
		original := mapedit.Clone(*e.doc)
		wasDirty := e.dirty
		old := e.doc.Stage.ID
		e.doc.Stage.ID = strings.TrimSpace(e.entry)
		if g.S.Data.StageIndex(e.doc.Stage.ID) >= 0 {
			e.message = "이미 있는 맵 ID입니다"
			e.doc.Stage.ID = old
			return
		}
		for i := range e.doc.Stage.Duels {
			e.doc.Stage.Duels[i].ID = e.doc.Stage.ID + fmt.Sprintf("-duel-%d", i+1)
		}
		e.dirty = true
		g.editorSave(false)
		if e.dirty {
			e.doc = &original
			e.dirty = wasDirty
		}
		return
	}
	var n int
	if _, err := fmt.Sscan(e.entry, &n); err != nil {
		e.message = "숫자를 입력하세요"
		return
	}
	if e.draft == nil {
		return
	}
	switch kind {
	case "hp":
		e.draft.HP = &n
	case "mp":
		e.draft.MP = &n
	case "level":
		e.draft.Level = n
	}
}
func (g *Game) editorApply() {
	e := g.editor
	if e.draft == nil || e.selected < 0 || e.selected >= len(e.doc.Stage.Enemies) {
		return
	}
	if err := g.S.Data.ValidateSpawn(*e.draft); err != nil {
		e.message = err.Error()
		return
	}
	v := g.editorPreview(*e.draft)
	if e.draft.HP != nil && *e.draft.HP > v.Stats.MaxHP || e.draft.MP != nil && *e.draft.MP > v.Stats.MaxMP {
		e.message = "HP/MP 최대값을 초과했습니다"
		return
	}
	e.history.Push(e.doc)
	e.doc.Stage.Enemies[e.selected] = mapedit.Clone(*e.draft)
	e.dirty = true
	e.message = "배치 설정 적용됨"
}
func (g *Game) drawEditor(dst *ebiten.Image) {
	e := g.editor
	rect(dst, 0, 0, Width, Height, color.NRGBA{R: 22, G: 27, B: 31, A: 255})
	if e.doc == nil {
		g.label(dst, "전투맵 선택", 64, 44, 30, gold)
		for i, st := range g.S.Data.Stages {
			i, st := i, st
			g.btn(dst, image.Rect(64, 108+i*64, 680, 156+i*64), fmt.Sprintf("%s · %s · %d×%d", st.ID, st.Name, st.Width, st.Height), func() { g.editorOpen(i) })
		}
		g.btn(dst, image.Rect(1060, 20, 1250, 54), "시작 메뉴", func() { g.editor = nil })
		g.label(dst, e.message, 64, 680, 18, bad)
		return
	}
	d := e.doc
	title := d.Stage.Name
	if e.dirty {
		title += " *"
	}
	g.label(dst, title, 20, 18, 22, gold)
	for i, b := range []struct {
		name string
		fn   func()
	}{{"저장", func() { g.editorSave(false) }}, {"다른 이름", func() { g.editorSave(true) }}, {"시험 전투", g.editorTrial}, {"영역 맞춤", func() { g.editorRebuild(true) }}, {"맵 목록", func() {
		if e.dirty {
			e.popup = "maps"
		} else {
			e.doc = nil
		}
	}}, {"나가기", g.editorLeave}} {
		g.btn(dst, image.Rect(490+i*126, 14, 608+i*126, 48), b.name, b.fn)
	}
	if e.field != nil {
		units := g.editorUnits()
		e.field.prepareRender(e.zoom, e.cx-float64(editorViewport.Dx())/2/e.zoom, e.cy-float64(editorViewport.Dy())/2/e.zoom, editorViewport.Dx(), editorViewport.Dy())
		e.field.DrawBounds = image.Rect(int(e.cx-float64(editorViewport.Dx())/2/e.zoom)-64, int(e.cy-float64(editorViewport.Dy())/2/e.zoom)-96, int(e.cx+float64(editorViewport.Dx())/2/e.zoom)+64, int(e.cy+float64(editorViewport.Dy())/2/e.zoom)+96)
		e.field.Draw(g.clockMS, units, nil)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-e.cx, -e.cy)
		op.GeoM.Scale(e.zoom, e.zoom)
		op.GeoM.Translate(float64(editorViewport.Min.X+editorViewport.Dx()/2), float64(editorViewport.Min.Y+editorViewport.Dy()/2))
		sub := dst.SubImage(image.Rect(int(float64(editorViewport.Min.X)*scale), int(float64(editorViewport.Min.Y)*scale), int(float64(editorViewport.Max.X)*scale), int(float64(editorViewport.Max.Y)*scale))).(*ebiten.Image)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(editorViewport.Min.X), float64(editorViewport.Min.Y))
		drawScaled(sub, e.field.Canvas, op)
		g.editorGrid(dst)
	}
	window(dst, image.Rect(1004, 64, 1266, 688))
	g.editorDetail(dst)
	g.editorToolbar(dst)
	g.label(dst, e.message, 24, Height-29, 15, bad)
	if e.popup != "" {
		g.editorPopup(dst)
	}
	if e.editing != "" {
		g.buttons = nil
		rect(dst, 0, 0, Width, Height, color.NRGBA{A: 160})
		g.titled(dst, image.Rect(340, 290, 940, 440), "입력 · Enter 적용 / Esc 취소")
		g.label(dst, e.entry+"_", 370, 345, 23, ink)
	}
}
func (g *Game) editorUnits() []*unitVis {
	e := g.editor
	keyBytes, _ := json.Marshal(e.doc.Stage)
	key := string(keyBytes) + g.S.Data.Hash
	if key == e.unitsKey {
		return e.units
	}
	units := []*unitVis{}
	for i, sp := range append(append([]content.Spawn{}, e.doc.Stage.Enemies...), e.doc.Stage.Party...) {
		if g.S.Data.ValidateSpawn(sp) != nil {
			continue
		}
		v := core.SpawnPreview(g.S.Data, sp)
		if i < len(e.doc.Stage.Enemies) {
			v = g.editorPreview(sp)
		}
		art := g.battleClassArt(v.Def, v.Class, v.Color)
		if art == nil {
			continue
		}
		dir := sp.Facing
		if dir == "" {
			dir = "NW"
		}
		units = append(units, &unitVis{id: fmt.Sprint(i), ally: i >= len(e.doc.Stage.Enemies), art: art, u: float64(sp.X - e.doc.Play.X), v: float64(sp.Y - e.doc.Play.Y), lift: e.field.Lift(sp.X-e.doc.Play.X, sp.Y-e.doc.Play.Y), dir: dir, anim: "idle", dying: -1, shownHP: v.HP, maxHP: v.Stats.MaxHP, shownMP: v.MP, maxMP: v.Stats.MaxMP, top: art.TopOffset(), inactive: true})
	}
	e.units, e.unitsKey = units, key
	return units
}
func (g *Game) editorDiamond(dst *ebiten.Image, u, v, w, h float64, c color.Color) {
	e := g.editor
	x, y := e.field.Center(u-float64(e.doc.Play.X), v-float64(e.doc.Play.Y))
	x, y = e.screen(x, y)
	points := [][2]float64{{x, y - h*e.zoom}, {x + w*e.zoom, y}, {x, y + h*e.zoom}, {x - w*e.zoom, y}}
	for i, a := range points {
		b := points[(i+1)%4]
		strokeLine(dst, float32(a[0]), float32(a[1]), float32(b[0]), float32(b[1]), 1.5, c)
	}
}
func (g *Game) editorGrid(dst *ebiten.Image) {
	e := g.editor
	for _, p := range e.doc.Stage.Allies {
		g.editorDiamond(dst, float64(p.X), float64(p.Y), 32, 16, good)
	}
	r := e.doc.Play
	if e.resizing {
		r = e.proposed
	}
	for i := 0; i < 4; i++ {
		x, y := e.cornerAt(r, i)
		nx, ny := e.cornerAt(r, (i+1)%4)
		strokeLine(dst, float32(x), float32(y), float32(nx), float32(ny), 2, gold)
		rect(dst, float32(x-5), float32(y-5), 10, 10, gold)
	}
	x, y := cursorPos()
	if !image.Pt(x, y).In(editorViewport) {
		return
	}
	p := e.smallAt(x, y)
	u, v := mapedit.FloorHalf(p.X), mapedit.FloorHalf(p.Y)
	g.editorDiamond(dst, float64(u), float64(v), 32, 16, mpC)
	g.editorDiamond(dst, (float64(p.X)+.5)/2-.5, (float64(p.Y)+.5)/2-.5, 16, 8, gold)
	if e.tab == 0 && e.brushSize > 1 {
		n := e.brushSize
		sx, sy := p.X-(n-1)/2, p.Y-(n-1)/2
		g.editorDiamond(dst, (float64(sx)+float64(n)/2)/2-.5, (float64(sy)+float64(n)/2)/2-.5, float64(16*n), float64(8*n), good)
	}
	g.outlined(dst, g.S.Data.Tiles[e.doc.Small(p.X, p.Y)].Name+"("+g.S.Data.Tiles[e.doc.Tile(u, v)].Name+")", float64(x+18), float64(y+14), 15)
}

func (g *Game) editorPreview(sp content.Spawn) core.UnitView {
	e := g.editor
	if sp.Faction == "" {
		sp.Faction = e.doc.Stage.Faction
	}
	b, _ := json.Marshal(sp)
	key := string(b) + g.S.Data.Hash
	if e.previews == nil {
		e.previews = map[string]core.UnitView{}
	}
	if v, ok := e.previews[key]; ok {
		return v
	}
	v := core.SpawnPreview(g.S.Data, sp)
	if len(e.previews) > 256 {
		e.previews = map[string]core.UnitView{}
	}
	e.previews[key] = v
	return v
}

func (e *mapEditor) clampCamera() {
	size := e.field.Size()
	e.cx = max(0, min(float64(size.X), e.cx))
	e.cy = max(0, min(float64(size.Y), e.cy))
}

func editorEdgeScroll(p image.Point, r image.Rectangle) (int, int) {
	if !p.In(r) {
		return 0, 0
	}
	x, y := 0, 0
	if p.X < r.Min.X+8 {
		x = -1
	} else if p.X >= r.Max.X-8 {
		x = 1
	}
	if p.Y < r.Min.Y+8 {
		y = -1
	} else if p.Y >= r.Max.Y-8 {
		y = 1
	}
	return x, y
}
