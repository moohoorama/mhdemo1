// Package gui renders the same Session used by the CLI. All mutations cross
// Session.Handle with a revision; the renderer only consumes copied DTOs.
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
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/ai"
	"srpg/internal/cli"
	"srpg/internal/core"
	"srpg/internal/session"
)

const Width, Height = 1280, 850

type button struct {
	rect  image.Rectangle
	label string
	click func()
}
type Game struct {
	art                                    art
	motion                                 motion
	shownNode                              string
	nodeStart                              int
	Verify                                 bool
	verified                               map[string]bool
	verifyDone                             int
	confirmation                           func()
	S                                      *session.Session
	font                                   *text.GoTextFaceSource
	buttons                                []button
	selected, mode, overlay, notice, entry string
	page, slot, ticks                      int
	input, autoplay, capture               bool
	seed                                   uint64
	log                                    *os.File
	auditDir                               string
	last                                   []string
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
	g := &Game{S: s, font: source, seed: seed, verified: map[string]bool{}, slot: 1, auditDir: auditDir}
	g.art, err = loadArt(fontPath)
	if err != nil {
		return nil, err
	}
	g.verified = map[string]bool{}
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
func (g *Game) request(q session.Request) {
	if g.S.Engine != nil {
		rev := g.S.Engine.Observe().Revision
		q.Revision = &rev
	}
	var before *core.Observation
	if g.S.Engine != nil {
		o := g.S.Engine.Observe()
		before = &o
	}
	r := g.S.Handle(q)
	if r.OK {
		if q.Op == "load" || q.Op == "retry" || q.Op == "new" {
			g.motion = motion{}
			g.mode = ""
			g.selected = ""
		}
		g.beginMotion(before, r.Events)
	}
	if g.log != nil {
		_ = json.NewEncoder(g.log).Encode(struct {
			Time     string
			Request  session.Request
			Response session.Response
		}{time.Now().UTC().Format(time.RFC3339Nano), q, r})
	}
	if !r.OK {
		g.notice = r.Error
		if r.Error == "OverwriteRequired" {
			g.confirm("기존 저장을 덮어쓰시겠습니까?", func() { q.Overwrite = true; g.request(q) })
		}
		if r.Error == "DiscardRequired" {
			g.confirm("저장하지 않은 진행을 버리시겠습니까?", func() { q.Discard = true; g.request(q) })
		}
		return
	}
	if g.S.Engine != nil && (q.Op == "command" || q.Op == "ai" || q.Op == "retry" || q.Op == "new") {
		if err := g.S.Store.Save("auto", g.S.Engine.Snapshot(), true); err != nil {
			g.notice = "자동 저장 실패: " + err.Error()
			return
		}
	}
	g.notice = "완료"
	if q.Op == "save" {
		g.notice = fmt.Sprintf("슬롯 %s 저장 완료", q.Slot)
	}
	if q.Op == "load" {
		g.notice = fmt.Sprintf("슬롯 %s 불러오기 완료", q.Slot)
	}
	for _, v := range r.Events {
		if !g.S.Settings.Detail && v.Kind != "level" && v.Kind != "experience" && v.Kind != "duel" && v.Kind != "retreat" && v.Kind != "death" && v.Kind != "victory" {
			continue
		}
		line := fmt.Sprintf("%s %s → %s %d", v.Kind, v.Actor, v.Target, v.Amount)
		if v.Text != "" {
			line = v.Text
		}
		g.last = append(g.last, line)
	}
	if len(g.last) > 6 {
		g.last = g.last[len(g.last)-6:]
	}
	if len(r.Events) > 0 && len(g.last) > 0 {
		g.notice = g.last[len(g.last)-1]
	}
}

func (g *Game) confirm(message string, f func()) {
	g.overlay = "confirm"
	g.notice = message
	g.confirmation = f
	g.autoplay = false
}
func (g *Game) command(c core.Command) {
	if g.busy() {
		return
	}
	g.request(session.Request{Op: "command", Command: c})
	g.mode = ""
	g.page = 0
}
func (g *Game) assist() {
	if g.busy() {
		return
	}
	if g.S.Engine == nil {
		return
	}
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
	if o.Turn == "enemy" {
		g.request(session.Request{Op: "ai"})
		return
	}
	for _, u := range o.UnitViews {
		if u.Faction == "ally" && u.HP > 0 && !u.Done {
			c, err := ai.Choose(g.S.Engine, u.ID)
			if err != nil {
				g.notice = err.Error()
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
func (g *Game) Update() error {
	g.ticks++
	if g.Verify {
		if err := g.verifyFrame(); err != nil {
			if g.verifyDone > 0 {
				return ebiten.Termination
			}
			return err
		}
		return nil
	}
	if g.S.Screen == "quit" {
		return ebiten.Termination
	}
	if g.input {
		g.entry = string(ebiten.AppendInputChars([]rune(g.entry)))
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			r := []rune(g.entry)
			if len(r) > 0 {
				g.entry = string(r[:len(r)-1])
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			c, err := cli.Parse(g.entry)
			if err != nil {
				g.notice = err.Error()
			} else {
				g.command(c)
				g.entry = ""
				g.input = false
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.input = false
		}
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		g.capture = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.request(session.Request{Op: "save", Slot: fmt.Sprint(g.slot)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		g.request(session.Request{Op: "load", Slot: fmt.Sprint(g.slot)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.overlay != "" {
			g.overlay = ""
		} else {
			g.overlay = "menu"
		}
		g.autoplay = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) && g.S.Screen == "playing" && g.overlay == "" {
		g.autoplay = !g.autoplay
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.overlay == "" {
		g.assist()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) && g.S.Engine != nil {
		o := g.S.Engine.Observe()
		ids := []string{}
		for _, u := range o.UnitViews {
			if u.Faction == "ally" && u.HP > 0 {
				ids = append(ids, u.ID)
			}
		}
		for i, id := range ids {
			if id == g.selected {
				g.selected = ids[(i+1)%len(ids)]
				break
			}
		}
		if g.selected == "" && len(ids) > 0 {
			g.selected = ids[0]
		}
		g.page = 0
		g.mode = ""
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		hit := false
		for _, b := range g.buttons {
			if image.Pt(x, y).In(b.rect) {
				b.click()
				hit = true
				break
			}
		}
		if !hit && g.overlay == "" && g.S.Screen == "playing" {
			g.mapClick(x, y)
		}
	}
	if g.overlay == "" && g.S.Screen == "playing" && g.ticks%12 == 0 {
		if g.autoplay {
			g.assist()
		} else if g.S.Engine != nil {
			o := g.S.Engine.Observe()
			if o.Phase == "battle" && o.Turn == "enemy" && g.S.Settings.AI == "auto" && !g.busy() {
				g.request(session.Request{Op: "ai"})
			}
		}
	}
	return nil
}
func (g *Game) mapClick(x, y int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.Engine.Observe()
	if o.Map == nil || o.Phase != "battle" {
		return
	}
	tile := min(44, 800/o.Map.Width, 550/o.Map.Height)
	mx, my := (x-32)/tile, (y-112)/tile
	if x < 32 || y < 112 || mx >= o.Map.Width || my >= o.Map.Height {
		return
	}
	if g.mode == "move" {
		g.command(core.Command{Kind: "move", Actor: g.selected, X: mx, Y: my})
		return
	}
	for _, u := range o.UnitViews {
		if u.HP > 0 && u.X == mx && u.Y == my {
			if g.mode == "attack" {
				g.command(core.Command{Kind: "attack", Actor: g.selected, Target: u.ID})
				return
			}
			g.selected = u.ID
			g.mode = ""
			g.page = 0
			return
		}
	}
}
func (g *Game) Layout(_, _ int) (int, int) { return Width, Height }

var (
	ink   = color.NRGBA{R: 232, G: 227, B: 212, A: 255}
	muted = color.NRGBA{R: 159, G: 170, B: 175, A: 255}
	gold  = color.NRGBA{R: 226, G: 182, B: 93, A: 255}
	panel = color.NRGBA{R: 32, G: 44, B: 52, A: 255}
)

func rect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	vector.FillRect(dst, x, y, w, h, c, false)
}
func (g *Game) label(dst *ebiten.Image, s string, x, y, size float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = size * 1.45
	text.Draw(dst, s, &text.GoTextFace{Source: g.font, Size: size}, op)
}
func wrap(s string, n int) string {
	r := []rune(s)
	out := []string{}
	for len(r) > n {
		out = append(out, string(r[:n]))
		r = r[n:]
	}
	return strings.Join(append(out, string(r)), "\n")
}
func (g *Game) button(dst *ebiten.Image, label string, x, y, w int, f func()) {
	b := button{image.Rect(x, y, x+w, y+32), label, f}
	g.buttons = append(g.buttons, b)
	mx, my := ebiten.CursorPosition()
	c := panel
	if image.Pt(mx, my).In(b.rect) {
		c = color.NRGBA{R: 57, G: 74, B: 83, A: 255}
	}
	rect(dst, float32(x-1), float32(y-1), float32(w+2), 34, gold)
	rect(dst, float32(x), float32(y), float32(w), 32, c)
	rect(dst, float32(x+1), float32(y+1), float32(w-2), 1, color.NRGBA{R: 121, G: 128, B: 114, A: 255})
	rect(dst, float32(x), float32(y), 3, 32, gold)
	g.label(dst, label, float64(x+10), float64(y+5), 15, ink)
}
func (g *Game) Draw(dst *ebiten.Image) {
	dst.Fill(color.NRGBA{R: 18, G: 28, B: 34, A: 255})
	g.buttons = nil
	g.label(dst, "桃園  ·  삼국지 SRPG", 32, 22, 26, gold)
	g.label(dst, "F5 저장  ·  F9 불러오기  ·  Tab 부대  ·  Space 한 명령  ·  Esc 메뉴", 32, 61, 14, muted)
	g.button(dst, "메뉴", 1110, 24, 130, func() { g.overlay = "menu"; g.autoplay = false })
	if g.S.Screen == "title" || g.S.Engine == nil {
		g.title(dst)
	} else {
		g.play(dst)
	}
	if g.busy() {
		for _, e := range g.motion.events {
			if e.Kind == "duel" {
				rect(dst, 280, 230, 710, 245, panel)
				vector.StrokeRect(dst, 278, 228, 714, 249, 3, gold, false)
				g.portrait(dst, e.Actor, 310, 258, 135)
				g.portrait(dst, e.Target, 825, 258, 135)
				g.label(dst, "一 騎 討", 542, 267, 31, gold)
				g.label(dst, e.Actor+"  VS  "+e.Target, 510, 322, 21, ink)
				g.label(dst, wrap(e.Text, 37), 310, 410, 18, ink)
			}
		}
	}
	if g.overlay != "" {
		g.drawOverlay(dst)
	}
	rect(dst, 0, 790, 1280, 60, panel)
	g.label(dst, wrap(g.notice, 78), 28, 800, 15, gold)
	if g.capture {
		g.capture = false
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
			g.notice = err.Error()
		}
	}
}
func (g *Game) title(dst *ebiten.Image) {
	g.portrait(dst, "유비", 650, 205, 160)
	g.portrait(dst, "관우", 820, 250, 160)
	g.portrait(dst, "장비", 990, 205, 160)
	g.label(dst, "뜻을 모아, 난세를 건너다.", 120, 185, 34, ink)
	g.label(dst, "도원결의 → 황건적의 난 → 사수관 → 호로관", 120, 249, 18, muted)
	g.label(dst, "전술과 육성 · 부대별 경험치 · 병법 · 일기토", 120, 285, 18, muted)
	g.button(dst, "시작하기", 120, 355, 280, func() { g.request(session.Request{Op: "new", Seed: g.seed}); g.overlay = "" })
	g.button(dst, "불러오기", 120, 403, 280, func() { g.overlay = "load" })
	g.button(dst, "설정", 120, 451, 280, func() { g.overlay = "settings" })
	g.button(dst, "종료", 120, 499, 280, func() { g.request(session.Request{Op: "quit"}) })
}
func (g *Game) play(dst *ebiten.Image) {
	o := g.S.Engine.Observe()
	if o.Phase == "complete" {
		g.label(dst, "호로관 전투 완료", 100, 180, 36, gold)
		g.label(dst, "세 번의 싸움을 마쳤습니다.", 100, 244, 22, ink)
		y := 302
		for _, of := range o.Officers {
			if of.ID == "유비" || of.ID == "관우" || of.ID == "장비" || of.ID == "간옹" {
				g.label(dst, fmt.Sprintf("%s  Lv%d  EXP %.1f/100", of.ID, of.Level, core.ExperienceProgress(of.Level, of.XP)), 100, float64(y), 22, ink)
				y += 42
			}
		}
		g.autoplay = false
		return
	}
	if o.Map != nil {
		g.drawMap(dst, o)
	}
	if o.Phase == "scenario" {
		g.label(dst, "제 "+fmt.Sprint(o.Stage+1)+"장", 32, 110, 17, gold)
		dialogue := o.Dialogue.Text
		if g.shownNode != o.Node {
			g.shownNode = o.Node
			g.nodeStart = g.ticks
		}
		if g.S.Settings.TextDelayMS > 0 && o.Dialogue.Kind == "dialogue" {
			r := []rune(dialogue)
			n := min(len(r), (g.ticks-g.nodeStart)*1000/(60*g.S.Settings.TextDelayMS))
			dialogue = string(r[:n])
		}
		rect(dst, 29, 141, 780, 146, panel)
		vector.StrokeRect(dst, 29, 141, 780, 146, 2, gold, false)
		speaker := "유비"
		for _, name := range []string{"간옹", "관우", "장비"} {
			if strings.HasPrefix(o.Dialogue.Text, name+":") {
				speaker = name
			}
		}
		g.portrait(dst, speaker, 44, 156, 114)
		g.label(dst, wrap(dialogue, 27), 182, 159, 21, ink)
		if o.Dialogue.Kind == "preparation" {
			g.preparation(dst, o)
		} else {
			if o.Dialogue.Kind == "choice" {
				y := 310
				for _, option := range sorted(o.Dialogue.Choices) {
					key := option
					g.button(dst, key, 32, y, 330, func() { g.command(core.Command{Kind: "choose", Option: key}) })
					y += 44
				}
			} else {
				g.button(dst, "대사 진행", 32, 340, 260, func() { g.command(core.Command{Kind: "next"}) })
			}
		}
	} else if o.Phase == "result" {
		g.label(dst, map[string]string{"victory": "승리", "defeat": "패배"}[o.Result], 880, 114, 32, gold)
		if o.Result == "victory" {
			g.button(dst, "다음 이야기", 880, 166, 350, func() { g.command(core.Command{Kind: "continue"}) })
		} else {
			g.button(dst, "출전 준비부터 재도전", 880, 166, 350, func() { g.request(session.Request{Op: "retry"}) })
		}
	} else {
		rect(dst, 858, 98, 390, 506, panel)
		vector.StrokeRect(dst, 856, 96, 394, 510, 2, gold, false)
		g.battlePanel(dst, o)
	}
	g.label(dst, "전투 기록", 880, 612, 16, gold)
	for i, line := range g.last {
		g.label(dst, wrap(line, 30), 880, float64(639+i*21), 12, muted)
	}
	if g.input {
		g.label(dst, "> "+g.entry, 32, 735, 18, gold)
	}
	g.button(dst, "명령 입력", 32, 747, 150, func() { g.input = true; g.entry = "" })
	g.button(dst, "저장 / 불러오기", 195, 747, 185, func() { g.overlay = "save" })
	g.button(dst, "자동 진행 "+map[bool]string{true: "중지", false: "시작"}[g.autoplay], 395, 747, 170, func() { g.autoplay = !g.autoplay })
}
func (g *Game) drawMap(dst *ebiten.Image, o core.Observation) {
	tile := min(44, 800/o.Map.Width, 550/o.Map.Height)
	vector.StrokeRect(dst, 29, 109, float32(o.Map.Width*tile+6), float32(o.Map.Height*tile+6), 3, gold, false)
	indices := map[byte]int{'.': 0, 'd': 1, 'f': 2, 's': 3, 'c': 4, 'i': 5, 'v': 6, '~': 7}
	reachable := map[image.Point]bool{}
	if g.mode == "move" {
		for _, c := range g.S.Engine.LegalActions(g.selected).Commands {
			if c.Kind == "move" {
				reachable[image.Pt(c.X, c.Y)] = true
			}
		}
	}
	for y, row := range o.Map.Tiles {
		for x, t := range []byte(row) {
			px, py := 32+x*tile, 112+y*tile
			index := indices[t]
			blit(dst, cell(g.art.terrain, index%4, index/4, 4, 2), float64(px), float64(py), float64(tile), float64(tile), nil)
			if reachable[image.Pt(x, y)] {
				rect(dst, float32(px+3), float32(py+3), float32(tile-7), float32(tile-7), color.NRGBA{R: 65, G: 148, B: 202, A: 110})
			}

		}
	}
	for _, u := range o.UnitViews {
		if u.HP <= 0 && !(g.busy() && g.motion.before[u.ID].HP > 0) {
			continue
		}
		px, py := g.animatedPosition(u, tile)
		x, y := int(px), int(py)
		g.sprite(dst, u, px, py, float64(tile))
		if u.ID == g.selected {
			vector.StrokeRect(dst, float32(x), float32(y), float32(tile-1), float32(tile-1), 2, gold, false)
		}

		rect(dst, float32(x+2), float32(y+tile-6), float32(tile-5), 3, color.NRGBA{R: 61, G: 36, B: 34, A: 255})
		rect(dst, float32(x+2), float32(y+tile-6), float32(tile-5)*float32(u.HP)/float32(u.Stats.MaxHP), 3, color.NRGBA{R: 117, G: 192, B: 117, A: 255})
	}
	g.projectile(dst, tile)
	g.effects(dst, tile)
	g.miniMap(dst, o)
	g.label(dst, fmt.Sprintf("%s  ·  %d턴  ·  %s 진영", o.Map.Name, o.Round, map[string]string{"ally": "아군", "enemy": "적군"}[o.Turn]), 32, 680, 20, gold)
	g.label(dst, "클릭: 부대 선택  ·  공격 메뉴에서 적 클릭: 공격", 32, 715, 14, muted)
}
func (g *Game) battlePanel(dst *ebiten.Image, o core.Observation) {
	var selected *core.UnitView
	for i := range o.UnitViews {
		u := &o.UnitViews[i]
		if u.ID == g.selected {
			selected = u
		}
	}
	if selected == nil {
		for i := range o.UnitViews {
			if o.UnitViews[i].Faction == "ally" && o.UnitViews[i].HP > 0 {
				selected = &o.UnitViews[i]
				g.selected = selected.ID
				break
			}
		}
	}
	if selected == nil {
		return
	}
	u := selected
	g.portrait(dst, u.ID, 880, 104, 88)
	g.label(dst, u.Name, 986, 105, 25, gold)
	g.label(dst, fmt.Sprintf("%s  (%d,%d)", u.Class, u.X, u.Y), 986, 142, 16, ink)
	g.label(dst, fmt.Sprintf("Lv%d  ·  EXP %.1f / 100", u.Level, u.XPProgress), 986, 173, 14, gold)
	rect(dst, 880, 288, 350, 7, panel)
	rect(dst, 880, 288, float32(3.5*u.XPProgress), 7, gold)
	g.label(dst, fmt.Sprintf("병력  %d / %d    병법치  %d / %d", u.HP, u.Stats.MaxHP, u.MP, u.Stats.MaxMP), 880, 200, 15, ink)
	rect(dst, 880, 226, 168, 5, color.NRGBA{R: 20, G: 26, B: 22, A: 255})
	rect(dst, 880, 226, 168*float32(u.HP)/float32(u.Stats.MaxHP), 5, color.NRGBA{R: 100, G: 174, B: 102, A: 255})
	rect(dst, 1064, 226, 166, 5, color.NRGBA{R: 20, G: 26, B: 32, A: 255})
	rect(dst, 1064, 226, 166*float32(u.MP)/float32(max(1, u.Stats.MaxMP)), 5, color.NRGBA{R: 88, G: 157, B: 205, A: 255})
	g.label(dst, fmt.Sprintf("공격 %d   방어 %d   정신 %d", u.Stats.Attack, u.Stats.Defense, u.Stats.Mind), 880, 237, 14, muted)
	g.label(dst, wrap("특성: "+strings.Join(u.Traits, " · "), 29), 880, 255, 13, muted)
	if o.Turn == "enemy" {
		g.button(dst, "적 AI 한 명령", 880, 320, 350, func() { g.request(session.Request{Op: "ai"}) })
		return
	}

	if g.mode == "" || g.mode == "move" {
		groups := []struct{ name, kind string }{{"이동", "move"}, {"공격", "attack"}, {"병법", "skill"}, {"아이템", "item"}, {"학습", "learn"}, {"일기토", "duel"}}
		for i, group := range groups {
			v := group
			g.button(dst, v.name, 880+i%2*180, 320+i/2*42, 170, func() {
				g.mode = v.kind
				g.page = 0
				if v.kind == "move" {
					g.notice = "이동 범위를 선택하십시오."
				}
			})
		}
		g.button(dst, "대기", 880, 450, 170, func() { g.command(core.Command{Kind: "wait", Actor: g.selected}) })
		g.button(dst, "진영 종료", 1060, 450, 170, func() { g.command(core.Command{Kind: "end"}) })
		g.button(dst, "행동 추천 한 명령", 880, 499, 350, func() { g.assist() })
		g.label(dst, "부대를 선택한 뒤 행동과 대상을 선택하세요.", 880, 551, 13, muted)
	} else {
		options := []core.Command{}
		for _, c := range g.S.Engine.LegalActions(g.selected).Commands {
			if c.Kind == g.mode {
				options = append(options, c)
			}
		}
		start := g.page * 4
		if start >= len(options) {
			g.page = 0
			start = 0
		}
		g.button(dst, "← 행동 메뉴", 880, 320, 350, func() { g.mode = ""; g.page = 0 })
		for i := start; i < min(start+4, len(options)); i++ {
			c := options[i]
			g.button(dst, actionLabel(c), 880, 365+(i-start)*40, 350, func() { g.command(c) })
		}
		if len(options) == 0 {
			g.label(dst, "사용 가능한 대상이 없습니다.", 880, 370, 16, muted)
		}
		g.button(dst, "이전", 880, 542, 100, func() { g.page = max(0, g.page-1) })
		g.button(dst, "다음", 990, 542, 100, func() {
			if (g.page+1)*4 < len(options) {
				g.page++
			}
		})
		g.label(dst, fmt.Sprintf("%d / %d", g.page+1, max(1, (len(options)+3)/4)), 1110, 548, 14, muted)
	}
	options := g.S.Engine.LegalActions(g.selected).Commands
	mx, my := ebiten.CursorPosition()
	for _, b := range g.buttons {
		if image.Pt(mx, my).In(b.rect) && strings.HasPrefix(b.label, "공격") {
			for _, c := range options {
				if actionLabel(c) == b.label {
					p, err := g.S.Engine.Preview(c)
					if err == nil {
						g.notice = fmt.Sprintf("명중 %.0f%% · 최대 피해 %d · 병법치 %d", p.Hit, p.MaxDamage, p.Cost)
					}
				}
			}
		}
	}
}
func actionLabel(c core.Command) string {
	switch c.Kind {
	case "wait":
		return "대기"
	case "attack":
		return "공격 → " + c.Target
	case "skill":
		return c.Skill + " → " + c.Target
	case "item":
		return c.Item + " → " + c.Target
	case "learn":
		return "학습: " + c.Trait
	case "duel":
		return "일기토 → " + c.Target
	}
	return c.Kind
}
func (g *Game) preparation(dst *ebiten.Image, o core.Observation) {
	if g.selected == "" {
		g.selected = "유비"
	}
	y := 290
	for _, officer := range o.Officers {
		of := officer
		if !(of.ID == "유비" || of.ID == "관우" || of.ID == "장비" || of.ID == "간옹") {
			continue
		}
		label := fmt.Sprintf("%s · Lv%d · %s", of.ID, of.Level, of.Class)
		if of.Dead {
			label += " [사망]"
		}
		if of.Deputy != "" {
			label += " / 부관 " + of.Deputy
		}
		g.button(dst, label, 32, y, 480, func() { g.selected = of.ID; g.page = 0 })
		y += 44
	}
	g.label(dst, "편성 · "+g.selected, 880, 110, 24, ink)
	g.label(dst, fmt.Sprintf("출전 %d / %d", len(o.Deployment), g.S.Data.Stages[o.Stage].Limit), 880, 152, 17, gold)
	g.button(dst, "선택 장수 출전 / 제외", 880, 192, 350, func() {
		ids := []string{}
		found := false
		for _, id := range o.Deployment {
			if id == g.selected {
				found = true
			} else {
				ids = append(ids, id)
			}
		}
		if !found {
			ids = append(ids, g.selected)
		}
		g.command(core.Command{Kind: "deploy", Deployment: ids})
	})
	g.button(dst, "장비", 880, 232, 165, func() { g.overlay = "equipment"; g.page = 0 })
	g.button(dst, "부관", 1055, 232, 175, func() { g.overlay = "deputy"; g.page = 0 })
	g.button(dst, "전투 시작", 880, 290, 350, func() { g.command(core.Command{Kind: "start"}) })
	for _, of := range o.Officers {
		if of.ID == g.selected {
			g.label(dst, fmt.Sprintf("특성치 %d  ·  EXP %.1f/100", of.Points, core.ExperienceProgress(of.Level, of.XP)), 880, 350, 15, gold)
			y := 388
			for _, slot := range sorted(of.Equipment) {
				item := of.Equipment[slot]
				g.label(dst, slot+": "+g.S.Data.Equipment[item].Name, 880, float64(y), 16, ink)
				y += 32
			}
		}
	}
}
func (g *Game) drawOverlay(dst *ebiten.Image) {
	rect(dst, 0, 90, 1280, 690, color.NRGBA{R: 14, G: 23, B: 30, A: 248})
	g.buttons = nil
	g.label(dst, "메뉴 · "+g.overlay, 300, 125, 28, gold)
	g.button(dst, "닫기", 850, 130, 110, func() { g.overlay = "" })
	switch g.overlay {
	case "confirm":
		g.label(dst, wrap(g.notice, 36), 300, 200, 20, ink)
		g.button(dst, "확인", 300, 300, 220, func() {
			f := g.confirmation
			g.overlay = ""
			if f != nil {
				f()
			}
		})
		g.button(dst, "취소", 540, 300, 220, func() { g.overlay = "" })
	case "menu":
		g.button(dst, "게임으로", 300, 200, 450, func() { g.overlay = ""; g.request(session.Request{Op: "resume"}) })
		g.button(dst, "저장", 300, 245, 450, func() { g.overlay = "save" })
		g.button(dst, "불러오기", 300, 290, 450, func() { g.overlay = "load" })
		g.button(dst, "설정", 300, 335, 450, func() { g.overlay = "settings" })
		g.button(dst, "시작 메뉴", 300, 380, 450, func() {
			g.request(session.Request{Op: "title"})
			if g.S.Screen == "title" {
				g.overlay = ""
			}
		})
		g.button(dst, "종료", 300, 425, 450, func() { g.request(session.Request{Op: "quit"}) })
	case "save", "load":
		op := g.overlay
		g.label(dst, "슬롯을 선택하십시오. 덮어쓰기는 확인합니다.", 300, 175, 17, muted)
		for i := 1; i <= 10; i++ {
			slot := i
			g.button(dst, fmt.Sprintf("%d  ·  %s", i, op), 300+(i-1)%2*240, 218+(i-1)/2*48, 220, func() {
				g.slot = slot
				g.request(session.Request{Op: op, Slot: fmt.Sprint(slot)})
				if g.overlay != "confirm" {
					g.overlay = ""
				}
			})
		}
		g.button(dst, "자동 저장 불러오기", 300, 480, 460, func() {
			g.request(session.Request{Op: "load", Slot: "auto"})
			if g.overlay != "confirm" {
				g.overlay = ""
			}
		})
	case "settings":
		g.label(dst, fmt.Sprintf("적 AI: %s   로그 상세: %t   대사 속도: %dms", g.S.Settings.AI, g.S.Settings.Detail, g.S.Settings.TextDelayMS), 300, 185, 17, ink)
		g.button(dst, "적 AI 자동 / 한 명령", 300, 240, 460, func() {
			v := g.S.Settings
			if v.AI == "auto" {
				v.AI = "step"
			} else {
				v.AI = "auto"
			}
			g.request(session.Request{Op: "settings", Settings: &v})
		})
		g.button(dst, "상세 로그 전환", 300, 288, 460, func() {
			v := g.S.Settings
			v.Detail = !v.Detail
			g.request(session.Request{Op: "settings", Settings: &v})
		})
		g.button(dst, "대사 속도: 즉시 / 20ms", 300, 336, 460, func() {
			v := g.S.Settings
			if v.TextDelayMS == 0 {
				v.TextDelayMS = 20
			} else {
				v.TextDelayMS = 0
			}
			g.request(session.Request{Op: "settings", Settings: &v})
		})
	case "equipment", "deputy":
		if g.S.Engine == nil {
			return
		}
		o := g.S.Engine.Observe()
		cmds := []core.Command{}
		labels := []string{}
		if g.overlay == "equipment" {
			for _, of := range o.Officers {
				if of.ID == g.selected {
					for _, slot := range sorted(of.Equipment) {
						item := of.Equipment[slot]
						cmds = append(cmds, core.Command{Kind: "unequip", Actor: g.selected, Item: item})
						labels = append(labels, "해제: "+g.S.Data.Equipment[item].Name)
					}
				}
			}
			for _, item := range sorted(o.Warehouse) {
				if o.Warehouse[item] > 0 {
					cmds = append(cmds, core.Command{Kind: "equip", Actor: g.selected, Item: item})
					labels = append(labels, "장착: "+g.S.Data.Equipment[item].Name)
				}
			}
		} else {
			cmds = append(cmds, core.Command{Kind: "undeputy", Actor: g.selected})
			labels = append(labels, "부관 해제")
			for _, of := range o.Officers {
				if of.ID != g.selected && (of.ID == "유비" || of.ID == "관우" || of.ID == "장비" || of.ID == "간옹") {
					cmds = append(cmds, core.Command{Kind: "deputy", Actor: g.selected, Target: of.ID})
					labels = append(labels, "임명: "+of.ID)
				}
			}
		}
		for i := g.page * 8; i < min(len(cmds), g.page*8+8); i++ {
			c := cmds[i]
			g.button(dst, labels[i], 300, 200+(i%8)*44, 460, func() { g.command(c) })
		}
		g.button(dst, "이전", 300, 580, 220, func() { g.page = max(0, g.page-1) })
		g.button(dst, "다음", 540, 580, 220, func() {
			if (g.page+1)*8 < len(cmds) {
				g.page++
			}
		})
	}
}
