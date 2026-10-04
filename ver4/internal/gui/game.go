// Package gui renders the same Session used by the CLI. All mutations cross
// Session.Handle with a revision; the renderer only consumes copied DTOs.
// The battlefield is the ver3 isometric map; every outcome is shown by claim replay
// (design.md 0.9): AI phases are resolved first and replayed with independent actions
// overlapping, a player's command is a one-action replay. Input follows 조조전:
// a unit moves tentatively, then an action menu opens beside it (command.go).
package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"srpg/internal/ai"
	"srpg/internal/cli"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
	"srpg/internal/session"
)

const Width, Height = 1280, 850

type Game struct {
	assets               *Assets
	field                *Field
	vis                  map[string]*unitVis
	popups               []*popup
	effects              []*effect
	player               Player
	banner               string
	bannerT, clockMS     float64
	camX, camY           float64
	zoom, speed          int
	dragging, dragMoved  bool
	dragX, dragY         int
	shownNode            string
	lineIndex, lineStart int
	Verify               bool
	Shots                []int // capture the screen at these ticks while autoplaying, then quit
	verified             map[string]bool
	verifyDone           int
	S                    *session.Session
	font                 *text.GoTextFaceSource
	buttons              []button
	blocks               []image.Rectangle
	ord                  order
	hover                hoverCell
	threatAll            bool
	threats              map[string][]content.Point
	diffs                map[string]string
	question             string
	answer               func()
	toasts               []*toast
	history              []string
	modals               []*modal
	battleXP, startLevel map[string]int
	slotInfo             map[string]string
	selected, overlay    string
	notice, entry        string
	page, slot, ticks    int
	input, autoplay      bool
	capture              bool
	seed                 uint64
	log                  *os.File
	auditDir             string
}

