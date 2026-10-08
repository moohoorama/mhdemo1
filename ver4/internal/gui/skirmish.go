package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/skirmish"
)

// Skirmish turns the screen into the battle simulator: a setup window for both sides, the
// battlefield and the class and terrain numbers, then a match both sides play themselves.
type Skirmish struct {
	Config skirmish.Config
	Path   string // where 설정 저장 writes Config

	base    *content.Data   // campaign data the overrides apply to
	maps    []content.Stage // one per terrain
	work    *content.Data   // base with the classes and terrain being edited
	setup   bool
	tab     string
	pick    *skirmish.Member // the officer slot the picker fills
	cells   []numCell
	preview map[*skirmish.Member]core.Stats
	err     string
	notice  string
	next    uint64
}

// numCell is a number on the setup screen: click +1, right click -1, wheel, Shift ×10.
type numCell struct {
	r      image.Rectangle
	adjust func(steps int)
}

// NewSkirmish opens the simulator on its setup window with c's settings.
func NewSkirmish(base *content.Data, maps []content.Stage, c skirmish.Config, path string) (*Skirmish, error) {
	work := skirmish.Clone(base)
	if err := skirmish.Apply(work, c); err != nil {
		return nil, err
	}
	k := &Skirmish{Config: c, Path: path, base: base, maps: maps, work: work, setup: true, tab: "sides"}
	k.refresh()
	return k, nil
}

// config is Config with the edited classes and terrain as overrides.
func (k *Skirmish) config() skirmish.Config {
	c := k.Config
	skirmish.SetOverrides(&c, k.base, k.work)
	return c
}

// refresh rebuilds the match data after an edit: each officer's stats, or the reason the
// match cannot start.
func (k *Skirmish) refresh() {
	k.preview, k.err = map[*skirmish.Member]core.Stats{}, ""
	d, err := skirmish.Build(k.base, k.maps, k.config())
	if err != nil {
		k.err = err.Error()
		return
	}
	e, err := core.NewSkirmish(d, 1)
	if err != nil {
		k.err = err.Error()
		return
	}
	for i, side := range []*skirmish.Side{&k.Config.Ally, &k.Config.Enemy} {
		spawns := d.Stages[0].Party
		if i == 1 {
			spawns = d.Stages[0].Enemies
		}
		for j := range side.Officers {
			k.preview[&side.Officers[j]], _ = e.OfficerStats(spawns[j].Officer)
		}
	}
}

// skirmishOver is whether the match has ended, by a rout or by the round limit.
func (g *Game) skirmishOver(o core.Observation) bool {
	return o.Phase == "result" || o.Round > g.Skirmish.Config.MaxRounds
}

// skirmishTick runs the setup window, reporting true while it is open, or keeps both sides
// on autoplay until the match ends; R starts the next match and S goes back to the setup.
func (g *Game) skirmishTick() bool {
	k := g.Skirmish
	if k.setup {
		g.setupInput()
		if len(g.Shots) > 0 { // audit runs capture the setup at the first tick, then play
			g.capture = g.capture || g.ticks == g.Shots[0]
			if g.ticks > g.Shots[0] {
				g.startSkirmish()
			}
		}
		return true
	}
	if g.S.Engine == nil {
		return false
	}
	g.autoplay = !g.skirmishOver(g.S.Engine.Observe())
	if !g.busy() {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.restartSkirmish()
		} else if inpututil.IsKeyJustPressed(ebiten.KeyS) && !g.autoplay {
			g.backToSetup()
		}
	}
	return false
}

func (g *Game) startSkirmish() {
	k := g.Skirmish
	d, err := skirmish.Build(k.base, k.maps, k.config())
	if err != nil {
		k.err = err.Error()
		return
	}
	g.S.Data = d
	k.next = k.Config.Seed
	g.speed = max(1, k.Config.Speed)
	k.setup = false
	g.restartSkirmish()
}

func (g *Game) restartSkirmish() {
	k := g.Skirmish
	e, err := core.NewSkirmish(g.S.Data, k.next)
	if err != nil {
		g.toast(err.Error(), bad)
		return
	}
	k.next++
	g.S.Engine, g.field, g.modals, g.overlay = e, nil, nil, ""
}

