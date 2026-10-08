package content

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

type Class struct {
	Name, Family, Weapon, Armor string
	Move, Range, HPFactor       int
	Bonus                       [6]float64
	MPBase, RecoveryBase        int
	MPCoeff, RecoveryCoeff      float64
	Skills, Learn               []string
	Promotes                    []string
	Tier                        int
}
type Officer struct {
	ID, Name, Class          string
	Stats                    [6]int
	Level                    int
	Traits, Learn, Equipment []string
}
type Equipment struct {
	Name, Slot, Type      string
	Bonus                 [6]float64
	Traits, Learn, Skills []string
}
type Trait struct {
	Cost   int
	Skills []string
}
type Skill struct {
	Kind, Mode, Shape, Effect, Condition string
	Cost, Hit, Min, Max, Duration        int
	Coeff                                float64
}
type Terrain struct {
	Factor float64
	Cost   int
}
type Point struct{ X, Y int }
type Spawn struct {
	Officer string
	X, Y    int
}
type Duel struct {
	ID, Ally, Enemy, Outcome, AllyResult, EnemyResult, Text string
	AttackDown, DefenseDown                                 int `json:",omitempty"` // an ally win cuts every enemy's attack or defense by this percent for the battle
}
type Stage struct {
	ID, Name, Boss                  string
	Width, Height, Limit, Threshold int
	Tiles                           []string
	Allies                          []Point
	Enemies                         []Spawn
	Reward                          map[string]int
	Duels                           []Duel
	Goal                            string  `json:",omitempty"` // "rout": a skirmish won by routing the other side
	Party                           []Spawn `json:",omitempty"` // a skirmish's allies
	Charge                          bool    `json:",omitempty"` // the enemy marches on the allies from the first turn
}
type Node struct {
	ID, Kind, Text, Next, Event, Officer, Item string
	Count                                      int
	Choices                                    map[string]string
}
type Data struct {
	InitialInventory map[string]int
	Version, Draft   string
	Hash             string `json:"-"`
	Classes          map[string]Class
	Officers         map[string]Officer
	Equipment        map[string]Equipment
	Traits           map[string]Trait
	Skills           map[string]Skill
	Terrain          map[string]map[string]Terrain
	Items            map[string]int
	CommonLearn      []string
	Stages           []Stage
	Nodes            []Node
}

