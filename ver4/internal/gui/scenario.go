package gui

import (
	"fmt"
	"image"
	"image/color"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// line is one speaker's part of a dialogue node. Content keeps several speakers in one
// text ("유비: … 관우·장비: …"); the screen shows them one by one.
type line struct {
	speakers []string
	text     string
}

var speakerMark = regexp.MustCompile(`(?:^|\s)([가-힣]{1,4}(?:·[가-힣]{1,4})*):\s*`)

func splitLines(s string) []line {
	marks := speakerMark.FindAllStringSubmatchIndex(s, -1)
	if len(marks) == 0 {
		return []line{{text: s}}
	}
	out := []line{}
	if head := strings.TrimSpace(s[:marks[0][0]]); head != "" {
		out = append(out, line{text: head})
	}
	for i, m := range marks {
		end := len(s)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}
		out = append(out, line{speakers: strings.Split(s[m[2]:m[3]], "·"), text: strings.TrimSpace(s[m[1]:end])})
	}
	return out
}

// shownText is the typed-out part of the current line.
func (g *Game) shownText(l line) (string, bool) {
	r := []rune(l.text)
	if g.S.Settings.TextDelayMS <= 0 {
		return l.text, true
	}
	n := min(len(r), (g.ticks-g.lineStart)*1000/(60*g.S.Settings.TextDelayMS))
	return string(r[:n]), n == len(r)
}

// advanceDialogue finishes typing, then shows the next speaker, then the next node.
func (g *Game) advanceDialogue() {
	o := g.S.Engine.Observe()
	if o.Phase != "scenario" || o.Dialogue.Kind != "dialogue" || g.busy() {
		return
	}
	lines := splitLines(o.Dialogue.Text)
	if _, done := g.shownText(lines[min(g.lineIndex, len(lines)-1)]); !done {
		g.lineStart = -1 << 30
		return
	}
	if g.lineIndex+1 < len(lines) {
		g.lineIndex++
		g.lineStart = g.ticks
		return
	}
	g.command(core.Command{Kind: "next"})
}

func (g *Game) backdrop(dst *ebiten.Image, o core.Observation) {
	for i := 0; i < Height; i += 10 {
		k := float64(i) / Height
		rect(dst, 0, float32(i), Width, 10, color.NRGBA{R: uint8(36 - 20*k), G: uint8(28 - 14*k), B: uint8(26 - 10*k), A: 255})
	}
	g.label(dst, "桃園", 80, 120, 140, color.NRGBA{R: 226, G: 182, B: 93, A: 26})
	g.label(dst, fmt.Sprintf("제 %d 장", o.Stage+1), 60, 40, 22, gold)
	g.label(dst, g.S.Data.Stages[o.Stage].Name, 60, 74, 16, muted)
}

func (g *Game) scenario(dst *ebiten.Image, o core.Observation) {
	g.backdrop(dst, o)
	if g.shownNode != o.Node {
		g.shownNode, g.lineIndex, g.lineStart = o.Node, 0, g.ticks
	}
	if o.Dialogue.Kind == "preparation" {
		g.preparation(dst, o)
		return
	}
	lines := splitLines(o.Dialogue.Text)
	l := lines[min(g.lineIndex, len(lines)-1)]
	// speakers stand above the box
	for i, sp := range l.speakers {
		if isParty(sp) || hasEnemyPortrait[sp] {
			g.portrait(dst, sp, float64(60+i*190), 330, 180)
		}
	}
	r := image.Rect(40, 560, Width-40, 800)
	window(dst, r)
	text, done := g.shownText(l)
	x := float64(r.Min.X + 36)
	if len(l.speakers) > 0 {
		name := strings.Join(l.speakers, "·")
		w := g.width(name, 20) + 40
		rect(dst, float32(r.Min.X+24), float32(r.Min.Y-16), float32(w), 34, color.NRGBA{R: 108, G: 30, B: 26, A: 255})
		rect(dst, float32(r.Min.X+24), float32(r.Min.Y-16), float32(w), 2, gold)
		g.label(dst, name, float64(r.Min.X+44), float64(r.Min.Y-13), 20, ink)
	}
	g.label(dst, wrap(text, 44), x, float64(r.Min.Y+34), 23, ink)
	if o.Dialogue.Kind == "choice" {
		y := 380
		box := image.Rect(Width/2-180, y-30, Width/2+180, y+20+len(o.Dialogue.Choices)*52)
		g.titled(dst, box, "선택")
		for _, option := range sorted(o.Dialogue.Choices) {
			key := option
			g.btn(dst, image.Rect(box.Min.X+24, y, box.Max.X-24, y+40), key, func() { g.command(core.Command{Kind: "choose", Option: key}) })
			y += 52
		}
		return
	}
	if done && g.ticks/30%2 == 0 {
		g.label(dst, "▼", float64(r.Max.X-44), float64(r.Max.Y-40), 16, gold)
	}
	g.buttons = append(g.buttons, button{image.Rect(0, 0, Width, Height), "", g.advanceDialogue})
}

