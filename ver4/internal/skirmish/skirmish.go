// Package skirmish sets up battle simulator matches: two configured sides on a map of one
// terrain, fought out by the AI on both sides.
package skirmish

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// Config is one simulated match. Classes and Terrains override the campaign data field by field.
type Config struct {
	Terrain   string
	Seed      uint64
	MaxRounds int
	Speed     int // replay speed on screen: 1, 2, 4 or 8
	Ally      Side
	Enemy     Side
	Classes   map[string]json.RawMessage            `json:",omitempty"`
	Terrains  map[string]map[string]json.RawMessage `json:",omitempty"` // family → tile → {"Factor", "Cost"}
}

// Side is one faction's level and officers.
type Side struct {
	Level    int
	Officers []Member
}

// Member is an officer, written as a name or as {"Officer", "Level", "Class"}.
// A zero Level is the side's level and an empty Class the officer's own.
type Member struct {
	Officer string
	Level   int    `json:",omitempty"`
	Class   string `json:",omitempty"`
}

func (m *Member) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &m.Officer); err == nil {
		return nil
	}
	type plain Member
	return json.Unmarshal(b, (*plain)(m))
}

func (m Member) MarshalJSON() ([]byte, error) {
	if m.Level == 0 && m.Class == "" {
		return json.Marshal(m.Officer)
	}
	type plain Member
	return json.Marshal(plain(m))
}

// MaxOfficers is how many officers a side can field: two columns of the 12-row maps.
const MaxOfficers = 24

func Load(path string) (Config, error) {
	c := Config{Seed: 1, MaxRounds: 30, Speed: 1}
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func (c Config) Save(path string) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}

// LoadMaps reads the simulator maps, one per terrain.
func LoadMaps(path string) ([]content.Stage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var maps []content.Stage
	return maps, json.Unmarshal(b, &maps)
}

// Clone is a deep copy of d.
func Clone(d *content.Data) *content.Data {
	b, _ := json.Marshal(d)
	var c content.Data
	_ = json.Unmarshal(b, &c)
	c.Hash = d.Hash
	return &c
}

// Apply writes c's class and terrain overrides into d.
func Apply(d *content.Data, c Config) error {
	for name, raw := range c.Classes {
		cl, ok := d.Classes[name]
		if !ok {
			return fmt.Errorf("unknown class %s", name)
		}
		if err := json.Unmarshal(raw, &cl); err != nil {
			return fmt.Errorf("class %s: %w", name, err)
		}
		d.Classes[name] = cl
	}
	for family, tiles := range c.Terrains {
		if d.Terrain[family] == nil {
			return fmt.Errorf("unknown family %s", family)
		}
		for tile, raw := range tiles {
			t := d.Terrain[family][tile]
			if err := json.Unmarshal(raw, &t); err != nil {
				return fmt.Errorf("terrain %s %s: %w", family, tile, err)
			}
			d.Terrain[family][tile] = t
		}
	}
	return nil
}

// tuned are the class fields the simulator edits.
type tuned struct {
	Bonus                 [6]float64
	HPFactor, Move, Range int
}

// SetOverrides makes c's Classes and Terrains the differences of work from base.
func SetOverrides(c *Config, base, work *content.Data) {
	c.Classes, c.Terrains = nil, nil
	for name, w := range work.Classes {
		a, b := tuned{w.Bonus, w.HPFactor, w.Move, w.Range}, base.Classes[name]
		if a != (tuned{b.Bonus, b.HPFactor, b.Move, b.Range}) {
			if c.Classes == nil {
				c.Classes = map[string]json.RawMessage{}
			}
			c.Classes[name], _ = json.Marshal(a)
		}
	}
	for family, tiles := range work.Terrain {
		for tile, t := range tiles {
			if reflect.DeepEqual(t, base.Terrain[family][tile]) {
				continue
			}
			if c.Terrains == nil {
				c.Terrains = map[string]map[string]json.RawMessage{}
			}
			if c.Terrains[family] == nil {
				c.Terrains[family] = map[string]json.RawMessage{}
			}
			c.Terrains[family][tile], _ = json.Marshal(t)
		}
	}
}