func Load(path string) (*Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d Data
	if err = json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	d.Hash = fmt.Sprintf("%x", sha256.Sum256(b))
	return &d, d.Validate()
}
func (d *Data) Validate() error {
	if len(d.InitialInventory) == 0 || len(d.Classes) < 5 || len(d.Officers) < 12 || len(d.Terrain) != 5 {
		return fmt.Errorf("missing content definitions")
	}
	for id, n := range d.InitialInventory {
		if _, ok := d.Items[id]; !ok || n < 0 {
			return fmt.Errorf("initial item %s", id)
		}
	}
	if d.Version == "" || len(d.Stages) != 3 || len(d.Nodes) == 0 {
		return fmt.Errorf("invalid content version/stages/nodes")
	}
	for id, c := range d.Classes {
		if c.Move < 1 || c.Range < 1 || c.HPFactor < 1 || c.Tier < 0 || c.Tier > 2 {
			return fmt.Errorf("class %s", id)
		}
		for _, next := range c.Promotes {
			n, ok := d.Classes[next]
			if !ok || n.Tier != c.Tier+1 || n.Family != c.Family {
				return fmt.Errorf("promotion %s", id)
			}
		}
		for _, s := range c.Skills {
			if _, ok := d.Skills[s]; !ok {
				return fmt.Errorf("unknown skill %s", s)
			}
		}
		for _, t := range c.Learn {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("unknown trait %s", t)
			}
		}
	}
	for id, o := range d.Officers {
		c, ok := d.Classes[o.Class]
		if !ok || o.Level < 1 || o.Level > 99 {
			return fmt.Errorf("officer %s", id)
		}
		for _, s := range o.Stats {
			if s < 1 || s > 100 {
				return fmt.Errorf("stats %s", id)
			}
		}
		for _, t := range append(o.Traits, o.Learn...) {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("trait %s", t)
			}
		}
		for _, eq := range o.Equipment {
			e, ok := d.Equipment[eq]
			if !ok || (e.Slot == "weapon" && e.Type != c.Weapon) || (e.Slot == "armor" && e.Type != c.Armor) {
				return fmt.Errorf("equipment %s on %s", eq, id)
			}
		}
	}
	for _, e := range d.Equipment {
		for _, t := range append(e.Traits, e.Learn...) {
			if _, ok := d.Traits[t]; !ok {
				return fmt.Errorf("equipment trait %s", t)
			}
		}
		for _, s := range e.Skills {
			if _, ok := d.Skills[s]; !ok {
				return fmt.Errorf("equipment skill %s", s)
			}
		}
	}
	for _, t := range d.Traits {
		if t.Cost < 1 {
			return fmt.Errorf("trait cost")
		}
		for _, s := range t.Skills {
			if _, ok := d.Skills[s]; !ok {
				return fmt.Errorf("trait skill %s", s)
			}
		}
	}
	for _, t := range d.CommonLearn {
		if _, ok := d.Traits[t]; !ok {
			return fmt.Errorf("common trait %s", t)
		}
	}
	duelIDs := map[string]bool{}
	for _, s := range d.Stages {
		if s.Width < 8 || s.Height < 8 || len(s.Tiles) != s.Height || len(s.Allies) < s.Limit {
			return fmt.Errorf("map %s", s.ID)
		}
		seen := map[Point]bool{}
		boss := false
		for _, r := range s.Tiles {
			if len(r) != s.Width {
				return fmt.Errorf("row %s", s.ID)
			}
			for _, v := range r {
				for f := range d.Terrain {
					if _, ok := d.Terrain[f][string(v)]; !ok {
						return fmt.Errorf("terrain %c", v)
					}
				}
			}
		}
		for _, e := range s.Enemies {
			p := Point{e.X, e.Y}
			if _, ok := d.Officers[e.Officer]; !ok || e.X < 0 || e.Y < 0 || e.X >= s.Width || e.Y >= s.Height || seen[p] {
				return fmt.Errorf("spawn %s", e.Officer)
			}
			seen[p] = true
			boss = boss || e.Officer == s.Boss
		}
		for _, event := range s.Duels {
			enemy := false
			for _, sp := range s.Enemies {
				if sp.Officer == event.Enemy {
					enemy = true
				}
			}
			ally := event.Ally == "유비" || event.Ally == "관우" || event.Ally == "장비" || event.Ally == "간옹"
			valid := func(s string, list ...string) bool {
				for _, v := range list {
					if v == s {
						return true
					}
				}
				return false
			}
			if event.ID == "" || duelIDs[event.ID] || !enemy || !ally || event.AttackDown < 0 || event.AttackDown > 90 || event.DefenseDown < 0 || event.DefenseDown > 90 || !valid(event.Outcome, "victory", "defeat", "draw") || !valid(event.AllyResult, "stay", "retreat", "death") || !valid(event.EnemyResult, "stay", "retreat", "death") {
				return fmt.Errorf("duel %s", event.ID)
			}
			duelIDs[event.ID] = true
		}
		if !boss && s.Goal != "rout" {
			return fmt.Errorf("boss %s", s.ID)
		}
		for _, p := range s.Allies {
			if p.X < 0 || p.Y < 0 || p.X >= s.Width || p.Y >= s.Height || seen[p] {
				return fmt.Errorf("ally spawn")
			}
			seen[p] = true
		}
	}
	nodes := map[string]bool{}
	for _, n := range d.Nodes {
		if n.ID == "" || nodes[n.ID] {
			return fmt.Errorf("node id")
		}
		nodes[n.ID] = true
	}
	for _, n := range d.Nodes {
		if n.Next != "" && !nodes[n.Next] {
			return fmt.Errorf("node next")
		}
		for _, v := range n.Choices {
			if !nodes[v] {
				return fmt.Errorf("choice next")
			}
		}
	}
	return nil
}
func (d *Data) Node(id string) Node {
	for _, n := range d.Nodes {
		if n.ID == id {
			return n
		}
	}
	return Node{}
}
