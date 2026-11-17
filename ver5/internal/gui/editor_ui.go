package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"srpg/internal/content"
)

func (g *Game) editorToolbar(dst *ebiten.Image) {
	e := g.editor
	g.window(dst, image.Rect(16, 700, 1266, 811))
	g.btn(dst, image.Rect(24, 710, 142, 746), "타일 [Tab]", func() { e.tab = 0 })
	g.btn(dst, image.Rect(24, 751, 142, 790), "병력 [Tab]", func() { e.tab = 1 })
	if e.tab == 0 {
		for i, id := range g.S.Data.MapEditor.Brushes {
			if i < e.brushOffset || i >= e.brushOffset+11 {
				continue
			}
			i, id := i, id
			x := 152 + (i-e.brushOffset)*96
			label := brushName(g.S.Data, id)
			if i < len(editorKeyNames) {
				label = string(editorKeyNames[i]) + " " + label
			}
			if e.brush == i {
				label = "✓ " + label
			}
			g.btn(dst, image.Rect(x, 714, x+91, 755), label, func() { e.brush = i })
		}
		for i, n := range []int{1, 2, 4, 8} {
			label := fmt.Sprintf("%d×%d", n, n)
			if max(1, e.brushSize) == n {
				label = "✓ " + label
			}
			g.btn(dst, image.Rect(152+i*86, 764, 232+i*86, 795), label, func() { e.brushSize = n })
		}
		g.label(dst, "붓 크기(칸) · 좌드래그: 칠하기 / 우클릭: 가져오기", 510, 773, 14, muted)
		return
	}
	g.btn(dst, image.Rect(152, 708, 284, 738), "아군 배치 칸", func() { e.ally = true })
	g.btn(dst, image.Rect(292, 708, 408, 738), "일반병", func() { e.ally = false; e.officer = false; e.pick = "" })
	g.btn(dst, image.Rect(416, 708, 530, 738), "장수 선택", func() { e.ally = false; e.officer = true; e.popup = "officers"; e.choices = tableList{}; e.entry = "" })
	for i, f := range g.S.Data.MapEditor.Families {
		i, f := i, f
		g.btn(dst, image.Rect(540+i*112, 708, 648+i*112, 738), f.Name, func() { e.family = i; e.classIndex = 0; e.pick = ""; e.officerClass = "" })
	}
	families := g.S.Data.MapEditor.Families
	if len(families) == 0 {
		return
	}
	f := families[e.family]
	for i, c := range f.Classes {
		i, c := i, c
		x := 152 + i*124
		fn := func() { e.classIndex = i }
		label := c.Name
		if e.classIndex == i {
			label = "✓ " + label
		}
		if _, ok := g.S.Data.Classes[c.ID]; !ok {
			label = c.Name + "(미구현)"
			fn = nil
		}
		g.btn(dst, image.Rect(x, 748, x+118, 778), label, fn)
	}
	if g.S.Data.Rules.StageLevelBase > 0 {
		g.btn(dst, image.Rect(1020, 748, 1172, 780), editorRankNames[e.rank], func() { e.rank = nextRank(e.rank) })
	} else {
		g.btn(dst, image.Rect(1020, 748, 1064, 780), "−", func() { e.level = max(1, e.level-1) })
		g.label(dst, fmt.Sprintf("LV %d", e.level), 1070, 755, 16, ink)
		g.btn(dst, image.Rect(1130, 748, 1172, 780), "+", func() { e.level = min(g.S.Data.Rules.MaxLevel, e.level+1) })
	}
	g.editorFacingButton(dst, image.Rect(1180, 742, 1256, 800), g.editorBrushSpawn(), func() { e.facing = nextFacing(e.facing) })
}

var editorFacings = []string{"NE", "SE", "SW", "NW"}

// editorRanks are the ranks a spawn cycles through when the rules set a stage level; none is the
// character's default (troop for a template, officer for a named one).
var editorRanks = []string{"", "troop", "officer", "commander", "boss"}
var editorRankNames = map[string]string{"": "직급 기본", "troop": "병졸", "officer": "장수", "commander": "대장", "boss": "거물"}

func nextRank(rank string) string {
	for i, r := range editorRanks {
		if r == rank {
			return editorRanks[(i+1)%len(editorRanks)]
		}
	}
	return ""
}

func nextFacing(dir string) string {
	for i, d := range editorFacings {
		if d == dir {
			return editorFacings[(i+1)%len(editorFacings)]
		}
	}
	return "NW"
}