func (g *Game) backToSetup() {
	g.Skirmish.setup, g.Skirmish.notice = true, ""
	g.S.Engine, g.field, g.modals, g.overlay, g.autoplay = nil, nil, nil, "", false
}

// setupInput handles the setup window's clicks, right clicks and wheel.
func (g *Game) setupInput() {
	k := g.Skirmish
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.leftClick()
	}
	right := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if k.pick != nil {
		if right || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			k.pick = nil
		}
		return
	}
	steps, n := 1, 0
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		steps = 10
	}
	if right {
		n = -steps
	}
	if _, dy := ebiten.Wheel(); dy != 0 {
		n = int(math.Copysign(float64(steps), dy))
	}
	if n == 0 {
		return
	}
	for _, c := range k.cells {
		if hovered(c.r) {
			c.adjust(n)
			k.refresh()
			return
		}
	}
}

// cell registers a number: a click adds a step (×10 with Shift).
func (g *Game) cell(dst *ebiten.Image, r image.Rectangle, value string, adjust func(steps int)) {
	k := g.Skirmish
	k.cells = append(k.cells, numCell{r, adjust})
	g.btn(dst, r, value, func() {
		steps := 1
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			steps = 10
		}
		adjust(steps)
		k.refresh()
	})
}

// choice is a button that shows whether it is the current option.
func (g *Game) choice(dst *ebiten.Image, r image.Rectangle, label string, on bool, click func()) {
	g.btn(dst, r, label, click)
	if on {
		strokeRect(dst, float32(r.Min.X)+.5, float32(r.Min.Y)+.5, float32(r.Dx())-1, float32(r.Dy())-1, 2, gold)
	}
}

func clampInt(v, lo, hi int) int { return min(hi, max(lo, v)) }

func trimFloat(v float64) string { return strings.TrimSuffix(fmt.Sprintf("%.2f", v), ".00") }

// drawSetup is the simulator's first window.
func (g *Game) drawSetup(dst *ebiten.Image) {
	k := g.Skirmish
	k.cells = nil
	g.titled(dst, image.Rect(16, 24, Width-16, Height-16), "전투 시뮬레이터")
	for i, t := range []struct{ key, label string }{{"sides", "진영 · 전장"}, {"classes", "병종 수치"}, {"terrain", "지형 수치"}} {
		g.choice(dst, image.Rect(40+i*150, 48, 180+i*150, 80), t.label, k.tab == t.key, func() { k.tab = t.key })
	}
	switch k.tab {
	case "sides":
		g.setupSides(dst)
	case "classes":
		g.setupClasses(dst)
	case "terrain":
		g.setupTerrain(dst)
	}
	y := Height - 82
	start := g.startSkirmish
	if k.err != "" {
		start = nil
		g.label(dst, k.err, 40, float64(y-34), 15, bad)
	} else if k.notice != "" {
		g.label(dst, k.notice, 40, float64(y-34), 15, good)
	}
	g.label(dst, "숫자: 클릭 +1 · 우클릭 −1 · 휠 · Shift ×10", float64(Width-420), float64(y-34), 14, muted)
	g.btn(dst, image.Rect(40, y, 260, y+40), "전투 시작", start)
	g.btn(dst, image.Rect(276, y, 456, y+40), "설정 저장", func() {
		k.notice = "저장했습니다: " + k.Path
		if err := k.config().Save(k.Path); err != nil {
			k.notice, k.err = "", err.Error()
		}
	})
	g.btn(dst, image.Rect(472, y, 712, y+40), "병종·지형 원래 값으로", func() {
		k.work = skirmish.Clone(k.base)
		k.refresh()
	})
	g.btn(dst, image.Rect(Width-200, y, Width-40, y+40), "종료", func() { g.S.Screen = "quit" })
	if k.pick != nil {
		g.officerPicker(dst)
	}
}

