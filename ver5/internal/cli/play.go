package cli

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// RunHuman plays the game as numbered menus: every screen ends with its choices, the answer is
// the choice's number (or what the screen asks for, such as "x,y"). The story shows its scenes as
// letter maps and its lines one by one; a battle shows the map, the player's officers, what each
// may do, and the targets with their numbers before the blow. Raw commands (help) still work.
func RunHuman(s *session.Session, contentDir string, in io.Reader, out io.Writer, seed uint64) error {
	p := &player{s: s, d: s.Data, out: out, seed: seed, scenes: loadScenes(contentDir)}
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	for !p.quit {
		p.show()
		fmt.Fprint(out, "> ")
		if !scan.Scan() {
			fmt.Fprintln(out)
			break
		}
		p.answer(strings.TrimSpace(scan.Text()))
	}
	return scan.Err()
}

type choice struct {
	label string
	do    func()
}

type player struct {
	s       *session.Session
	d       *content.Data
	out     io.Writer
	seed    uint64
	scenes  *sceneBook
	stage   stage
	quit    bool
	screen  string // title, load, settings, unit, move, attack, skill, item; empty follows the game
	actor   string // the officer whose orders are being given
	choices []choice
	ask     func(string) bool // a screen that takes free input, such as "x,y"; false when it was not understood
	shown   string            // the dialogue node last written, so a line is written once
}

func (p *player) printf(format string, a ...any) { fmt.Fprintf(p.out, format, a...) }

func (p *player) add(label string, do func()) { p.choices = append(p.choices, choice{label, do}) }

func (p *player) obs() core.Observation { return p.s.Engine.Observe() }

// request sends one request to the session and writes what came of it.
func (p *player) request(q session.Request) bool {
	if p.s.Engine != nil {
		rev := p.obs().Revision
		q.Revision = &rev
	}
	r := p.s.Handle(q)
	if !r.OK {
		p.printf("  ✗ %s\n", r.Error)
		return false
	}
	p.events(r.Events)
	return true
}

func (p *player) command(c core.Command) bool {
	return p.request(session.Request{Op: "command", Command: c})
}

// answer runs the numbered choice, the screen's free input, or a raw command.
func (p *player) answer(line string) {
	if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(p.choices) {
		p.choices[n-1].do()
		return
	}
	switch {
	case line == "" && len(p.choices) > 0 && p.ask == nil:
		p.choices[0].do()
	case line == "0" || line == "b":
		p.screen, p.actor = "", ""
	case line == "help":
		p.printf("%s\n숫자: 선택 · 0: 뒤로 · map: 지도 · quit: 끝\n", Help)
	case line == "map" && p.s.Engine != nil:
		Board(p.d, p.obs(), p.out, nil)
	case line == "quit" || line == "q":
		p.quit = true
	case p.ask != nil && p.ask(line):
	case p.s.Engine != nil:
		c, err := Parse(line)
		if err != nil {
			p.printf("  ? %s (help)\n", line)
			return
		}
		p.command(c)
	default:
		p.printf("  ? %s\n", line)
	}
}

// show writes the screen and its choices.
func (p *player) show() {
	p.choices, p.ask = nil, nil
	switch {
	case p.screen == "load":
		p.loadScreen()
	case p.screen == "settings":
		p.settingsScreen()
	case p.s.Engine == nil || p.s.Screen == "title":
		p.titleScreen()
	default:
		p.gameScreen()
	}
	for i, c := range p.choices {
		p.printf("  (%d) %s\n", i+1, c.label)
	}
}

func (p *player) titleScreen() {
	p.printf("\n== %s ==\n", p.d.Title)
	p.add("시작하기", func() { p.request(session.Request{Op: "new", Seed: p.seed, Discard: true}) })
	p.add("불러오기", func() { p.screen = "load" })
	p.add("설정", func() { p.screen = "settings" })
	p.add("종료", func() { p.quit = true })
}

