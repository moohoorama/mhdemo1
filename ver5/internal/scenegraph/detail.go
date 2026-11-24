package scenegraph

import (
	"fmt"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/maplight"
)

// NodeInfo is a story node as the admin shows it.
type NodeInfo struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	Speaker   string         `json:"speaker,omitempty"`
	Text      string         `json:"text,omitempty"`
	Place     string         `json:"place,omitempty"`
	Flags     map[string]int `json:"flags,omitempty"`
	Character string         `json:"character,omitempty"`
	Item      string         `json:"item,omitempty"`
	Event     string         `json:"event,omitempty"`
	Stage     string         `json:"stage,omitempty"`
	Explore   string         `json:"explore,omitempty"`
	Opening   string         `json:"opening,omitempty"`
	Area      string         `json:"area,omitempty"`
	Leave     bool           `json:"leave,omitempty"`
	Cases     []CaseInfo     `json:"cases,omitempty"`
	Else      string         `json:"else,omitempty"`
	Speakers  []Speaker      `json:"speakers,omitempty"`
}

// Speaker is who says a line: the character's Korean name and the portrait to show (empty when there is none).
type Speaker struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Portrait string `json:"portrait,omitempty"`
}

func speakers(d *content.Data, joined string) []Speaker {
	if joined == "" {
		return nil
	}
	var out []Speaker
	for _, id := range strings.Split(joined, "·") {
		sp := Speaker{ID: id, Name: id}
		if c, ok := d.Characters[id]; ok {
			if c.Name != "" {
				sp.Name = c.Name
			}
			if !c.Look.NoPortrait {
				sp.Portrait = c.Look.Portrait
				if sp.Portrait == "" {
					sp.Portrait = id
				}
			}
		}
		out = append(out, sp)
	}
	return out
}

// CaseInfo is a branch's case.
type CaseInfo struct {
	If   []string `json:"if"`
	Next string   `json:"next"`
}

func info(d *content.Data, n content.Node) NodeInfo {
	out := NodeInfo{ID: n.ID, Kind: n.Kind, Speaker: n.Speaker, Text: n.Text, Place: n.Place, Flags: n.Flags, Character: n.Character,
		Item: n.Item, Event: n.Event, Stage: n.Stage, Explore: n.Explore, Opening: n.Opening, Area: n.Area, Leave: n.Leave, Else: n.Else}
	for _, c := range n.Cases {
		out.Cases = append(out.Cases, CaseInfo(c))
	}
	out.Speakers = speakers(d, n.Speaker)
	return out
}

// TalkLine is one line of a talk: who says it (empty for narration) and what.
type TalkLine struct {
	Speaker
	Text string `json:"text"`
}

func talkLines(d *content.Data, lines []content.Line) []TalkLine {
	var out []TalkLine
	for _, l := range lines {
		t := TalkLine{Text: l[1]}
		if l[0] != "" {
			t.Speaker = speakers(d, l[0])[0]
		}
		out = append(out, t)
	}
	return out
}

// Pixel is a place on the map's canvas.
type Pixel struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Mark is something placed on an area: an exit, a person, a sign.
type Mark struct {
	Kind   string            `json:"kind"` // exit, person, sign, arrival
	Name   string            `json:"name"`
	At     [2]int            `json:"at"`
	Px     Pixel             `json:"px"`
	Walk   bool              `json:"walk"` // the tile can be walked
	Face   string            `json:"face,omitempty"`
	If     []string          `json:"if,omitempty"`
	To     string            `json:"to,omitempty"`
	Node   string            `json:"node,omitempty"`
	Talk   []TalkLine        `json:"talk,omitempty"`
	Events []EventRef        `json:"events,omitempty"`
	Sortie string            `json:"sortie,omitempty"`
	Shop   string            `json:"shop,omitempty"`
	Shut   []TalkLine        `json:"shut,omitempty"`
	Extra  map[string]string `json:"extra,omitempty"`
	// Display is the shown name; Char the character the person is, Sprite the scene sprite sheet that draws them and
	// Portrait the portrait stem, each empty when there is none.
	Display  string   `json:"display,omitempty"`
	Char     string   `json:"char,omitempty"`
	Sprite   string   `json:"sprite,omitempty"`
	Portrait string   `json:"portrait,omitempty"`
	Chatter  []string `json:"chatter,omitempty"`
}