func New(s *session.Session, fontPath string, seed uint64, auditDir string) (*Game, error) {
	f, err := os.Open(fontPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	source, err := text.NewGoTextFaceSource(f)
	if err != nil {
		return nil, err
	}
	g := &Game{S: s, font: source, seed: seed, verified: map[string]bool{}, slot: 1, auditDir: auditDir,
		vis: map[string]*unitVis{}, zoom: 2, speed: 1, threats: map[string][]content.Point{}, diffs: map[string]string{},
		battleXP: map[string]int{}, startLevel: map[string]int{}}
	g.player.g = g
	g.assets, err = LoadAssets(filepath.Join(filepath.Dir(fontPath), "..", "graphics"))
	if err != nil {
		return nil, err
	}
	if auditDir != "" {
		if err = os.MkdirAll(auditDir, 0755); err != nil {
			return nil, err
		}
		g.log, err = os.Create(filepath.Join(auditDir, "gui-input.jsonl"))
		if err != nil {
			return nil, err
		}
	}
	return g, nil
}
func (g *Game) Close() {
	if g.log != nil {
		g.log.Close()
	}
}

// send passes one request to the session: logging, confirmations, autosave and the
// battle log. It does not replay anything.
func (g *Game) send(q session.Request) session.Response {
	var before *core.Observation
	if g.S.Engine != nil {
		o := g.S.Engine.Observe()
		before = &o
		q.Revision = &o.Revision
	}
	r := g.S.Handle(q)
	if g.log != nil {
		_ = json.NewEncoder(g.log).Encode(struct {
			Time     string
			Request  session.Request
			Response session.Response
		}{time.Now().UTC().Format(time.RFC3339Nano), q, r})
	}
	if !r.OK {
		switch r.Error {
		case "OverwriteRequired":
			g.ask("기존 저장을 덮어쓰시겠습니까?", func() { q.Overwrite = true; g.request(q) })
		case "DiscardRequired":
			g.ask("저장하지 않은 진행을 버리시겠습니까?", func() { q.Discard = true; g.request(q) })
		default:
			g.notice = r.Error
			g.toast(koreanError(r.Error), bad)
		}
		return r
	}
	switch q.Op {
	case "load", "retry", "new", "title":
		g.player.Skip()
		g.vis = map[string]*unitVis{}
		g.popups, g.effects, g.modals = nil, nil, nil
		g.clearOrder()
		g.selected = ""
		g.field = nil
	}
	if g.S.Engine != nil && (q.Op == "command" || q.Op == "ai" || q.Op == "retry" || q.Op == "new") {
		if err := g.S.Store.Save("auto", g.S.Engine.Snapshot(), true); err != nil {
			g.toast("자동 저장 실패: "+err.Error(), bad)
		}
	}
	g.notice = "완료"
	switch q.Op {
	case "save":
		g.toast(fmt.Sprintf("슬롯 %s 저장 완료", q.Slot), gold)
	case "load":
		g.toast("불러오기 완료", gold)
	}
	g.record(before, r.Events)
	return r
}

// request is send plus the replay of a battle command.
func (g *Game) request(q session.Request) session.Response {
	var before *core.Observation
	aiCommand := core.Command{}
	if g.S.Engine != nil {
		o := g.S.Engine.Observe()
		before = &o
		if q.Op == "ai" && o.Phase == "battle" {
			aiCommand, _ = ai.Next(g.S.Engine) // the command the session's AI step will apply
		}
	}
	r := g.send(q)
	if r.OK && before != nil && before.Phase == "battle" && (q.Op == "command" || q.Op == "ai") {
		c := q.Command
		if q.Op == "ai" {
			c = aiCommand
		}
		g.replayOne(*before, c, r.Events)
	}
	return r
}

// record turns events into log lines, announcements, level-up windows and battle XP.
func (g *Game) record(before *core.Observation, events []core.Event) {
	levels := map[string][2]int{}
	order := []string{}
	for _, e := range events {
		line, important := g.eventText(e)
		if e.Kind == "experience" && g.battleXP != nil {
			g.battleXP[e.Actor] += e.Amount
		}
		if e.Kind == "level" {
			l, ok := levels[e.Actor]
			if !ok {
				l[0] = e.Amount - 1
				order = append(order, e.Actor)
			}
			l[1] = e.Amount
			levels[e.Actor] = l
		}
		if line == "" || !important && !g.S.Settings.Detail {
			continue
		}
		g.history = append(g.history, line)
		g.notice = line
		if important && e.Kind != "level" {
			g.toast(line, gold)
		}
	}
	if len(g.history) > 50 {
		g.history = g.history[len(g.history)-50:]
	}
	for _, id := range order {
		l := levels[id]
		up := levelUp{id: id, from: l[0], to: l[1]}
		if after, err := g.S.Engine.OfficerStats(id); err == nil {
			up.before, up.after = after, after
		}
		if g.S.Engine != nil {
			o := g.S.Engine.Observe()
			if u := g.unitView(o, id); u != nil {
				up.after, up.progress = u.Stats, u.XPProgress
			}
			for _, of := range o.Officers {
				if of.ID == id && up.progress == 0 {
					up.progress = core.ExperienceProgress(of.Level, of.XP)
				}
			}
		}
		if before != nil {
			if u := g.unitView(*before, id); u != nil {
				up.before = u.Stats
			}
		}
		g.modals = append(g.modals, &modal{kind: "levelup", level: up})
	}
}

// replayOne shows a single applied command (the player's, or one AI step) as a one-action replay.
func (g *Game) replayOne(before core.Observation, c core.Command, events []core.Event) {
	a := replay.New(before, c.Actor)
	a.Add(c, events)
	actions := []*replay.Action{a}
	replay.Link(actions)
	g.player.Play(actions, g.S.Engine.Observe())
}

// resolvePhase runs the current faction's whole phase with the AI and replays it.
func (g *Game) resolvePhase() {
	g.clearOrder()
	before := g.S.Engine.Observe()
	actions, err := g.S.ResolvePhase()
	if err != nil {
		g.toast(err.Error(), bad)
	}
	if g.log != nil {
		_ = json.NewEncoder(g.log).Encode(struct {
			Time    string
			Op      string
			Actions []*replay.Action
			Error   string `json:",omitempty"`
		}{time.Now().UTC().Format(time.RFC3339Nano), "resolve-phase", actions, errorText(err)})
	}
	events := []core.Event{}
	for _, a := range actions {
		for _, s := range a.Steps {
			events = append(events, s.Events...)
		}
	}
	g.record(&before, events)
	g.player.Play(actions, g.S.Engine.Observe())
}

func (g *Game) command(c core.Command) {
	if g.busy() {
		return
	}
	g.request(session.Request{Op: "command", Command: c})
	g.page = 0
}

// assist plays one recommended command (Space, autoplay and the audit runs).
func (g *Game) assist() {
	if g.busy() || g.S.Engine == nil {
		return
	}
	g.clearOrder()
	o := g.S.Engine.Observe()
	if o.Phase == "scenario" {
		switch o.Dialogue.Kind {
		case "dialogue":
			g.command(core.Command{Kind: "next"})
		case "choice":
			for _, key := range sorted(o.Dialogue.Choices) {
				g.command(core.Command{Kind: "choose", Option: key})
				break
			}
		case "preparation":
			if o.Warehouse["item_024"] > 0 {
				g.command(core.Command{Kind: "equip", Actor: "유비", Item: "item_024"})
			} else {
				g.command(core.Command{Kind: "start"})
			}
		}
		return
	}
	if o.Phase == "result" {
		if o.Result == "victory" {
			g.command(core.Command{Kind: "continue"})
		} else {
			g.autoplay = false
		}
		return
	}
	if o.Phase != "battle" {
		g.autoplay = false
		return
	}
	if o.Turn == "enemy" || g.autoplay {
		g.resolvePhase()
		return
	}
	for _, u := range o.UnitViews {
		if u.Faction == "ally" && u.HP > 0 && !u.Done {
			c, err := ai.Choose(g.S.Engine, u.ID)
			if err != nil {
				g.toast(err.Error(), bad)
				return
			}
			g.command(c)
			return
		}
	}
	g.command(core.Command{Kind: "end"})
}
func sorted[V any](m map[string]V) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// rightClick cancels the current window or order step, else clears the selection.
func (g *Game) rightClick() {
	switch {
	case g.overlay != "":
		g.overlay = ""
	case g.cancel():
	default:
		g.selected = ""
	}
}

func (g *Game) Update() error {
	g.ticks++
	g.advance(1.0 / 60)
	if g.Verify {
		if err := g.verifyFrame(); err != nil {
			if g.verifyDone > 0 {
				return ebiten.Termination
			}
			return err
		}
		g.modals = nil
		return nil
	}
	if g.S.Screen == "quit" {
		return ebiten.Termination
	}
	if len(g.Shots) > 0 {
		if g.S.Engine == nil {
			g.request(session.Request{Op: "new", Seed: g.seed})
		}
		g.autoplay, g.speed = true, 4
		if g.ticks > g.Shots[len(g.Shots)-1]+1 {
			return ebiten.Termination
		}
		for _, t := range g.Shots {
			g.capture = g.capture || g.ticks == t
		}
	}
	if g.input {
		g.cliInput()
		return nil
	}
	g.updateHover()
	g.keys()
	g.camera()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.leftClick()
	}
	playing := g.S.Screen == "playing" && g.S.Engine != nil
	if g.autoplay && len(g.modals) > 0 && g.modals[0].t > 1.2 {
		g.closeModal()
	}
	if playing && g.overlay == "" && len(g.modals) == 0 && g.ticks%12 == 0 {
		if g.autoplay {
			g.assist()
		} else {
			o := g.S.Engine.Observe()
			if o.Phase == "battle" && o.Turn == "enemy" && g.S.Settings.AI == "auto" && !g.busy() {
				g.resolvePhase()
			}
		}
	}
	return nil
}