func (g *Game) editorBrushSpawn() content.Spawn {
	e := g.editor
	class := g.editorClass()
	id := e.pick
	if !e.officer {
		id = ""
		for _, candidate := range sorted(g.S.Data.Characters) {
			c := g.S.Data.Characters[candidate]
			if c.Template && g.S.Data.ClassFamily(c.Class) == g.S.Data.ClassFamily(class) {
				id = candidate
				break
			}
		}
	}
	return g.editorRanked(content.Spawn{Character: id, Class: class, Facing: e.facing})
}

func (g *Game) editorFacingButton(dst *ebiten.Image, r image.Rectangle, sp content.Spawn, click func()) {
	g.btn(dst, r, "", click)
	if _, ok := g.S.Data.Characters[sp.Character]; !ok {
		g.centered(dst, "방향", float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+16), 14, muted)
		return
	}
	v := g.editorPreview(sp)
	art := g.battleClassArt(v.Def, v.Class, v.Weapon, v.Mount, v.Color)
	if art == nil {
		return
	}
	dir := sp.Facing
	if dir == "" {
		dir = "NW"
	}
	z := min(2, float64(r.Dy()-6)/max(1, art.TopOffset()))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(float64(r.Min.X+r.Dx()/2)-art.Pivot[0]*z, float64(r.Max.Y-3)-art.Pivot[1]*z)
	dst.DrawImage(art.Frame(dir, "idle", 0), op)
	x, y := cursorPos()
	if image.Pt(x, y).In(r) {
		g.tooltip(dst, "클릭하면 방향이 바뀝니다", r.Min.X, r.Min.Y-70)
	}
}