func (g *Game) setupSides(dst *ebiten.Image) {
	k := g.Skirmish
	c := &k.Config
	y := 104
	g.label(dst, "지형", 40, float64(y+6), 16, gold)
	for i, m := range k.maps {
		name := m.Name
		g.choice(dst, image.Rect(100+i*92, y, 184+i*92, y+32), name, c.Terrain == name, func() { c.Terrain = name; k.refresh() })
	}
	g.label(dst, "배속", 500, float64(y+6), 16, gold)
	for i, s := range []int{1, 2, 4, 8} {
		g.choice(dst, image.Rect(556+i*62, y, 610+i*62, y+32), fmt.Sprintf("×%d", s), max(1, c.Speed) == s, func() { c.Speed = s })
	}
	g.label(dst, "최대 턴", 830, float64(y+6), 16, gold)
	g.cell(dst, image.Rect(904, y, 964, y+32), fmt.Sprint(c.MaxRounds), func(n int) { c.MaxRounds = clampInt(c.MaxRounds+n, 1, 99) })
	g.label(dst, "시드", 990, float64(y+6), 16, gold)
	g.cell(dst, image.Rect(1036, y, 1116, y+32), fmt.Sprint(c.Seed), func(n int) { c.Seed = uint64(max(1, int(c.Seed)+n)) })
	g.setupSide(dst, &c.Ally, "ally", 40)
	g.setupSide(dst, &c.Enemy, "enemy", Width/2+8)
}

// setupSide lists one side's officers with their level, class and resulting stats.
func (g *Game) setupSide(dst *ebiten.Image, side *skirmish.Side, faction string, x int) {
	k := g.Skirmish
	w := Width/2 - 48
	y := 156
	g.label(dst, factionName(faction)+" 진영", float64(x), float64(y+6), 19, gold)
	g.label(dst, "기본 레벨", float64(x+120), float64(y+8), 15, ink)
	g.cell(dst, image.Rect(x+200, y+2, x+256, y+32), fmt.Sprint(side.Level), func(n int) { side.Level = clampInt(side.Level+n, 1, 99) })
	g.label(dst, "장수", float64(x), float64(y+44), 14, muted)
	g.label(dst, "레벨", float64(x+118), float64(y+44), 14, muted)
	g.label(dst, "병종", float64(x+178), float64(y+44), 14, muted)
	g.label(dst, "공격 · 방어 · 순발 · 사기 · 병력", float64(x+330), float64(y+44), 14, muted)
	const rows = 10
	names := classNames(k.work)
	for j := range side.Officers[:min(rows, len(side.Officers))] {
		m := &side.Officers[j]
		ry := y + 68 + j*40
		g.btn(dst, image.Rect(x, ry, x+110, ry+32), k.work.Officers[m.Officer].Name, func() { k.pick = m })
		lv := "기본"
		if m.Level > 0 {
			lv = fmt.Sprint(m.Level)
		}
		g.cell(dst, image.Rect(x+118, ry, x+170, ry+32), lv, func(n int) {
			if m.Level == 0 {
				m.Level = side.Level
			}
			m.Level = clampInt(m.Level+n, 1, 99)
		})
		class := m.Class
		if class == "" {
			class = k.work.Officers[m.Officer].Class
		}
		at := 0
		for i, n := range names {
			if n == class {
				at = i
			}
		}
		g.btn(dst, image.Rect(x+178, ry, x+204, ry+32), "◀", func() { m.Class = names[(at+len(names)-1)%len(names)]; k.refresh() })
		g.label(dst, class, float64(x+212), float64(ry+6), 15, ink)
		g.btn(dst, image.Rect(x+290, ry, x+316, ry+32), "▶", func() { m.Class = names[(at+1)%len(names)]; k.refresh() })
		if s, ok := k.preview[m]; ok {
			g.label(dst, fmt.Sprintf("%d · %d · %d · %d · %d", s.Attack, s.Defense, s.Agility, s.Morale, s.MaxHP), float64(x+330), float64(ry+6), 15, ink)
		}
		if len(side.Officers) > 1 {
			g.btn(dst, image.Rect(x+w-30, ry, x+w, ry+32), "X", func() {
				side.Officers = append(side.Officers[:j:j], side.Officers[j+1:]...)
				k.refresh()
			})
		}
	}
	if n := len(side.Officers); n < rows {
		ry := y + 68 + n*40
		g.btn(dst, image.Rect(x, ry, x+160, ry+32), "+ 장수 추가", func() {
			side.Officers = append(side.Officers, skirmish.Member{Officer: side.Officers[n-1].Officer})
			k.refresh()
			k.pick = &side.Officers[n]
		})
	}
}