func (p *player) loadScreen() {
	slots := p.s.Store.Slots()
	p.printf("\n불러오기 (0: 뒤로)\n")
	for _, slot := range slots {
		p.add("슬롯 "+slot, func() {
			if p.request(session.Request{Op: "load", Slot: slot, Discard: true}) {
				p.screen = ""
			}
		})
	}
	if len(slots) == 0 {
		p.printf("  저장된 슬롯이 없습니다\n")
	}
}

func (p *player) settingsScreen() {
	v := p.s.Settings
	p.printf("\n설정 (0: 뒤로)\n")
	set := func(f func(*session.Request)) {
		q := session.Request{Op: "settings", Settings: &v}
		f(&q)
		p.request(q)
	}
	p.add(fmt.Sprintf("전투 기록 자세히: %t", v.Detail), func() { set(func(*session.Request) { v.Detail = !v.Detail }) })
	p.add("돌아가기", func() { p.screen = "" })
}

func (p *player) gameScreen() {
	o := p.obs()
	switch o.Phase {
	case "scenario":
		p.storyScreen(o)
	case "deployment":
		p.printf("\n== %s · 배치 ==\n", o.Map.Name)
		Board(p.d, o, p.out, cells(o.Map.Allies))
		p.add("자동 배치하고 시작", func() { p.request(session.Request{Op: "auto"}) })
	case "battle":
		p.battleScreen(o)
	case "result":
		p.printf("\n== %s: %s (라운드 %d) ==\n", o.Map.Name, map[string]string{"victory": "승리", "defeat": "패배"}[o.Result], o.Round)
		if len(o.Learning) > 0 {
			p.learningScreen(o)
			return
		}
		p.add("계속", func() { p.command(core.Command{Kind: "continue"}) })
		p.add("다시 하기", func() { p.request(session.Request{Op: "retry"}) })
	case "complete":
		p.printf("\n== 캠페인 완료 ==\n")
		p.add("시작 화면", func() { p.request(session.Request{Op: "title", Discard: true}) })
	}
}

func cells(ps []content.Point) map[content.Point]bool {
	out := map[content.Point]bool{}
	for _, p := range ps {
		out[p] = true
	}
	return out
}

func (p *player) storyScreen(o core.Observation) {
	n := o.Dialogue
	if p.shown != o.Node {
		p.shown = o.Node
		if n.Kind == "say" || n.Kind == "narrate" {
			p.scenes.sceneFor(p.d, &p.stage, o.Node, o.Place, p.out)
		}
		switch n.Kind {
		case "say":
			p.printf("\n%s: %s\n", speakerNames(p.d, n.Speaker), n.Text)
		case "narrate", "choice":
			p.printf("\n  %s\n", n.Text)
		case "battle":
			p.printf("\n== 출전 준비: %s ==\n", p.d.Stages[o.Stage].Name)
		}
	}
	switch n.Kind {
	case "say", "narrate":
		p.add("다음 (엔터)", func() { p.command(core.Command{Kind: "next"}) })
	case "choice":
		keys := make([]string, 0, len(n.Choices))
		for k := range n.Choices {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p.add(k, func() { p.command(core.Command{Kind: "choose", Option: k}) })
		}
	case "explore":
		p.exploreScreen(o)
	case "battle":
		var names []string
		for _, id := range o.Deployment {
			names = append(names, p.d.Characters[id].Name)
		}
		p.printf("편성: %s\n", strings.Join(names, ", "))
		p.add("출전", func() { p.command(core.Command{Kind: "start"}) })
	default:
		p.add("다음", func() { p.command(core.Command{Kind: "next"}) })
	}
}

func (p *player) learningScreen(o core.Observation) {
	l := o.Learners[0]
	p.printf("\n%s 특성 배우기 (특성치 %d)\n", l.Name, l.Points)
	for _, t := range l.Options {
		tr := p.d.Traits[t]
		p.add(fmt.Sprintf("%s (%d) %s", tr.Name, tr.Cost, tr.Desc), func() {
			p.command(core.Command{Kind: "learn", Actor: l.ID, Trait: t})
		})
	}
	p.add("닫기", func() { p.command(core.Command{Kind: "learn_close", Actor: l.ID}) })
}