// EventRef is an event a person starts.
type EventRef struct {
	If   []string `json:"if,omitempty"`
	Node string   `json:"node"`
	Side bool     `json:"side,omitempty"`
}

// AreaDetail is an area for the admin: its map, the canvas it is drawn on and everything standing on it.
type AreaDetail struct {
	Kind    string         `json:"kind"`
	Explore string         `json:"explore"`
	Area    string         `json:"area"`
	Name    string         `json:"name"`
	Map     string         `json:"map"`
	Time    string         `json:"time,omitempty"`
	Start   bool           `json:"start"`
	Frame   maplight.Frame `json:"frame"`
	Marks   []Mark         `json:"marks"`
	Shops   []string       `json:"shops,omitempty"`
}

// EventDetail is a run of story nodes.
type EventDetail struct {
	Kind  string     `json:"kind"`
	Start string     `json:"start"`
	Nodes []NodeInfo `json:"nodes"`
	End   *NodeInfo  `json:"end,omitempty"`
	Sets  []string   `json:"sets,omitempty"`
}

// BattleDetail is the battle a node leads to.
type BattleDetail struct {
	Kind    string `json:"kind"`
	Node    string `json:"node"`
	Stage   any    `json:"stage"`
	Enemies int    `json:"enemies"`
	Allies  int    `json:"allies"`
}

// Detail is what a vertex holds, by its id.
func Detail(d *content.Data, env Env, id string) (any, error) {
	kind, rest, _ := strings.Cut(id, ":")
	switch kind {
	case KindArea:
		file, area, ok := strings.Cut(rest, "/")
		if !ok {
			return nil, fmt.Errorf("bad area id %q", id)
		}
		return areaDetail(d, env, file, area)
	case KindEvent:
		run := Walk(d, rest)
		if len(run.Nodes) == 0 && run.End.ID == "" {
			return nil, fmt.Errorf("no node %q", rest)
		}
		out := EventDetail{Kind: kind, Start: rest, Nodes: []NodeInfo{}, Sets: []string{}}
		sets := map[string]bool{}
		for _, n := range run.Nodes {
			out.Nodes = append(out.Nodes, info(d, n))
			for k := range n.Flags {
				sets[k] = true
			}
		}
		if run.End.ID != "" {
			e := info(d, run.End)
			out.End = &e
		}
		for k := range sets {
			out.Sets = append(out.Sets, k)
		}
		sort.Strings(out.Sets)
		return out, nil
	case KindBattle:
		n := d.Node(rest)
		i := d.StageIndex(n.Stage)
		if i < 0 {
			return nil, fmt.Errorf("no stage %q", n.Stage)
		}
		s := d.Stages[i]
		return BattleDetail{Kind: kind, Node: rest, Enemies: len(s.Enemies), Allies: len(s.Allies), Stage: map[string]any{
			"id": s.ID, "name": s.Name, "order": s.Order, "width": s.Width, "height": s.Height, "turnLimit": s.TurnLimit, "max": s.Max,
			"required": s.Required, "victory": nonNil(s.Victory), "defeat": nonNil(s.Defeat), "sub": nonNil(s.Sub), "reward": nonNilMap(s.Reward), "map": s.Map}}, nil
	case KindExplore:
		n := d.Node(rest)
		x := d.Explores[n.Explore]
		out := map[string]any{"kind": kind, "node": info(d, n)}
		if x != nil {
			out["hero"], out["start"] = x.Hero, x.Start
			areas := make([]string, 0, len(x.Areas))
			for k := range x.Areas {
				areas = append(areas, k)
			}
			sort.Strings(areas)
			out["areas"] = areas
		}
		return out, nil
	case KindEnd:
		return map[string]any{"kind": kind, "node": info(d, d.Node(rest))}, nil
	}
	return nil, fmt.Errorf("unknown vertex %q", id)
}