// officerPicker chooses the officer for a slot.
func (g *Game) officerPicker(dst *ebiten.Image) {
	k := g.Skirmish
	g.buttons = nil
	rect(dst, 0, 0, Width, Height, color.NRGBA{A: 140})
	ids := officerIDs(k.work)
	r := image.Rect(Width/2-460, 150, Width/2+460, 210+(len(ids)+3)/4*44)
	g.titled(dst, r, "장수 선택")
	for i, id := range ids {
		o := k.work.Officers[id]
		x, y := r.Min.X+28+i%4*218, r.Min.Y+30+i/4*44
		g.choice(dst, image.Rect(x, y, x+206, y+36), o.Name+"  "+o.Class, k.pick.Officer == id, func() {
			k.pick.Officer, k.pick.Class, k.pick = id, "", nil
			k.refresh()
		})
	}
}

// officerIDs are the named officers in name order, then the unnamed troops.
func officerIDs(d *content.Data) []string {
	ids := make([]string, 0, len(d.Officers))
	for id := range d.Officers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := strings.Contains(ids[i], "_troop"), strings.Contains(ids[j], "_troop")
		if a != b {
			return b
		}
		if a {
			return ids[i] < ids[j]
		}
		return d.Officers[ids[i]].Name < d.Officers[ids[j]].Name
	})
	return ids
}

// classNames are the classes by family, then tier.
func classNames(d *content.Data) []string {
	names := make([]string, 0, len(d.Classes))
	for n := range d.Classes {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := d.Classes[names[i]], d.Classes[names[j]]
		if a.Family != b.Family {
			return a.Family < b.Family
		}
		if a.Tier != b.Tier {
			return a.Tier < b.Tier
		}
		return names[i] < names[j]
	})
	return names
}

// setupClasses edits each class's stat bonuses, 병력 factor, move and range.
func (g *Game) setupClasses(dst *ebiten.Image) {
	k := g.Skirmish
	used := map[string]bool{}
	for _, side := range []skirmish.Side{k.Config.Ally, k.Config.Enemy} {
		for _, m := range side.Officers {
			if m.Class != "" {
				used[m.Class] = true
			} else {
				used[k.work.Officers[m.Officer].Class] = true
			}
		}
	}
	x0, y := 40, 116
	g.label(dst, "공격~병력은 능력치 공식의 병과보정치입니다. 금색은 출전 중인 병종", float64(x0), float64(y-26), 13, muted)
	g.label(dst, "병종", float64(x0), float64(y), 14, muted)
	g.label(dst, "계통", float64(x0+110), float64(y), 14, muted)
	for i, h := range []string{"공격", "방어", "정신", "순발", "사기", "병력", "병력계수", "이동", "사거리"} {
		g.label(dst, h, float64(x0+200+i*100), float64(y), 14, muted)
	}
	for j, name := range classNames(k.work) {
		ry := y + 26 + j*34
		c := k.work.Classes[name]
		fg := color.Color(ink)
		if used[name] {
			fg = gold
		}
		g.label(dst, name, float64(x0), float64(ry+5), 15, fg)
		g.label(dst, c.Family, float64(x0+110), float64(ry+5), 14, muted)
		for i := range 6 {
			g.cell(dst, image.Rect(x0+200+i*100, ry, x0+284+i*100, ry+28), trimFloat(c.Bonus[i]), func(n int) {
				c := k.work.Classes[name]
				c.Bonus[i] = math.Max(0, c.Bonus[i]+float64(n))
				k.work.Classes[name] = c
			})
		}
		for i, v := range []int{c.HPFactor, c.Move, c.Range} {
			g.cell(dst, image.Rect(x0+800+i*100, ry, x0+884+i*100, ry+28), fmt.Sprint(v), func(n int) {
				c := k.work.Classes[name]
				p := []*int{&c.HPFactor, &c.Move, &c.Range}[i]
				*p = clampInt(*p+n, 1, 30)
				k.work.Classes[name] = c
			})
		}
	}
}