func (p *player) battleScreen(o core.Observation) {
	if len(o.Learning) > 0 {
		p.learningScreen(o)
		return
	}
	if o.Turn != "ally" {
		p.printf("\n— %s 차례 —\n", p.armyName(o.Map, o.Turn, o.Army))
		p.request(session.Request{Op: "auto"})
		p.show()
		return
	}
	switch p.screen {
	case "move":
		p.moveScreen(o)
		return
	case "attack", "skill", "item":
		p.targetScreen(o, p.screen)
		return
	case "unit":
		p.unitScreen(o)
		return
	}
	p.printf("\n== %s · 라운드 %d · 우리 차례 ==\n", o.Map.Name, o.Round)
	abbr := Board(p.d, o, p.out, nil)
	p.printf("적·동맹:\n")
	roster(p.out, o, abbr, func(u core.UnitView) bool { return u.Faction != "ally" || u.AI })
	p.printf("목표: %s\n", goals(o.Map.Victory))
	for _, u := range o.UnitViews {
		if u.Faction != "ally" || u.AI || u.HP <= 0 || u.Done {
			continue
		}
		p.add(fmt.Sprintf("%s %s (%d,%d) HP %d/%d MP %d/%d%s", abbr[u.ID], u.Name, u.X, u.Y, u.HP, u.Stats.MaxHP, u.MP, u.Stats.MaxMP, map[bool]string{true: " [이동함]"}[u.Moved]), func() {
			p.screen, p.actor = "unit", u.ID
		})
	}
	p.add("턴 종료", func() { p.command(core.Command{Kind: "end"}) })
}

// armyName is what a turn's army is called: by the stage's Armies, else by its side.
func (p *player) armyName(m *content.Stage, turn, army string) string {
	if m != nil {
		for _, a := range m.Armies {
			if a.Faction == army || turn == "ally" && a.Ally && a.Faction == m.Armies[0].Faction {
				return a.Name
			}
		}
	}
	return map[string]string{"ally": "우리", "enemy": "적군", "support": "동맹군"}[turn]
}

func goals(gs []content.Goal) string {
	var out []string
	for _, g := range gs {
		out = append(out, g.Text)
	}
	return strings.Join(out, " / ")
}

func (p *player) unit(o core.Observation, id string) *core.UnitView {
	for i := range o.UnitViews {
		if o.UnitViews[i].ID == id {
			return &o.UnitViews[i]
		}
	}
	return nil
}

func (p *player) unitScreen(o core.Observation) {
	u := p.unit(o, p.actor)
	if u == nil || u.HP <= 0 || u.Done {
		p.screen, p.actor = "", ""
		p.battleScreen(o)
		return
	}
	p.printf("\n%s Lv%d (%d,%d) HP %d/%d MP %d/%d · 공격 %d 방어 %d 정신 %d 민첩 %d 사기 %d · 특성 %s\n", u.Name, u.Level, u.X, u.Y, u.HP, u.Stats.MaxHP, u.MP, u.Stats.MaxMP, u.Stats.Attack, u.Stats.Defense, u.Stats.Mind, u.Stats.Agility, u.Stats.Morale, p.traitNames(u.Traits))
	count := map[string]int{}
	for _, c := range p.s.Engine.LegalActions(u.ID).Commands {
		count[c.Kind]++
	}
	if count["move"] > 0 {
		p.add("이동", func() { p.screen = "move" })
	}
	for _, k := range []struct{ kind, label string }{{"attack", "공격"}, {"skill", "병법"}, {"item", "도구"}} {
		if count[k.kind] > 0 {
			p.add(fmt.Sprintf("%s (%d)", k.label, count[k.kind]), func() { p.screen = k.kind })
		}
	}
	p.add("대기", func() {
		p.command(core.Command{Kind: "wait", Actor: u.ID})
		p.screen, p.actor = "", ""
	})
	p.add("뒤로", func() { p.screen, p.actor = "", "" })
}