// Build is the match's data: a copy of base with c's overrides, a copy of every officer
// per side at its level, and one stage of the chosen terrain with the allies in the west
// columns and the enemies in the east.
func Build(base *content.Data, maps []content.Stage, c Config) (*content.Data, error) {
	var st *content.Stage
	names := []string{}
	for _, m := range maps {
		names = append(names, m.Name)
		if m.Name == c.Terrain {
			st = &m
		}
	}
	if st == nil {
		return nil, fmt.Errorf("terrain %q: one of %v", c.Terrain, names)
	}
	d := Clone(base)
	if err := Apply(d, c); err != nil {
		return nil, err
	}
	tile := string(st.Tiles[0][0])
	st.Goal, st.Party, st.Enemies = "rout", nil, nil
	for i, side := range []Side{c.Ally, c.Enemy} {
		if len(side.Officers) == 0 || len(side.Officers) > MaxOfficers {
			return nil, fmt.Errorf("each side needs 1 to %d officers", MaxOfficers)
		}
		for j, m := range side.Officers {
			o, ok := d.Officers[m.Officer]
			if !ok {
				return nil, fmt.Errorf("unknown officer %s", m.Officer)
			}
			o.Level = side.Level
			if m.Level > 0 {
				o.Level = m.Level
			}
			if m.Class != "" {
				o.Class = m.Class
			}
			cl, ok := d.Classes[o.Class]
			if !ok {
				return nil, fmt.Errorf("unknown class %s", o.Class)
			}
			if o.Level < 1 || o.Level > 99 {
				return nil, fmt.Errorf("%s: level %d is outside 1..99", o.Name, o.Level)
			}
			if d.Terrain[cl.Family][tile].Cost == 0 {
				return nil, fmt.Errorf("%s(%s)은 %s에 들어갈 수 없습니다. 지형 수치에서 %s의 이동 비용을 정하세요", o.Name, cl.Family, c.Terrain, cl.Family)
			}
			// two columns per side, filled from the middle row outwards
			k := j % st.Height
			off := (k + 1) / 2
			if k%2 == 1 {
				off = -off
			}
			sp := content.Spawn{X: 2 + j/st.Height, Y: st.Height/2 + off}
			o.ID = fmt.Sprintf("%c%d·%s", "AE"[i], j+1, o.Name)
			sp.Officer = o.ID
			if i == 1 {
				sp.X = st.Width - 1 - sp.X
				st.Enemies = append(st.Enemies, sp)
			} else {
				st.Party = append(st.Party, sp)
			}
			d.Officers[o.ID] = o
		}
	}
	st.Limit = len(st.Party)
	d.Stages = []content.Stage{*st}
	return d, nil
}

// Outcome is one finished match.
type Outcome struct {
	Winner string // ally, enemy or draw
	Rounds int
	Units  []core.UnitView
	Dealt  map[string]int // damage each unit dealt
}

// Play runs one match with the AI on both sides until a rout or the round limit.
func Play(d *content.Data, seed uint64, maxRounds int) (Outcome, error) {
	e, err := core.NewSkirmish(d, seed)
	if err != nil {
		return Outcome{}, err
	}
	out := Outcome{Winner: "draw", Dealt: map[string]int{}}
	for {
		o := e.Observe()
		if o.Phase == "result" || o.Round > maxRounds {
			out.Rounds = min(o.Round, maxRounds)
			out.Units = o.UnitViews
			if o.Phase == "result" {
				out.Winner = map[string]string{"victory": "ally", "defeat": "enemy"}[o.Result]
			}
			return out, nil
		}
		c, err := ai.Next(e)
		if err != nil {
			return out, err
		}
		r, err := e.Apply(c)
		if err != nil {
			return out, fmt.Errorf("%+v: %w", c, err)
		}
		for _, ev := range r.Events {
			if ev.Kind == "damage" {
				out.Dealt[ev.Actor] += ev.Amount
			}
		}
	}
}