func (g *Game) editorDetail(dst *ebiten.Image) {
	e := g.editor
	d := g.S.Data
	x, y := cursorPos()
	if image.Pt(x, y).In(editorViewport) {
		p := e.cellAt(x, y)
		c := e.doc.Cell(p.X, p.Y)
		layerName := func(name string, ok bool) string {
			if !ok {
				return "없음"
			}
			return name
		}
		st, hasStructure := d.Structures[string(c.Structure)]
		sn, hasState := d.States[string(c.State)]
		g.label(dst, fmt.Sprintf("칸 (%d,%d)", p.X, p.Y), 1018, 82, 14, muted)
		g.label(dst, "땅: "+d.Grounds[string(c.Ground)].Name, 1018, 108, 17, ink)
		g.label(dst, "구조물: "+layerName(st.Name, hasStructure), 1018, 136, 17, ink)
		g.label(dst, "상태: "+layerName(sn.Name, hasState), 1018, 164, 17, ink)
		if !e.doc.Play.Contains(p.X, p.Y) {
			g.label(dst, "외곽 · 병력 배치 불가", 1018, 230, 15, bad)
		} else {
			t := d.Props(c)
			g.label(dst, fmt.Sprintf("지형 회복 HP %d%% / MP ×%d", t.RestHP, max(1, t.RestMP)), 1018, 230, 13, muted)
		}
	}
	g.btn(dst, image.Rect(1018, 646, 1252, 676), "승리 조건 / 일기토", func() { e.popup = "scenario"; e.choices = tableList{} })
	g.btn(dst, image.Rect(1018, 254, 1252, 284), "외곽 어둡게 보기 전환", func() { e.field.ShowExterior = !e.field.ShowExterior })
	if e.draft == nil {
		g.label(dst, "병력을 클릭하면 상세 설정", 1018, 310, 16, muted)
		return
	}
	s := e.draft
	c := d.Characters[s.Character]
	g.label(dst, c.Name, 1018, 301, 23, gold)
	class := s.Class
	if class == "" {
		class = c.Class
	}
	g.btn(dst, image.Rect(1018, 340, 1252, 372), d.Classes[class].Name+" · 병종 변경", func() { e.popup = "classes"; e.choices = tableList{} })
	v := g.editorPreview(*s)
	if d.Rules.StageLevelBase > 0 {
		g.btn(dst, image.Rect(1018, 382, 1148, 412), fmt.Sprintf("%s · LV %d", editorRankNames[s.Rank], v.Level), func() { s.Rank = nextRank(s.Rank) })
	} else {
		g.btn(dst, image.Rect(1018, 382, 1148, 412), fmt.Sprintf("LV %d", editorLevel(c.Level, s.Level)), func() { e.editing = "level"; e.entry = fmt.Sprint(editorLevel(c.Level, s.Level)) })
	}
	g.editorFacingButton(dst, image.Rect(1156, 378, 1252, 418), *s, func() { s.Facing = nextFacing(s.Facing) })
	hp, mp := v.HP, v.MP
	if s.HP != nil {
		hp = *s.HP
	}
	if s.MP != nil {
		mp = *s.MP
	}
	g.btn(dst, image.Rect(1018, 422, 1252, 452), fmt.Sprintf("HP %d / %d", hp, v.Stats.MaxHP), func() { e.editing = "hp"; e.entry = fmt.Sprint(hp) })
	g.btn(dst, image.Rect(1018, 462, 1252, 492), fmt.Sprintf("MP %d / %d", mp, v.Stats.MaxMP), func() { e.editing = "mp"; e.entry = fmt.Sprint(mp) })
	g.btn(dst, image.Rect(1018, 502, 1252, 532), fmt.Sprintf("특성 설정 (%d)", len(v.Traits)), func() {
		if s.Traits == nil {
			traits := append([]string{}, c.Traits...)
			s.Traits = &traits
		}
		e.popup = "traits"
		e.choices = tableList{}
	})
	g.btn(dst, image.Rect(1018, 542, 1252, 572), "장비 설정", func() {
		if s.Equipment == nil {
			eq := append([]string{}, c.Equipment...)
			s.Equipment = &eq
		}
		e.popup = "equipment"
		e.choices = tableList{}
	})
	g.btn(dst, image.Rect(1018, 596, 1132, 632), "적용", g.editorApply)
	g.btn(dst, image.Rect(1140, 596, 1252, 632), "초기화", func() {
		e.draft = &content.Spawn{Character: s.Character, X: s.X, Y: s.Y, Facing: "NW", Policy: s.Policy, Goal: s.Goal, GoalArea: s.GoalArea}
	})
}
func (g *Game) editorPopup(dst *ebiten.Image) {
	e := g.editor
	kind := e.popup
	rect(dst, 0, 0, Width, Height, color.NRGBA{A: 185})
	g.buttons, g.wheelAreas = nil, nil
	r := g.browserPanel(dst, map[string]string{"officers": "장수 선택", "classes": "병종 선택", "traits": "특성 선택", "equipment": "장비 선택", "scenario": "승리 조건 / 일기토", "leave": "저장하지 않은 변경", "maps": "맵 목록으로"}[kind], -1, func() { e.popup = "" })
	choices := []editorChoice{}
	if kind == "leave" || kind == "maps" {
		g.label(dst, "저장하지 않은 변경이 있습니다.", 200, 210, 23, ink)
		exit := func() {
			e.dirty = false
			e.popup = ""
			if kind == "maps" {
				e.doc = nil
			} else {
				g.editorLeave()
			}
		}
		g.btn(dst, image.Rect(200, 300, 444, 348), "저장 후 이동", func() {
			g.editorSave(false)
			if !e.dirty {
				exit()
			}
		})
		g.btn(dst, image.Rect(456, 300, 700, 348), "버리고 이동", exit)
		g.btn(dst, image.Rect(712, 300, 956, 348), "취소", func() { e.popup = "" })
		return
	}
	if kind == "officers" {
		data := g.S.Data
		families := data.MapEditor.Families
		family := families[e.family]
		army := data.MapEditor.Armies[e.army].ID
		for i, a := range data.MapEditor.Armies {
			count := len(editorOfficers(data, family.ID, e.officerClass, a.ID, e.query))
			label := fmt.Sprintf("%s %d", a.Name, count)
			if e.army == i {
				label = "✓ " + label
			}
			g.btn(dst, image.Rect(182+i*148, 150, 324+i*148, 198), label, func() { e.army = i; e.choices = tableList{} })
		}
		for i, f := range families {
			count := len(editorOfficers(data, f.ID, "", army, e.query))
			label := fmt.Sprintf("%s %d", f.Name, count)
			if e.family == i {
				label = "✓ " + label
			}
			g.btn(dst, image.Rect(182+i*148, 206, 324+i*148, 254), label, func() { e.family = i; e.classIndex = 0; e.officerClass = ""; e.choices = tableList{} })
		}
		all := len(editorOfficers(data, family.ID, "", army, e.query))
		label := fmt.Sprintf("전체 %d", all)
		if e.officerClass == "" {
			label = "✓ " + label
		}
		g.btn(dst, image.Rect(182, 262, 290, 310), label, func() { e.officerClass = ""; e.choices = tableList{} })
		for i, c := range family.Classes {
			count := len(editorOfficers(data, family.ID, c.ID, army, e.query))
			label := fmt.Sprintf("%s %d", c.Name, count)
			if e.officerClass == c.ID {
				label = "✓ " + label
			}
			fn := func() { e.officerClass = c.ID; e.classIndex = i; e.choices = tableList{} }
			if _, ok := data.Classes[c.ID]; !ok {
				label = c.Name + "(미구현)"
				fn = nil
			}
			g.btn(dst, image.Rect(297+i*115, 262, 405+i*115, 310), label, fn)
		}
		g.btn(dst, image.Rect(182, 318, 690, 366), "이름 검색: "+e.query, func() { e.entry = e.query; e.editing = "search" })
		ids := editorOfficers(data, family.ID, e.officerClass, army, e.query)
		for _, id := range ids {
			used := false
			for _, sp := range append(append([]content.Spawn{}, e.doc.Stage.Enemies...), e.doc.Stage.Party...) {
				used = used || sp.Character == id
			}
			for _, a := range data.Campaign.Roster {
				used = used || a == id
			}
			fn := func() {
				e.pick = id
				e.popup = ""
				e.message = "장수 선택: " + data.Characters[id].Name
				for i, c := range family.Classes {
					if c.ID == data.ClassLine(data.Characters[id].Class)[0] {
						e.classIndex = i
						break
					}
				}
			}
			label := data.Characters[id].Name
			if used {
				label += " (배치됨)"
				fn = nil
			}
			choices = append(choices, editorChoice{label: label, detail: data.Classes[data.Characters[id].Class].Name, portrait: id, pick: fn})
		}
		r.Min.Y = 378
		g.editorChoices(dst, r, choices)
		return
	}
	if kind == "scenario" {
		st := &e.doc.Stage
		rout := st.RoutVictory()
		bossID := ""
		if targets := st.VictoryTargets(); len(targets) > 0 {
			bossID = targets[0]
		}
		g.btn(dst, image.Rect(200, 184, 1020, 224), "승리 조건: "+map[bool]string{true: "적군 전멸", false: "보스 격파"}[rout], func() {
			e.history.Push(e.doc)
			if rout {
				st.Victory = bossVictory(g, bossID)
			} else {
				st.Victory = routVictory()
			}
			e.dirty = true
		})
		bosses := []string{""}
		for _, s := range st.Enemies {
			if !g.S.Data.Characters[s.Character].Template {
				bosses = append(bosses, s.Character)
			}
		}
		g.btn(dst, image.Rect(200, 238, 1020, 278), "보스: "+g.name(bossID), func() {
			e.history.Push(e.doc)
			n := 0
			for i, id := range bosses {
				if id == bossID {
					n = i + 1
				}
			}
			st.Victory = bossVictory(g, bosses[n%len(bosses)])
			e.dirty = true
		})
		for i, event := range st.Duels {
			choices = append(choices, editorChoice{label: event.ID + " · 삭제", pick: func() { e.history.Push(e.doc); st.Duels = append(st.Duels[:i], st.Duels[i+1:]...); e.dirty = true }})
		}
		r.Min.Y = 306
		g.editorChoices(dst, r, choices)
		return
	}
	if e.draft == nil {
		return
	}
	s := e.draft
	data := g.S.Data
	if kind == "classes" {
		family := data.ClassFamily(data.ClassLine(data.Characters[s.Character].Class)[0])
		for _, f := range data.MapEditor.Families {
			if f.ID != family {
				continue
			}
			for _, c := range f.Classes {
				fn := func() {
					s.Class = c.ID
					if s.Equipment != nil {
						eq := []string{}
						cl := data.Classes[c.ID]
						for _, id := range *s.Equipment {
							v := data.Equipment[id]
							if !data.Carries(cl, data.SpawnTraits(*s), v) {
								continue
							}
							eq = append(eq, id)
						}
						s.Equipment = &eq
					}
					e.popup = ""
				}
				if _, ok := data.Classes[c.ID]; !ok {
					fn = nil
				}
				choices = append(choices, editorChoice{label: c.Name, pick: fn})
			}
		}
		g.editorChoices(dst, r, choices)
		return
	}
	ids := []string{}
	if kind == "traits" {
		base := *s
		base.Traits = nil
		ids = g.editorPreview(base).Traits
	} else {
		class := s.Class
		if class == "" {
			class = data.Characters[s.Character].Class
		}
		cl := data.Classes[class]
		for _, id := range sorted(data.Equipment) {
			v := data.Equipment[id]
			if !data.Carries(cl, data.SpawnTraits(*s), v) {
				continue
			}
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		label := ""
		checked := false
		var fn func()
		if kind == "traits" {
			label = data.Traits[id].Name
			var given []string
			eq := data.Characters[s.Character].Equipment
			if s.Equipment != nil {
				eq = *s.Equipment
			}
			for _, item := range eq {
				given = append(given, data.Equipment[item].Traits...)
			}
			inherited := slices.Contains(given, id)
			if inherited {
				checked = true
				label += " (병종/장비)"
			}
			for _, t := range *s.Traits {
				checked = checked || t == id
			}
			fn = func() {
				list := []string{}
				found := false
				for _, t := range *s.Traits {
					if t == id {
						found = true
					} else {
						list = append(list, t)
					}
				}
				if !found {
					list = append(list, id)
				}
				s.Traits = &list
			}
			if inherited {
				fn = nil
			}
		} else {
			eq := data.Equipment[id]
			label = eq.Name + " / " + map[string]string{"weapon": "무기", "armor": "방어구", "accessory": "장신구"}[eq.Slot]
			for _, t := range *s.Equipment {
				checked = checked || t == id
			}
			fn = func() {
				list := []string{}
				found := false
				for _, t := range *s.Equipment {
					if t == id {
						found = true
					}
					if data.Equipment[t].Slot != eq.Slot {
						list = append(list, t)
					}
				}
				if !found {
					list = append(list, id)
				}
				s.Equipment = &list
			}
		}
		detail := ""
		if kind == "traits" {
			detail = g.traitDesc(id)
		}
		choices = append(choices, editorChoice{label: label, detail: detail, checked: checked, pick: fn})
	}
	g.editorChoices(dst, r, choices)
}

type editorChoice struct {
	label, detail, portrait string
	checked                 bool
	pick                    func()
}

func (g *Game) editorChoices(dst *ebiten.Image, r image.Rectangle, choices []editorChoice) {
	help := ""
	if len(choices) == 0 {
		g.label(dst, "선택할 항목이 없습니다", float64(r.Min.X+12), float64(r.Min.Y+16), 14, muted)
	}
	g.listRows(dst, &g.editor.choices, r, len(choices), listRowH, func(i int, rr image.Rectangle) {
		c := choices[i]
		rowBack(dst, rr, c.checked || hovered(rr))
		x := rr.Min.X + 12
		if c.portrait != "" {
			g.portrait(dst, c.portrait, "", float64(x), float64(rr.Min.Y+8), 40)
			x += 52
		}
		label, fg := c.label, ink
		if c.checked {
			label = "✓ " + label
		}
		if c.pick == nil {
			fg = faintInk
		}
		y := rr.Min.Y + 16
		if c.detail != "" {
			y = rr.Min.Y + 6
			g.label(dst, ellipsis(g, strings.ReplaceAll(c.detail, "\n", " · "), 12, rr.Max.X-x-12), float64(x), float64(rr.Min.Y+31), 12, muted)
			if c.portrait == "" && hovered(rr) {
				help = c.label + "\n" + c.detail
			}
		}
		g.label(dst, ellipsis(g, label, 16, rr.Max.X-x-12), float64(x), float64(y), 16, fg)
	}, func(i int) {
		if choices[i].pick != nil {
			choices[i].pick()
		}
	})
	if help != "" {
		x, y := cursorPos()
		g.tooltip(dst, help, x+16, y+16)
	}
}

func editorLevel(base, override int) int {
	if override > 0 {
		return override
	}
	return base
}

func editorOfficers(data *content.Data, family, class, army, query string) []string {
	ids := []string{}
	for _, id := range sorted(data.Characters) {
		c := data.Characters[id]
		first := data.ClassLine(c.Class)[0]
		if !c.Template && data.ClassFamily(first) == family && (class == "" || first == class) && data.MapEditor.CharacterArmies[id] == army && strings.Contains(c.Name, query) {
			ids = append(ids, id)
		}
	}
	return ids
}

func routVictory() []content.Goal {
	return []content.Goal{{If: []string{"alldefeated:enemy"}, Text: "적군 전멸"}}
}

func bossVictory(g *Game, boss string) []content.Goal {
	if boss == "" {
		return []content.Goal{{If: []string{"alldefeated:enemy"}, Text: "적군 전멸"}}
	}
	return []content.Goal{{If: []string{"defeated:" + boss}, Text: g.name(boss) + " 격파"}}
}

// brushName is what a brush paints: a layer and a glyph, or "none" for a structure or state brush.
func brushName(d *content.Data, brush string) string {
	glyph := string(brush[1])
	switch brush[0] {
	case content.LayerGround:
		return d.Grounds[glyph].Name
	case content.LayerStructure:
		if glyph == string(content.NoLayer) {
			return "구조물 없음"
		}
		return d.Structures[glyph].Name
	}
	if glyph == string(content.NoLayer) {
		return "상태 없음"
	}
	return d.States[glyph].Name
}