func areaDetail(d *content.Data, env Env, file, area string) (*AreaDetail, error) {
	x := d.Explores[file]
	if x == nil || x.Areas[area] == nil {
		return nil, fmt.Errorf("no area %s/%s", file, area)
	}
	a := x.Areas[area]
	out := &AreaDetail{Kind: KindArea, Explore: file, Area: area, Name: a.Name, Map: a.Map, Time: a.Time, Start: x.Start.Area == area}
	m := env.Map(a.Map)
	if m == nil {
		return out, nil
	}
	out.Frame = maplight.FrameOf(m)
	w := walkOf(env, m)
	mark := func(k Mark) Mark {
		x, y := out.Frame.Cell(k.At[0], k.At[1])
		k.Px = Pixel{x, y}
		k.Walk = w == nil || w.Open(k.At[0], k.At[1])
		return k
	}
	out.Marks = append(out.Marks, mark(Mark{Kind: "arrival", Name: "도착 칸", At: a.At}))
	for _, e := range a.Exits {
		out.Marks = append(out.Marks, mark(Mark{Kind: "exit", Name: e.Name, Display: e.Name, At: e.At, To: e.To, Node: e.Node, If: e.If, Shut: talkLines(d, e.Shut)}))
	}
	keys := make([]string, 0, len(a.People))
	for k := range a.People {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := a.People[k]
		mk := Mark{Kind: "person", Name: k, Display: content.PersonName(k), At: p.At, Face: p.Face, If: p.If, Talk: talkLines(d, p.Talk), Shop: p.Shop, Extra: map[string]string{}}
		for _, ev := range p.Event {
			mk.Events = append(mk.Events, EventRef{ev.If, ev.Node, ev.Side})
		}
		if p.Sortie != nil {
			mk.Sortie = p.Sortie.Ask
		}
		mk.Char, mk.Sprite, mk.Portrait = resolvePerson(d, env, mk.Display, p)
		if p.Chatter != "" {
			mk.Chatter = strings.Split(p.Chatter, "|")
		}
		if p.Move != nil {
			mk.Extra["이동"] = fmt.Sprintf("순찰 %v / 배회 %d", p.Move.Patrol, p.Move.Wander)
		}
		if p.Shop != "" {
			out.Shops = append(out.Shops, p.Shop)
		}
		out.Marks = append(out.Marks, mark(mk))
	}
	for _, s := range a.Signs {
		out.Marks = append(out.Marks, mark(Mark{Kind: "sign", Name: s.Name, At: s.At, Talk: talkLines(d, s.Talk)}))
	}
	return out, nil
}

// resolvePerson finds the character a shown name belongs to, the scene sprite sheet the game would draw them with
// (the person's own pick, then the character's look and class) and the portrait stem.
func resolvePerson(d *content.Data, env Env, name string, p *content.ExplorePerson) (char, sheet, portrait string) {
	if _, ok := d.Characters[name]; ok {
		char = name
	} else {
		for id, c := range d.Characters {
			if !c.Template && c.Name == name && (char == "" || id < char) {
				char = id
			}
		}
	}
	portrait = p.Portrait
	pick := []string{p.Sprite}
	if char != "" {
		c := d.Characters[char]
		if portrait == "" && !c.Look.NoPortrait {
			if portrait = c.Look.Portrait; portrait == "" {
				portrait = char
			}
		}
		line := d.ClassLine(c.Class)
		class := content.Class{}
		if len(line) > 0 {
			class = d.Classes[line[0]]
			pick = append(pick, c.Look.Scene, class.Scene, c.Look.Sprites[line[0]])
		} else {
			pick = append(pick, c.Look.Scene)
		}
		pick = append(pick, c.Look.Sprite, class.Sprite)
	}
	for _, id := range pick {
		if id != "" && env.HasSheet != nil && env.HasSheet(id) {
			return char, id, portrait
		}
	}
	return char, "", portrait
}

// nonNil keeps an empty list an empty JSON array, which the page maps over.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func nonNilMap(m map[string]int) map[string]int {
	if m == nil {
		return map[string]int{}
	}
	return m
}