func (p *player) traitNames(ids []string) string {
	var out []string
	for _, id := range ids {
		out = append(out, p.d.Traits[id].Name)
	}
	if len(out) == 0 {
		return "-"
	}
	return strings.Join(out, ",")
}

func (p *player) moveScreen(o core.Observation) {
	marks := map[content.Point]bool{}
	for _, c := range p.s.Engine.LegalActions(p.actor).Commands {
		if c.Kind == "move" {
			marks[content.Point{X: c.X, Y: c.Y}] = true
		}
	}
	u := p.unit(o, p.actor)
	p.printf("\n%s 이동: o 표시 칸 중 x,y 입력 (0: 뒤로)\n", u.Name)
	Board(p.d, o, p.out, marks)
	p.ask = func(line string) bool {
		f := strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == ' ' })
		if len(f) != 2 {
			return false
		}
		x, err1 := strconv.Atoi(f[0])
		y, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			return false
		}
		if !marks[content.Point{X: x, Y: y}] {
			p.printf("  ✗ (%d,%d)로는 갈 수 없습니다\n", x, y)
			return true
		}
		if p.command(core.Command{Kind: "move", Actor: p.actor, X: x, Y: y}) {
			p.printf("  ✓ (%d,%d)로 이동\n", x, y)
			p.screen = "unit"
		}
		return true
	}
}

// targetScreen lists what the selected officer may strike (or heal, or use an item on) with the
// target's numbers and the preview of the blow.
func (p *player) targetScreen(o core.Observation, kind string) {
	abbr := abbreviations(p.d, o.UnitViews)
	p.printf("\n%s 대상 (0: 뒤로)\n", map[string]string{"attack": "공격", "skill": "병법", "item": "도구"}[kind])
	for _, c := range p.s.Engine.LegalActions(p.actor).Commands {
		if c.Kind != kind {
			continue
		}
		t := p.unit(o, c.Target)
		if t == nil {
			continue
		}
		what := ""
		switch kind {
		case "skill":
			what = p.d.Skills[c.Skill].Name + " → "
		case "item":
			what = p.d.Items[c.Item].Name + " → "
		}
		pv, err := p.s.Engine.Preview(c)
		guess := ""
		if err == nil {
			switch {
			case pv.Healing > 0:
				guess = fmt.Sprintf("회복 %d", pv.Healing)
			case pv.MaxDamage > 0:
				guess = fmt.Sprintf("피해 %d~%d 명중 %.0f%% 치명 %.0f%%", pv.Damage[c.Target], pv.CritDamage[c.Target], pv.Hit, pv.Crit)
			default:
				guess = fmt.Sprintf("명중 %.0f%% %s", pv.Hit, pv.Effect)
			}
		}
		p.add(fmt.Sprintf("%s%s%s %s Lv%d (%d,%d) HP %d/%d MP %d/%d 공%d 방%d 정%d 민%d 사%d 특성 %s — %s", what, abbr[t.ID], sideMark(*t), t.Name, t.Level, t.X, t.Y, t.HP, t.Stats.MaxHP, t.MP, t.Stats.MaxMP, t.Stats.Attack, t.Stats.Defense, t.Stats.Mind, t.Stats.Agility, t.Stats.Morale, p.traitNames(t.Traits), guess), func() {
			p.command(c)
			p.screen, p.actor = "", ""
		})
	}
}

