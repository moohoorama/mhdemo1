package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"srpg/internal/core"
	"srpg/internal/session"
	"strconv"
	"strings"
	"time"
)

const Help = "next | choose 선택ID | deploy 장수... | equip/unequip 장수 장비ID | start\nmove 장수 x y | attack 장수 적ID | skill 장수 병법 대상 | item 장수 아이템 대상\ndeputy 주장 부관 | undeputy 주장\nlearn 장수 특성 | wait 장수 | end | ai(적 한 명령) | auto(현재 진영)\nmap | status | legal 장수 | preview attack 장수 대상 | save/load 슬롯 | slots\nsettings [ai auto/step | color true/false | detail true/false | speed 밀리초]\nmenu | resume | title | retry | quit"

func Parse(line string) (core.Command, error) {
	p := strings.Fields(line)
	if len(p) == 0 {
		return core.Command{}, fmt.Errorf("EmptyCommand")
	}
	c := core.Command{Kind: p[0]}
	argc := func(n int) error {
		if len(p) != n {
			return fmt.Errorf("Usage: %s", Help)
		}
		return nil
	}
	switch c.Kind {
	case "next", "start", "end", "continue":
		if err := argc(1); err != nil {
			return c, err
		}
	case "choose":
		if err := argc(2); err != nil {
			return c, err
		}
		c.Option = p[1]
	case "deploy":
		if len(p) < 2 {
			return c, fmt.Errorf("InvalidDeployment")
		}
		c.Deployment = p[1:]
	case "wait", "undeputy":
		if err := argc(2); err != nil {
			return c, err
		}
		c.Actor = p[1]
	case "attack", "deputy":
		if err := argc(3); err != nil {
			return c, err
		}
		c.Actor = p[1]
		c.Target = p[2]
	case "equip", "unequip", "learn":
		if err := argc(3); err != nil {
			return c, err
		}
		c.Actor = p[1]
		c.Item = p[2]
		c.Trait = p[2]
	case "move":
		if err := argc(4); err != nil {
			return c, err
		}
		c.Actor = p[1]
		var err error
		c.X, err = strconv.Atoi(p[2])
		if err != nil {
			return c, err
		}
		c.Y, err = strconv.Atoi(p[3])
		if err != nil {
			return c, err
		}
	case "skill", "item":
		if err := argc(4); err != nil {
			return c, err
		}
		c.Actor = p[1]
		c.Skill = p[2]
		c.Item = p[2]
		c.Target = p[3]
	default:
		return c, fmt.Errorf("UnknownCommand")
	}
	return c, nil
}
func RunJSON(s *session.Session, in io.Reader, out io.Writer) error {
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	enc := json.NewEncoder(out)
	for scan.Scan() {
		var q session.Request
		if err := json.Unmarshal(scan.Bytes(), &q); err != nil {
			if err = enc.Encode(session.Response{OK: false, Error: "InvalidJSON"}); err != nil {
				return err
			}
			continue
		}
		r := s.Handle(q)
		if err := enc.Encode(r); err != nil {
			return err
		}
		if r.OK && s.Screen == "quit" {
			return nil
		}
	}
	if err := scan.Err(); err != nil {
		return err
	}
	return nil
}
func show(s *session.Session, out io.Writer) {
	if s.Settings.Color {
		fmt.Fprint(out, "\x1b[36m")
	}
	defer func() {
		if s.Settings.Color {
			fmt.Fprint(out, "\x1b[0m")
		}
	}()
	if s.Screen == "title" {
		fmt.Fprintln(out, "1. 시작하기\n2. 불러오기 (load 슬롯)\n3. 설정 (settings)\n4. 종료")
		return
	}
	if s.Screen == "menu" {
		fmt.Fprintln(out, "계속하기 resume · 저장 save · 불러오기 load · 설정 settings · 시작 메뉴 title · 종료 quit")
		return
	}
	if s.Engine == nil {
		return
	}
	o := s.Engine.Observe()
	fmt.Fprintf(out, "[%s revision=%d]", o.Phase, o.Revision)
	switch o.Phase {
	case "scenario":
		fmt.Fprintln(out)
		for _, r := range o.Dialogue.Text {
			fmt.Fprint(out, string(r))
			if s.Settings.TextDelayMS > 0 {
				time.Sleep(time.Duration(s.Settings.TextDelayMS) * time.Millisecond)
			}
		}
		fmt.Fprintln(out)
		if o.Dialogue.Kind == "choice" {
			fmt.Fprintln(out, "선택: choose 결의")
		}
		if o.Dialogue.Kind == "preparation" {
			fmt.Fprintln(out, "편성:", o.Deployment, "창고:", o.Warehouse, "→ start")
		}
	case "battle":
		fmt.Fprintf(out, " %s / 라운드 %d / %s 차례\n", o.Map.Name, o.Round, o.Turn)
		Map(o, out)
	case "result":
		fmt.Fprintf(out, " %s — continue / retry / load 슬롯\n", o.Result)
	case "complete":
		fmt.Fprintln(out, " 호로관 전투 승리 — 초반 캠페인 완료")
	}
}
func Map(o core.Observation, out io.Writer) {
	if o.Map == nil {
		return
	}
	fmt.Fprint(out, "   ")
	for x := 0; x < o.Map.Width; x++ {
		fmt.Fprintf(out, "%2d", x)
	}
	fmt.Fprintln(out)
	glyph := map[[2]int]string{}
	for i, u := range o.UnitViews {
		if u.HP > 0 {
			g := fmt.Sprint(i % 10)
			if u.Faction == "ally" {
				g = string(rune('A' + i%26))
			}
			glyph[[2]int{u.X, u.Y}] = g
		}
	}
	for y, row := range o.Map.Tiles {
		fmt.Fprintf(out, "%2d ", y)
		for x, v := range row {
			g := string(v)
			if t, ok := glyph[[2]int{x, y}]; ok {
				g = t
			}
			fmt.Fprintf(out, "%2s", g)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, ".초원 d황무지 f숲 s산 c성 i성내 v마을 ~물 / 좌표는 0부터")
	for i, u := range o.UnitViews {
		if u.HP <= 0 {
			continue
		}
		g := fmt.Sprint(i % 10)
		if u.Faction == "ally" {
			g = string(rune('A' + i%26))
		}
		fmt.Fprintf(out, "%s %s(%s) Lv%d EXP %.1f/100 (%d,%d) HP %d/%d MP %d/%d 이동:%t 행동:%t\n", g, u.ID, u.Name, u.Level, u.XPProgress, u.X, u.Y, u.HP, u.Stats.MaxHP, u.MP, u.Stats.MaxMP, u.Moved, u.Acted)
	}
}
func RunHuman(s *session.Session, in io.Reader, out io.Writer, seed uint64) error {
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	show(s, out)
	ask := func(prompt string) bool {
		fmt.Fprint(out, prompt, " [y/N] ")
		return scan.Scan() && strings.EqualFold(strings.TrimSpace(scan.Text()), "y")
	}
	for {
		fmt.Fprint(out, "> ")
		if !scan.Scan() {
			break
		}
		line := strings.TrimSpace(scan.Text())
		p := strings.Fields(line)
		if len(p) == 0 {
			continue
		}
		q := session.Request{Op: p[0]}
		if s.Engine != nil {
			rev := s.Engine.Observe().Revision
			q.Revision = &rev
		}
		switch p[0] {
		case "help":
			fmt.Fprintln(out, Help)
			continue
		case "1", "new":
			q.Op = "new"
			q.Seed = seed
		case "2":
			fmt.Fprintln(out, "저장 슬롯:", s.Store.Slots(), "— load 슬롯")
			continue
		case "3":
			q.Op = "settings"
		case "4":
			q.Op = "quit"
		case "map":
			if s.Engine != nil {
				Map(s.Engine.Observe(), out)
			}
			continue
		case "status":
			if s.Engine != nil {
				b, _ := json.MarshalIndent(s.Engine.Observe(), "", "  ")
				fmt.Fprintln(out, string(b))
			}
			continue
		case "save", "load":
			if len(p) != 2 {
				fmt.Fprintln(out, "save/load 슬롯(1~10, auto)")
				continue
			}
			q.Slot = p[1]
		case "legal":
			if len(p) != 2 {
				fmt.Fprintln(out, "legal 장수")
				continue
			}
			q.Actor = p[1]
		case "preview":
			c, err := Parse(strings.TrimPrefix(line, "preview "))
			if err != nil {
				fmt.Fprintln(out, err)
				continue
			}
			q.Command = c
		case "settings":
			if len(p) == 1 {
				fmt.Fprintf(out, "%+v\nsettings ai auto/step | color true/false | detail true/false | speed 밀리초\n", s.Settings)
				continue
			}
			if len(p) != 3 {
				fmt.Fprintln(out, "InvalidSettings")
				continue
			}
			v := s.Settings
			var err error
			switch p[1] {
			case "ai":
				v.AI = p[2]
			case "color":
				v.Color, err = strconv.ParseBool(p[2])
			case "detail":
				v.Detail, err = strconv.ParseBool(p[2])
			case "speed":
				v.TextDelayMS, err = strconv.Atoi(p[2])
			default:
				err = fmt.Errorf("InvalidSettings")
			}
			if err != nil {
				fmt.Fprintln(out, err)
				continue
			}
			q.Settings = &v
		case "ai", "auto", "retry", "slots", "menu", "resume", "title", "quit":
		default:
			c, err := Parse(line)
			if err != nil {
				fmt.Fprintln(out, err)
				continue
			}
			q.Op = "command"
			q.Command = c
		}
		if q.Op == "save" && s.Store.Exists(q.Slot) {
			if !ask("기존 슬롯을 덮어쓸까요?") {
				continue
			}
			q.Overwrite = true
		}
		if (q.Op == "title" || q.Op == "quit" || q.Op == "load" || q.Op == "new") && s.Dirty {
			fmt.Fprint(out, "미저장 진행: save(자동 슬롯에 저장) / discard(포기) / cancel: ")
			if !scan.Scan() {
				break
			}
			switch strings.TrimSpace(scan.Text()) {
			case "save":
				r := s.Handle(session.Request{Op: "save", Slot: "auto", Overwrite: true})
				if !r.OK {
					fmt.Fprintln(out, r.Error)
					continue
				}
			case "discard":
				q.Discard = true
			default:
				continue
			}
		}
		r := s.Handle(q)
		if !r.OK {
			fmt.Fprintln(out, r.Error)
			continue
		}
		if r.Options != nil {
			for _, c := range r.Options.Commands {
				b, _ := json.Marshal(c)
				fmt.Fprintln(out, string(b))
			}
		}
		if r.Preview != nil {
			fmt.Fprintf(out, "%+v\n", *r.Preview)
		}
		if q.Op == "slots" {
			fmt.Fprintln(out, "저장 슬롯:", r.Slots)
		}
		if s.Settings.Detail {
			for _, v := range r.Events {
				fmt.Fprintf(out, "%s %s → %s %d\n", v.Kind, v.Actor, v.Target, v.Amount)
			}
		}
		if s.Screen == "quit" {
			return nil
		}
		if s.Screen == "playing" {
			events, err := s.AdvanceEnemies()
			if err != nil {
				fmt.Fprintln(out, err)
			}
			if s.Settings.Detail {
				for _, v := range events {
					fmt.Fprintf(out, "%s %s → %s %d\n", v.Kind, v.Actor, v.Target, v.Amount)
				}
			}
		}
		show(s, out)
	}
	if err := scan.Err(); err != nil {
		return err
	}
	if s.Engine != nil && s.Dirty {
		r := s.Handle(session.Request{Op: "save", Slot: "auto", Overwrite: true})
		if !r.OK {
			return fmt.Errorf("EOF autosave: %s", r.Error)
		}
	}
	return nil
}
