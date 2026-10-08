package gui

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
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
	f := loadScenes("../../assets/scenes.json")
	if f == nil {
		t.Fatal("assets/scenes.json does not load")
	}
	b, err := os.ReadFile("../../assets/graphics/index.json")
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Factions struct{ Factions []struct{ ID string } }
		Units    map[string]json.RawMessage
		Props    struct{ Sprites map[string]json.RawMessage }
	}
	if err := json.Unmarshal(b, &index); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../assets/scenes.json")
	if err != nil {
		t.Fatal(err)
	}
	var maps struct {
		Maps []struct {
			ID    string
			Props [][]any
		}
	}
	if err := json.Unmarshal(raw, &maps); err != nil {
		t.Fatal(err)
	}
	for _, m := range maps.Maps {
		for _, p := range m.Props {
			if name, _ := p[0].(string); index.Props.Sprites[name] == nil || (len(p) != 3 && len(p) != 5) {
				t.Errorf("%s: prop %v: unknown name or not [name, x, y] / [name, x0, y0, x1, y1]", m.ID, p)
			}
		}
	}
	factions := map[string]bool{}
	for _, fa := range index.Factions.Factions {
		factions[fa.ID] = true
	}
	diagonal := map[string]bool{"NW": true, "NE": true, "SW": true, "SE": true}
	for node, def := range f.Scenes {
		t.Run(node, func(t *testing.T) {
			for name, at := range def.Cast {
				if len(at) < 3 {
					t.Errorf("%s: cast %v needs [x, y, facing]", name, at)
					continue
				}
				if dir, _ := at[2].(string); !diagonal[dir] {
					t.Errorf("%s: facing %v is not a diagonal", name, at[2])
				}
				if len(at) > 3 {
					if fa, _ := at[3].(string); !factions[fa] {
						t.Errorf("%s: unknown faction %v", name, at[3])
					}
				}
				sprite := ""
				if len(at) > 4 {
					sprite, _ = at[4].(string)
					if index.Units[sprite] == nil {
						t.Errorf("%s: unknown sprite %q", name, sprite)
					}
				}
				if _, ok := officers[name]; !ok && sprite == "" {
					t.Errorf("%s: not in the officers table and no sprite given", name)
				}
			}
			for _, line := range def.Beats {
				for _, group := range line {
					for _, st := range group {
						if st.Face != "" && !diagonal[st.Face] {
							t.Errorf("%s: face %q is not a diagonal", st.Actor, st.Face)
						}
					}
				}
			}
		})
	}
}