// setupTerrain edits each family's damage factor and move cost on the simulator terrains.
func (g *Game) setupTerrain(dst *ebiten.Image) {
	k := g.Skirmish
	x0, y := 40, 120
	families := make([]string, 0, len(k.work.Terrain))
	for f := range k.work.Terrain {
		families = append(families, f)
	}
	sort.Strings(families)
	g.label(dst, "계수는 공격력·방어력에 곱하는 지형 보정, 비용은 이동력 소모(0은 들어갈 수 없음)입니다. 금색은 선택한 지형", float64(x0), float64(y-26), 13, muted)
	for i, m := range k.maps {
		cx := x0 + 120 + i*270
		fg := color.Color(ink)
		if m.Name == k.Config.Terrain {
			fg = gold
		}
		g.label(dst, m.Name, float64(cx), float64(y+4), 17, fg)
		g.label(dst, "계수", float64(cx), float64(y+34), 14, muted)
		g.label(dst, "비용", float64(cx+110), float64(y+34), 14, muted)
	}
	for j, f := range families {
		ry := y + 62 + j*44
		g.label(dst, f, float64(x0), float64(ry+6), 16, ink)
		for i, m := range k.maps {
			tile := string(m.Tiles[0][0])
			cx := x0 + 120 + i*270
			t := k.work.Terrain[f][tile]
			g.cell(dst, image.Rect(cx, ry, cx+96, ry+32), trimFloat(t.Factor), func(n int) {
				t := k.work.Terrain[f][tile]
				t.Factor = math.Max(.05, math.Round((t.Factor+float64(n)*.05)*100)/100)
				k.work.Terrain[f][tile] = t
			})
			cost := fmt.Sprint(t.Cost)
			if t.Cost == 0 {
				cost = "불가"
			}
			g.cell(dst, image.Rect(cx+110, ry, cx+206, ry+32), cost, func(n int) {
				t := k.work.Terrain[f][tile]
				t.Cost = clampInt(t.Cost+n, 0, 9)
				k.work.Terrain[f][tile] = t
			})
		}
	}
}

// skirmishResult is the end-of-match window: the winner and every unit's 병력.
func (g *Game) skirmishResult(dst *ebiten.Image, o core.Observation) {
	g.buttons, g.blocks = nil, nil
	rect(dst, 0, 0, Width, Height, color.NRGBA{A: 120})
	title, c := "무승부", gold
	switch o.Result {
	case "victory":
		title, c = "아군 승리", good
	case "defeat":
		title, c = "적군 승리", bad
	}
	rounds := o.Round
	if o.Phase != "result" {
		rounds = g.Skirmish.Config.MaxRounds
	}
	r := image.Rect(Width/2-420, 100, Width/2+420, 760)
	g.titled(dst, r, "전투 결과")
	g.centered(dst, title, float64(Width/2), 130, 30, c)
	g.centered(dst, fmt.Sprintf("%d턴 · 시드 %d", rounds, g.Skirmish.next-1), float64(Width/2), 176, 17, muted)
	for i, side := range []string{"ally", "enemy"} {
		x := r.Min.X + 40 + i*400
		g.label(dst, factionName(side)+" 진영", float64(x), 216, 19, gold)
		for j, u := range unitsByFaction(o, side) {
			y := 254 + j*40
			if y > r.Max.Y-110 {
				break
			}
			k := float64(u.HP) / float64(max(1, u.Stats.MaxHP))
			name := fmt.Sprintf("%s  Lv%d  %s", u.Name, u.Level, u.Class)
			if u.HP <= 0 {
				g.label(dst, name+"  퇴각", float64(x), float64(y), 16, muted)
				continue
			}
			g.label(dst, name, float64(x), float64(y), 16, ink)
			bar(dst, float32(x), float32(y+24), 240, 6, k, hpColor(k))
			g.label(dst, fmt.Sprintf("%d / %d", u.HP, u.Stats.MaxHP), float64(x+252), float64(y+14), 14, ink)
		}
	}
	w := (r.Dx() - 80 - 32) / 3
	g.btn(dst, image.Rect(r.Min.X+40, r.Max.Y-64, r.Min.X+40+w, r.Max.Y-26), "다시 (R)", g.restartSkirmish)
	g.btn(dst, image.Rect(r.Min.X+56+w, r.Max.Y-64, r.Min.X+56+2*w, r.Max.Y-26), "설정 (S)", g.backToSetup)
	g.btn(dst, image.Rect(r.Max.X-40-w, r.Max.Y-64, r.Max.X-40, r.Max.Y-26), "종료", func() { g.S.Screen = "quit" })
}