var party = []string{"유비", "관우", "장비", "간옹"}

var hasEnemyPortrait = map[string]bool{"여포": true, "화웅": true, "장각": true}

func isParty(id string) bool {
	for _, p := range party {
		if p == id {
			return true
		}
	}
	return false
}

// preparation is the sortie screen: who goes, their stats, equipment and deputy.
func (g *Game) preparation(dst *ebiten.Image, o core.Observation) {
	if !isParty(g.selected) {
		g.selected = "유비"
	}
	limit := g.S.Data.Stages[o.Stage].Limit
	r := image.Rect(40, 120, Width-40, 810)
	g.titled(dst, r, "출전 준비 · "+g.S.Data.Stages[o.Stage].Name)
	g.label(dst, fmt.Sprintf("출전 %d / %d", len(o.Deployment), limit), float64(r.Min.X+32), float64(r.Min.Y+26), 17, gold)
	deployed := map[string]bool{}
	for _, id := range o.Deployment {
		deployed[id] = true
	}
	y := r.Min.Y + 64
	for _, of := range o.Officers {
		if !isParty(of.ID) {
			continue
		}
		officer := of
		row := image.Rect(r.Min.X+28, y, r.Min.X+470, y+76)
		g.btn(dst, row, "", func() { g.selected = officer.ID; g.page = 0 })
		if g.selected == of.ID {
			rect(dst, float32(row.Min.X), float32(row.Min.Y), 4, float32(row.Dy()), gold)
		}
		mark := "대기"
		if deployed[of.ID] {
			mark = "출전 ✓"
		}
		var toggle func()
		if !of.Dead && of.ID != "유비" {
			toggle = func() { g.toggleDeploy(o, officer.ID) }
		}
		g.btn(dst, image.Rect(row.Max.X-96, y+22, row.Max.X-12, y+54), mark, toggle)
		g.portrait(dst, of.ID, float64(row.Min.X+14), float64(y+8), 60)
		c := ink
		if of.Dead {
			c = muted
		}
		g.label(dst, g.name(of.ID), float64(row.Min.X+88), float64(y+8), 19, c)
		sub := fmt.Sprintf("Lv%d · %s", of.Level, of.Class)
		if of.Dead {
			sub += " · 사망"
		}
		if of.Deputy != "" {
			sub += " · 부관 " + g.name(of.Deputy)
		}
		g.label(dst, sub, float64(row.Min.X+88), float64(y+40), 14, muted)
		y += 86
	}
	g.label(dst, "군량 · 도구", float64(r.Min.X+32), float64(y+12), 15, gold)
	items := []string{}
	for _, id := range sorted(o.Inventory) {
		if o.Inventory[id] > 0 {
			items = append(items, fmt.Sprintf("%s ×%d", id, o.Inventory[id]))
		}
	}
	g.label(dst, wrap(strings.Join(items, "  ·  "), 34), float64(r.Min.X+32), float64(y+38), 14, ink)
	g.officerDetail(dst, o, image.Rect(r.Min.X+500, r.Min.Y+50, r.Max.X-28, r.Max.Y-90))
	g.btn(dst, image.Rect(r.Max.X-330, r.Max.Y-72, r.Max.X-28, r.Max.Y-28), "출  전", func() { g.command(core.Command{Kind: "start"}) })
}

func (g *Game) toggleDeploy(o core.Observation, id string) {
	ids := []string{}
	found := false
	for _, d := range o.Deployment {
		if d == id {
			found = true
		} else {
			ids = append(ids, d)
		}
	}
	if !found {
		ids = append(ids, id)
	}
	g.command(core.Command{Kind: "deploy", Deployment: ids})
}