// events writes what a command came to, in words.
func (p *player) events(evs []core.Event) {
	name := func(id string) string {
		if id == "" {
			return ""
		}
		if p.s.Engine != nil {
			if u := p.unit(p.obs(), id); u != nil {
				return u.Name
			}
		}
		if c, ok := p.d.Characters[id]; ok {
			return c.Name
		}
		return id
	}
	for i, e := range evs {
		switch e.Kind {
		case "move":
			if len(e.Path) > 0 {
				end := e.Path[len(e.Path)-1]
				p.printf("  %s 이동 → (%d,%d)\n", name(e.Actor), end.X, end.Y)
			}
		case "damage", "critical":
			p.printf("  %s → %s %d 피해%s\n", name(e.Actor), name(e.Target), e.Amount, map[bool]string{true: " (치명)"}[e.Kind == "critical"])
		case "miss":
			p.printf("  %s → %s 빗나감\n", name(e.Actor), name(e.Target))
		case "heal":
			p.printf("  %s → %s %d 회복\n", name(e.Actor), name(e.Target), e.Amount)
		case "pouch-heal":
			p.printf("  %s 주머니의 콩 %d 회복\n", name(e.Actor), e.Amount)
		case "knockback":
			if len(e.Path) == 2 {
				p.printf("  %s → %s 밀어냄 (%d,%d)\n", name(e.Actor), name(e.Target), e.Path[1].X, e.Path[1].Y)
			}
		case "forage":
			p.printf("  %s 마을에서 보리·콩 획득\n", name(e.Actor))
		case "goals":
			if m := p.obs().Map; m != nil {
				for _, g := range m.Victory {
					p.printf("  ◎ 전투목표 변경 · 승리: %s\n", g.Text)
				}
				for _, o := range m.Sub {
					p.printf("  ◎ 전투목표 변경 · 부목표: %s\n", o.Text)
				}
			}
		case "retreat", "death":
			p.printf("  ✝ %s %s\n", name(e.Actor), map[string]string{"retreat": "퇴각", "death": "전사"}[e.Kind])
		case "speech":
			for _, l := range e.Speech {
				p.printf("  「%s」 %s\n", name(l.Actor), l.Text)
			}
		case "cutscene":
			for _, group := range e.Cutscene {
				for _, st := range group {
					switch st.Kind {
					case "move":
						end := st.Path[len(st.Path)-1]
						p.printf("  [컷신] %s 이동 → (%d,%d)\n", name(st.Unit), end.X, end.Y)
					case "say":
						p.printf("  [컷신] 「%s」 %s\n", name(st.Unit), st.Text)
					case "play":
						p.printf("  [컷신] %s %s\n", name(st.Unit), st.Motion)
					case "leave":
						p.printf("  [컷신] %s 퇴장\n", name(st.Unit))
					}
				}
			}
		case "duel":
			p.printf("  ⚔ 일기토 %s vs %s: %s\n", name(e.Actor), name(e.Target), e.Text)
		case "unit-arrival":
			p.printf("  + %s 등장\n", name(e.Actor))
		case "terrain":
			state := e.Target
			if st, ok := p.d.States[state]; ok {
				state = st.Name
			}
			p.printf("  지형 변화: 구역 %d → %s\n", e.Amount, state)
		case "mark":
			if len(e.Path) > 0 {
				p.printf("  지형 변화: (%d,%d) → %s\n", e.Path[0].X, e.Path[0].Y, p.d.States[e.Target].Name)
			}
		case "status": // a run of the same change is one line
			if i > 0 && evs[i-1].Kind == "status" && evs[i-1].Text == e.Text && evs[i-1].Amount == e.Amount {
				continue
			}
			var who []string
			for _, f := range evs[i:] {
				if f.Kind != "status" || f.Text != e.Text || f.Amount != e.Amount {
					break
				}
				who = append(who, name(f.Target))
			}
			p.printf("  %s %+d%%: %s\n", e.Text, e.Amount, strings.Join(who, " · "))
		case "level":
			p.printf("  ↑ %s Lv%d\n", name(e.Actor), e.Amount)
		case "turn":
			var m *content.Stage
			if p.s.Engine != nil {
				m = p.obs().Map
			}
			p.printf("  — 라운드 %d %s 차례\n", e.Amount, p.armyName(m, e.Target, e.Text))
		case "victory", "defeat":
			p.printf("  ★ %s\n", map[string]string{"victory": "승리", "defeat": "패배"}[e.Kind])
		case "objective":
			p.printf("  ◎ %s\n", e.Text)
		case "place", "deployment_finish", "experience", "cost", "recover-mp", "attack-hit", "end":
		default:
			if p.s.Settings.Detail {
				p.printf("  (%s %s %s %d)\n", e.Kind, name(e.Actor), name(e.Target), e.Amount)
			}
		}
	}
}
