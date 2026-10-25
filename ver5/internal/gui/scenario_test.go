package gui

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"srpg/internal/content"
	"srpg/internal/scenes"
	"srpg/internal/session"
	"srpg/internal/sprite"
)

func TestSplitLines(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []line
	}{
		{"title as speaker", "도원결의: 의용군을 일으킨다.", []line{{speakers: []string{"도원결의"}, text: "의용군을 일으킨다."}}},
		{"plain", "황건적의 난 — 출전 준비", []line{{text: "황건적의 난 — 출전 준비"}}},
		{"two speakers", "유비: 뜻을 세웁시다. 관우·장비: 함께하겠습니다.", []line{
			{speakers: []string{"유비"}, text: "뜻을 세웁시다."},
			{speakers: []string{"관우", "장비"}, text: "함께하겠습니다."},
		}},
		{"narration then speaker", "동탁 타도군 일어나다. 유비: 나아갑시다.", []line{
			{text: "동탁 타도군 일어나다."},
			{speakers: []string{"유비"}, text: "나아갑시다."},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := splitLines(c.text); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestNodeLinesBySpeakerFieldNotByText(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{S: &session.Session{Data: d}}
	say := g.nodeLines(content.Node{Kind: "say", Speaker: "guanyu·zhangfei", Text: "생사를 함께하겠습니다. 유비: 이것은 화자 표시가 아닙니다."})
	if want := []line{{speakers: []string{"관우", "장비"}, text: "생사를 함께하겠습니다. 유비: 이것은 화자 표시가 아닙니다."}}; !reflect.DeepEqual(say, want) {
		t.Fatalf("say: %+v", say)
	}
	if got := g.nodeLines(content.Node{Kind: "narrate", Text: "동탁 타도군 일어나다."}); !reflect.DeepEqual(got, []line{{text: "동탁 타도군 일어나다."}}) {
		t.Fatalf("narrate: %+v", got)
	}
	if got := g.nodeLines(content.Node{Kind: "choice", Text: "도원결의: 의용군을 일으킨다."}); !reflect.DeepEqual(got, []line{{speakers: []string{"도원결의"}, text: "의용군을 일으킨다."}}) {
		t.Fatalf("choice: %+v", got)
	}
}

func TestWrapAndParticle(t *testing.T) {
	t.Run("wrap at spaces", func(t *testing.T) {
		if got := wrap("이번 턴 적 옆을 지나가도 멈추지 않습니다", 12); got != "이번 턴 적 옆을\n지나가도 멈추지\n않습니다" {
			t.Fatalf("%q", got)
		}
	})
	t.Run("particle", func(t *testing.T) {
		for name, want := range map[string]string{"장비": "장비와", "관우": "관우와", "여포": "여포와", "화웅": "화웅과"} {
			if got := with(name); got != want {
				t.Fatalf("%s: %s", name, got)
			}
		}
	})
}

func TestScenesData(t *testing.T) {
	book, err := scenes.Load("../../assets/scenes")
	if err != nil {
		t.Fatal(err)
	}
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	sheets, err := sprite.LoadDir("../../assets/sprites")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../assets/graphics/index.json")
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Props, Extra struct{ Sprites map[string]json.RawMessage }
	}
	if err := json.Unmarshal(b, &index); err != nil {
		t.Fatal(err)
	}
	names := nameIndex(d)
	diagonal := map[string]bool{"NW": true, "NE": true, "SW": true, "SE": true}
	times := map[string]bool{"": true, "day": true, "dusk": true, "night": true, "dawn": true, "memory": true}
	effects := map[string]bool{"": true, "none": true, "petals": true, "fire": true, "smallfire": true, "bigfire": true, "smoke": true, "steam": true, "birds": true, "glint": true}
	for id, def := range book.Scenes {
		t.Run(id, func(t *testing.T) {
			if !times[def.Time] {
				t.Errorf("time %q", def.Time)
			}
			for key, a := range def.Cast {
				if !a.Stay && (len(a.At) != 2 || !diagonal[a.Face]) {
					t.Errorf("%s: needs At [u, v] and a diagonal Face, has %v %q", key, a.At, a.Face)
				}
				if a.Faction != "" {
					if _, ok := d.Factions[a.Faction]; !ok {
						t.Errorf("%s: unknown faction %q", key, a.Faction)
					}
				}
				if a.Sprite != "" && sheets[a.Sprite] == nil {
					t.Errorf("%s: unknown sprite %q", key, a.Sprite)
				}
				if _, ok := names[scenes.Base(key)]; !ok && a.Sprite == "" && !a.Stay { // a walk's person brings its own
					t.Errorf("%s: not a character name and no sprite given", key)
				}
			}
			// the sheet an actor stands in, as sceneArt picks it, so a pose names a motion the sheet has
			sheetOf := func(key, pick string) *sprite.Sheet {
				c := d.Characters[names[scenes.Base(key)]]
				class := d.Classes[c.Class]
				for _, id := range []string{pick, c.Look.Scene, class.Scene, c.Look.Sprites[c.Class], c.Look.Sprite, class.Sprite} {
					if id != "" && sheets[id] != nil {
						return sheets[id]
					}
				}
				return nil
			}
			pose := func(key, pick, name string) {
				if name == "" || name == "idle" {
					return
				}
				if sh := sheetOf(key, pick); sh == nil || sh.Animations[name].Frames == 0 {
					t.Errorf("%s: pose %q is not a motion of its sprite", key, name)
				}
			}
			for key, a := range def.Cast {
				pose(key, a.Sprite, a.Pose)
			}
			inScene := map[string]bool{}
			for _, n := range def.Nodes {
				inScene[n] = true
			}
			check := func(st scenes.Step) {
				if st.Pose != "" {
					pose(st.Actor, def.Cast[st.Actor].Sprite, st.Pose)
				}
				if st.Actor != "" {
					if _, ok := def.Cast[st.Actor]; !ok {
						t.Errorf("step for %q, who is not in the cast", st.Actor)
					}
				}
				for _, dir := range []string{st.Face, st.Look} {
					if dir != "" && !diagonal[dir] {
						t.Errorf("%s: %q is not a diagonal", st.Actor, dir)
					}
				}
				if !effects[st.Effect] || !times[st.Time] {
					t.Errorf("%s: effect %q or time %q", st.Actor, st.Effect, st.Time)
				}
				if st.Sprite != "" && sheets[st.Sprite] == nil {
					t.Errorf("%s: unknown sprite %q", st.Actor, st.Sprite)
				}
				if len(st.Prop) > 0 {
					name, _ := st.Prop[0].(string)
					if index.Props.Sprites[name] == nil && index.Extra.Sprites[name] == nil || len(st.Prop) != 1 && len(st.Prop) != 3 {
						t.Errorf("prop %v: unknown name or not [name, u, v] / [name]", st.Prop)
					}
				}
			}
			for _, st := range def.Ambient {
				check(st)
			}
			for node, lines := range def.Beats {
				if !inScene[node] {
					t.Errorf("beats for %s, which is not one of the scene's nodes", node)
				}
				for _, line := range lines {
					for _, group := range line {
						for _, st := range group {
							check(st)
						}
					}
				}
			}
		})
	}
}

func TestCharacterArt(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	sheets, err := sprite.LoadDir("../../assets/sprites")
	if err != nil {
		t.Fatal(err)
	}
	for id, c := range d.Characters {
		sprites := []string{c.Look.Sprite, c.Look.Scene}
		for _, s := range c.Look.Sprites {
			sprites = append(sprites, s)
		}
		for _, s := range sprites {
			if s != "" {
				if sheets[s] == nil {
					t.Errorf("%s: unknown sprite %q", id, s)
				}
			}
		}
		if c.Look.Portrait != "" {
			if _, err := os.Stat("../../assets/graphics/portraits/" + c.Look.Portrait + ".png"); err != nil {
				t.Errorf("%s: %v", id, err)
			}
		}
		if c.Look.Faction != "" {
			if _, ok := d.Factions[c.Look.Faction]; !ok {
				t.Errorf("%s: unknown faction %q", id, c.Look.Faction)
			}
		}
		if sheets[d.Classes[c.Class].Sprite] == nil {
			t.Errorf("%s: unknown class sprite", id)
		}
	}
}

func TestStatusLabels(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for id := range content.Statuses {
		if statusLabels[id] == "" {
			t.Errorf("status %s has no label", id)
		}
	}
	for id, s := range d.Skills {
		if skillSummary(s) == "" {
			t.Errorf("skill %s has no summary", id)
		}
	}
}

func TestMotion(t *testing.T) {
	scene := &sprite.Art{Animations: map[string]sprite.Animation{"idle": {}, "walk": {}, "action": {}}}
	combat := &sprite.Art{Animations: map[string]sprite.Animation{"idle": {}, "walk": {}, "attack": {}, "exhausted": {}}}
	for play, want := range map[string]string{"action": "action", "salute": "action", "toast": "action", "talk": "action", "exhausted": "action", "": ""} {
		if got := motion(scene, play); got != want {
			t.Errorf("%q: got %q, want %q", play, got, want)
		}
	}
	check(t, motion(combat, "salute"), "")
	check(t, motion(combat, "attack"), "attack")
	check(t, motion(combat, "exhausted"), "exhausted")
}