// cliInput is the debug command line (backquote): CLI syntax, one command.
func (g *Game) cliInput() {
	g.entry = string(ebiten.AppendInputChars([]rune(g.entry)))
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if r := []rune(g.entry); len(r) > 0 {
			g.entry = string(r[:len(r)-1])
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		c, err := cli.Parse(g.entry)
		if err != nil {
			g.toast(err.Error(), bad)
		} else {
			g.clearOrder()
			g.command(c)
			g.entry, g.input = "", false
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.input = false
	}
}

func (g *Game) keys() {
	pressed := inpututil.IsKeyJustPressed
	if pressed(ebiten.KeyF12) {
		g.capture = true
	}
	if pressed(ebiten.KeyF5) && g.S.Engine != nil {
		g.request(session.Request{Op: "save", Slot: fmt.Sprint(g.slot)})
	}
	if pressed(ebiten.KeyF9) {
		g.request(session.Request{Op: "load", Slot: fmt.Sprint(g.slot)})
	}
	if pressed(ebiten.KeyEscape) {
		switch {
		case g.overlay != "":
			g.overlay = ""
		case len(g.modals) > 0:
			g.closeModal()
		case g.cancel():
		default:
			g.openOverlay("menu")
		}
		return
	}
	if g.overlay != "" || g.S.Screen != "playing" || g.S.Engine == nil {
		return
	}
	if len(g.modals) > 0 {
		if pressed(ebiten.KeyEnter) || pressed(ebiten.KeySpace) {
			g.closeModal()
		}
		return
	}
	o := g.S.Engine.Observe()
	dialogue := o.Phase == "scenario" && o.Dialogue.Kind == "dialogue"
	if pressed(ebiten.KeyGraveAccent) {
		g.input, g.entry = true, ""
	}
	if pressed(ebiten.KeyP) {
		g.autoplay = !g.autoplay
	}
	if pressed(ebiten.KeySpace) {
		if dialogue {
			g.advanceDialogue()
		} else {
			g.assist()
		}
	}
	if pressed(ebiten.KeyEnter) {
		if dialogue {
			g.advanceDialogue()
		}
		g.player.Skip()
	}
	if pressed(ebiten.KeyF) {
		g.speed = map[int]int{1: 2, 2: 4, 4: 1}[g.speed]
		g.toast(fmt.Sprintf("재생 속도 ×%d", g.speed), gold)
	}
	if o.Phase != "battle" {
		return
	}
	if pressed(ebiten.KeyU) {
		g.openOverlay("units")
	}
	if pressed(ebiten.KeyL) {
		g.openOverlay("log")
	}
	if pressed(ebiten.KeyT) {
		g.threatAll = !g.threatAll
	}
	if pressed(ebiten.KeyE) && o.Turn == "ally" && !g.busy() {
		g.endPhase()
	}
	if pressed(ebiten.KeyTab) {
		ids := []string{}
		for _, u := range o.UnitViews {
			if u.Faction == "ally" && u.HP > 0 && !u.Done {
				ids = append(ids, u.ID)
			}
		}
		if len(ids) > 0 {
			next := ids[0]
			for i, id := range ids {
				if id == g.selected {
					next = ids[(i+1)%len(ids)]
				}
			}
			g.selected = next
			if u := g.unitView(o, next); u != nil && !g.busy() {
				g.clearOrder()
				if o.Turn == "ally" {
					g.begin(*u)
				}
				g.focus(u.X, u.Y)
			}
		}
	}
}

func (g *Game) leftClick() {
	x, y := ebiten.CursorPosition()
	for i := len(g.buttons) - 1; i >= 0; i-- {
		if b := g.buttons[i]; image.Pt(x, y).In(b.rect) {
			b.click()
			return
		}
	}
	if g.overlay != "" || len(g.modals) > 0 || g.S.Screen != "playing" || g.S.Engine == nil || g.field == nil || g.busy() {
		return
	}
	if o := g.S.Engine.Observe(); o.Phase != "battle" {
		return
	}
	if g.hover.ok {
		g.mapClick(g.hover.x, g.hover.y)
	} else if !g.cancel() {
		g.selected = ""
	}
}

func (g *Game) Layout(_, _ int) (int, int) { return Width, Height }

func (g *Game) Draw(dst *ebiten.Image) {
	dst.Fill(color.NRGBA{R: 16, G: 13, B: 12, A: 255})
	g.buttons, g.blocks = nil, nil
	if g.S.Screen == "title" || g.S.Engine == nil {
		g.title(dst)
	} else {
		g.play(dst)
	}
	if e := g.player.Duel; e != nil {
		r := image.Rect(260, 250, 1020, 520)
		g.titled(dst, r, "一騎討")
		g.portrait(dst, e.Actor, 296, 290, 140)
		g.portrait(dst, e.Target, 844, 290, 140)
		g.centered(dst, g.name(e.Actor)+"  VS  "+g.name(e.Target), 640, 300, 24, gold)
		g.label(dst, wrap(e.Text, 20), 470, 350, 17, ink)
	}
	if g.overlay != "" {
		g.drawOverlay(dst)
	} else if len(g.modals) > 0 && !g.busy() && g.S.Engine != nil {
		g.drawModal(dst)
	}
	if g.input {
		rect(dst, 0, Height-40, Width, 40, color.NRGBA{A: 220})
		g.label(dst, "> "+g.entry+"_", 16, Height-34, 18, gold)
	}
	if g.capture {
		g.capture = false
		g.screenshot(dst)
	}
}

func (g *Game) screenshot(dst *ebiten.Image) {
	dir := g.auditDir
	if dir == "" {
		dir = "reports/gui"
	}
	_ = os.MkdirAll(dir, 0755)
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	dst.ReadPixels(img.Pix)
	file, err := os.Create(filepath.Join(dir, fmt.Sprintf("screen-%06d.png", g.ticks)))
	if err == nil {
		err = png.Encode(file, img)
		file.Close()
	}
	if err != nil {
		g.toast(err.Error(), bad)
	}
}

func (g *Game) title(dst *ebiten.Image) {
	for i := 0; i < Height; i += 10 {
		k := float64(i) / Height
		rect(dst, 0, float32(i), Width, 10, color.NRGBA{R: uint8(48 - 30*k), G: uint8(30 - 18*k), B: uint8(24 - 12*k), A: 255})
	}
	g.label(dst, "桃園結義", 96, 70, 64, gold)
	g.label(dst, "삼국지 SRPG", 104, 160, 26, ink)
	g.portrait(dst, "유비", 650, 205, 170)
	g.portrait(dst, "관우", 840, 255, 170)
	g.portrait(dst, "장비", 1030, 205, 170)
	g.label(dst, "도원결의 → 황건적의 난 → 사수관 → 호로관", 104, 214, 16, muted)
	r := image.Rect(96, 290, 420, 560)
	window(dst, r)
	items := []menuItem{
		{"새로 시작", func() { g.request(session.Request{Op: "new", Seed: g.seed}); g.overlay = "" }},
		{"불러오기", func() { g.openOverlay("load") }},
		{"설정", func() { g.openOverlay("settings") }},
		{"종료", func() { g.request(session.Request{Op: "quit"}) }},
	}
	for i, it := range items {
		g.btn(dst, image.Rect(r.Min.X+28, r.Min.Y+30+i*56, r.Max.X-28, r.Min.Y+30+i*56+44), it.label, it.click)
	}
}

func (g *Game) play(dst *ebiten.Image) {
	o := g.S.Engine.Observe()
	switch o.Phase {
	case "complete":
		g.complete(dst, o)
	case "scenario":
		g.scenario(dst, o)
		g.btn(dst, image.Rect(1176, 12, 1268, 42), "메뉴", func() { g.openOverlay("menu") })
	default:
		g.battle(dst, o)
		if o.Phase == "result" && !g.busy() {
			g.result(dst, o)
		}
	}
}

func (g *Game) complete(dst *ebiten.Image, o core.Observation) {
	g.backdrop(dst, o)
	r := image.Rect(Width/2-320, 180, Width/2+320, 620)
	g.titled(dst, r, "완료")
	g.centered(dst, "호로관 전투 완료", float64(Width/2), 220, 34, gold)
	g.centered(dst, "세 번의 싸움을 마쳤습니다.", float64(Width/2), 276, 19, ink)
	y := 336
	for _, of := range o.Officers {
		if !isParty(of.ID) {
			continue
		}
		g.portrait(dst, of.ID, float64(r.Min.X+80), float64(y), 48)
		g.label(dst, fmt.Sprintf("%s   Lv%d   %s", g.name(of.ID), of.Level, of.Class), float64(r.Min.X+148), float64(y+12), 19, ink)
		y += 62
	}
	g.autoplay = false
	g.btn(dst, image.Rect(r.Max.X-220, r.Max.Y-60, r.Max.X-32, r.Max.Y-24), "시작 메뉴", func() { g.request(session.Request{Op: "title"}) })
}

// battle draws the field full-screen with the HUD over it.
func (g *Game) battle(dst *ebiten.Image, o core.Observation) {
	g.ensureField(o)
	units := make([]*unitVis, 0, len(g.vis))
	for _, u := range o.UnitViews {
		if v, ok := g.vis[u.ID]; ok {
			units = append(units, v)
		}
	}
	g.field.Draw(g.clockMS, units, g.marks(o))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-g.camX, -g.camY)
	op.GeoM.Scale(float64(g.zoom), float64(g.zoom))
	op.GeoM.Translate(float64(viewport.Min.X+viewport.Dx()/2), float64(viewport.Min.Y+viewport.Dy()/2))
	dst.DrawImage(g.field.Canvas, op)
	g.cursor(dst)
	g.drawEffects(dst)
	for _, p := range g.popups {
		x, y := g.field.Center(p.unit.u, p.unit.v)
		sx, sy := g.toScreen(x+p.unit.shakeOffset(), y-p.unit.lift-p.unit.top-10-p.t*16)
		g.outlined(dst, p.text, sx-g.width(p.text, 16)/2, sy, 16, p.col)
	}
	if g.bannerT > 0 {
		w := g.width(g.banner, 30)
		rect(dst, float32(Width/2-w/2-30), 300, float32(w+60), 58, color.NRGBA{R: 40, G: 14, B: 10, A: 200})
		rect(dst, float32(Width/2-w/2-30), 300, float32(w+60), 2, gold)
		rect(dst, float32(Width/2-w/2-30), 356, float32(w+60), 2, gold)
		g.label(dst, g.banner, float64(Width/2)-w/2, 308, 30, gold)
	}
	g.topBar(dst, o)
	g.minimap(dst, o)
	g.unitCard(dst, o)
	g.terrainPanel(dst, o)
	if o.Phase == "battle" {
		g.dock(dst, o)
	}
	g.drawMenu(dst)
	g.forecast(dst)
	g.messages(dst)
}
