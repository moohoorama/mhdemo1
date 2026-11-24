package scenegraph

import (
	"fmt"
	"slices"
	"strings"

	"srpg/internal/content"
)

// describe puts the Korean reading of every condition into the graph, and gives each event the subtitle that tells
// what starts it.
func (b *builder) describe() {
	title := map[string]string{}
	for _, v := range b.g.Vertices {
		title[v.ID] = v.Title
	}
	setter := map[string]string{}
	for k, vs := range b.sets {
		if len(vs) > 0 {
			setter[k] = title[uniq(vs)[0]]
		}
	}
	b.g.Texts = map[string]string{}
	for _, e := range b.g.Edges {
		for _, c := range e.Cond {
			if _, ok := b.g.Texts[c]; !ok {
				b.g.Texts[c] = b.condText(c, setter)
			}
		}
	}
	trigger := map[string]string{}
	for _, e := range b.g.Edges {
		if _, ok := trigger[e.To]; ok {
			continue
		}
		switch e.Kind {
		case EdgeTalk:
			trigger[e.To] = strings.TrimSuffix(e.Label, " (곁가지)") + "에게 말 걸면"
			if e.Side {
				trigger[e.To] += " (곁가지)"
			}
		case EdgeGate:
			trigger[e.To] = e.Label + " 출구를 지나면"
		case EdgeOpen:
			trigger[e.To] = "탐험을 시작하면"
		case EdgeSort:
			trigger[e.To] = e.Label
		}
	}
	for i, v := range b.g.Vertices {
		if v.Kind != KindEvent {
			continue
		}
		lines := strings.TrimPrefix(v.Sub[strings.Index(v.Sub, "·")+len("·"):], " ")
		if t := trigger[v.ID]; t != "" {
			b.g.Vertices[i].Sub = t + " · " + lines
		} else {
			b.g.Vertices[i].Sub = lines
		}
	}
}

// condText reads a condition in Korean: a flag by the event that sets it, a character by name.
func (b *builder) condText(s string, setter map[string]string) string {
	c, err := content.ParseCond(s)
	if err != nil {
		return s
	}
	return b.condReading(c, setter)
}

func (b *builder) char(id string) string {
	if c, ok := b.d.Characters[id]; ok && c.Name != "" {
		return c.Name
	}
	return id
}

func (b *builder) condReading(c content.Cond, setter map[string]string) string {
	var t string
	switch c.Op {
	case "flag":
		name := setter[c.Arg]
		switch {
		case name != "" && c.Not:
			t = "「" + name + "」 전"
		case name != "":
			t = "「" + name + "」 이후"
		case slices.Contains(b.opt.Assume, c.Arg) && c.Not:
			t = "앞 장의 " + c.Arg + " 이전 (가정)"
		case slices.Contains(b.opt.Assume, c.Arg):
			t = "앞 장에서 끝낸 일 " + c.Arg + " (가정)"
		default:
			t = "플래그 " + c.Arg
			if c.Not {
				t += " 없음"
			}
		}
		return t
	case "alive":
		t = b.char(c.Arg) + "이(가) 살아 있음"
	case "dead":
		t = b.char(c.Arg) + "이(가) 죽음"
	case "party":
		t = b.char(c.Arg) + "이(가) 일행에 있음"
	case "won":
		t = b.stageLabel(c.Arg) + " 승리 후"
	case "choice":
		t = fmt.Sprintf("선택 %s = %s", c.Arg, c.Arg2)
	case "any":
		var parts []string
		for _, a := range c.Any {
			parts = append(parts, b.condReading(a, setter))
		}
		t = strings.Join(parts, " 또는 ")
	default:
		t = c.Op + ":" + c.Arg
	}
	if c.Not {
		t = "아님: " + t
	}
	return t
}

func (b *builder) stageLabel(id string) string {
	if i := b.d.StageIndex(id); i >= 0 {
		return b.d.Stages[i].Name
	}
	return id
}