func (g *Game) officerDetail(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	var of *core.Officer
	for i := range o.Officers {
		if o.Officers[i].ID == g.selected {
			of = &o.Officers[i]
		}
	}
	if of == nil {
		return
	}
	x, y := float64(r.Min.X), float64(r.Min.Y)
	g.portrait(dst, of.ID, x, y, 120)
	g.label(dst, g.name(of.ID), x+140, y, 28, gold)
	g.label(dst, fmt.Sprintf("%s · Lv%d", of.Class, of.Level), x+140, y+42, 17, ink)
	progress := core.ExperienceProgress(of.Level, of.XP)
	bar(dst, float32(x+140), float32(y+74), 260, 8, progress/100, xpC)
	g.label(dst, fmt.Sprintf("EXP %.1f / 100   ·   특성치 %d", progress, of.Points), x+140, y+86, 14, muted)
	stats, err := g.S.Engine.OfficerStats(of.ID)
	if err == nil {
		for i, s := range statRows {
			col, row := i%2, i/2
			g.label(dst, s.name, x+float64(col*200), y+140+float64(row*28), 15, muted)
			g.label(dst, fmt.Sprint(s.get(stats)), x+80+float64(col*200), y+140+float64(row*28), 15, ink)
		}
	}
	g.label(dst, "장비", x, y+262, 16, gold)
	ey := y + 290
	for _, slot := range []string{"weapon", "armor", "aux"} {
		name := "—"
		if item, ok := of.Equipment[slot]; ok {
			name = g.itemName(item)
		}
		g.label(dst, map[string]string{"weapon": "무기", "armor": "방어구", "aux": "보조"}[slot], x, ey, 15, muted)
		g.label(dst, name, x+80, ey, 15, ink)
		ey += 28
	}
	traits := strings.Join(of.Learned, " · ")
	if traits == "" {
		traits = "—"
	}
	g.label(dst, wrap("습득 특성  "+traits, 40), x, ey+8, 13, muted)
	var deputy func()
	if !of.Dead {
		deputy = func() { g.openOverlay("deputy") }
	}
	g.btn(dst, image.Rect(r.Min.X, r.Max.Y-44, r.Min.X+200, r.Max.Y-8), "장비 변경", func() { g.openOverlay("equipment") })
	g.btn(dst, image.Rect(r.Min.X+216, r.Max.Y-44, r.Min.X+416, r.Max.Y-8), "부관 지정", deputy)
}

// equipmentWindow lists equip/unequip (or deputy) commands with the stat change each makes.
func (g *Game) equipmentWindow(dst *ebiten.Image, x, y, w, h int) {
	if g.S.Engine == nil {
		return
	}
	o := g.S.Engine.Observe()
	cmds, labels := []core.Command{}, []string{}
	if g.overlay == "equipment" {
		for _, of := range o.Officers {
			if of.ID == g.selected {
				for _, slot := range sorted(of.Equipment) {
					item := of.Equipment[slot]
					cmds = append(cmds, core.Command{Kind: "unequip", Actor: g.selected, Item: item})
					labels = append(labels, "해제  "+g.itemName(item))
				}
			}
		}
		for _, item := range sorted(o.Warehouse) {
			if o.Warehouse[item] > 0 {
				cmds = append(cmds, core.Command{Kind: "equip", Actor: g.selected, Item: item})
				labels = append(labels, fmt.Sprintf("장착  %s ×%d", g.itemName(item), o.Warehouse[item]))
			}
		}
	} else {
		cmds = append(cmds, core.Command{Kind: "undeputy", Actor: g.selected})
		labels = append(labels, "부관 해제")
		for _, of := range o.Officers {
			if of.ID != g.selected && isParty(of.ID) {
				cmds = append(cmds, core.Command{Kind: "deputy", Actor: g.selected, Target: of.ID})
				labels = append(labels, "임명  "+g.name(of.ID))
			}
		}
	}
	g.label(dst, g.name(g.selected)+" — 항목에 마우스를 올리면 능력치 변화를 봅니다.", float64(x+32), float64(y+52), 14, muted)
	per := 9
	for i := g.page * per; i < min(len(cmds), g.page*per+per); i++ {
		c := cmds[i]
		row := image.Rect(x+32, y+84+(i%per)*42, x+w-32, y+84+(i%per)*42+36)
		g.btn(dst, row, labels[i], func() { g.command(c) })
		if hovered(row) {
			g.label(dst, g.statDiff(c), float64(x+300), float64(row.Min.Y+9), 13, good)
		}
	}
	g.btn(dst, image.Rect(x+32, y+h-56, x+182, y+h-22), "이전", func() { g.page = max(0, g.page-1) })
	g.btn(dst, image.Rect(x+198, y+h-56, x+348, y+h-22), "다음", func() {
		if (g.page+1)*per < len(cmds) {
			g.page++
		}
	})
}

// statDiff applies c to a copy of the game and reports the selected officer's stat changes.
func (g *Game) statDiff(c core.Command) string {
	key := fmt.Sprint(g.S.Engine.Observe().Revision, c)
	if s, ok := g.diffs[key]; ok {
		return s
	}
	out := "변화 없음"
	before, err := g.S.Engine.OfficerStats(c.Actor)
	sb, err2 := core.Restore(g.S.Data, g.S.Engine.Snapshot())
	if err == nil && err2 == nil {
		if _, err := sb.Apply(c); err != nil {
			out = koreanError(err.Error())
		} else if after, err := sb.OfficerStats(c.Actor); err == nil {
			parts := []string{}
			for _, s := range statRows {
				if d := s.get(after) - s.get(before); d != 0 {
					parts = append(parts, fmt.Sprintf("%s %+d", s.name, d))
				}
			}
			if len(parts) > 0 {
				out = strings.Join(parts, "  ")
			}
		}
	}
	if len(g.diffs) > 128 {
		g.diffs = map[string]string{}
	}
	g.diffs[key] = out
	return out
}
