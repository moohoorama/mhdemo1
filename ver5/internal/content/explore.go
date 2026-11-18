package content

import (
	"fmt"
	"strings"
)

// Explore is a story's walk between battles (assets/scenario/<id>/exploreNN.json): the hero walks areas, each on a
// scene map of format 2, joined by exits, and talks to the people standing in them. Talking to some starts an event
// (story nodes ending with a back node); talking to the confidant asks whether to sortie, which goes on with the
// story after the explore node. Every place is a small tile [U, V] of the map: half a game cell.
type Explore struct {
	Hero  string // the walker: a character id
	Start Spot
	Areas map[string]*ExploreArea
}

// Spot is where someone stands: an area's small tile, facing NE, NW, SE or SW.
type Spot struct {
	Area string
	At   [2]int
	Face string
}

type ExploreArea struct {
	Map, Name string
	Time      string        `json:",omitempty"` // the scene time of day (night, dusk); day when empty
	Times     []ExploreTime `json:",omitempty"` // the first whose If holds stands in for Time (night falls)
	Zoom      float64       `json:",omitempty"` // the camera's zoom on arrival, whole numbers; 2 when 0
	Grass     *GrassTint    `json:",omitempty"` // the grass here when the map stands for another land; nil keeps the map's
	At        [2]int        // where the hero stands on arriving where no exit leads back
	Exits     []ExploreExit
	People    map[string]*ExplorePerson `json:",omitempty"`
	Effects   []ExploreEffect           `json:",omitempty"`
	Signs     []ExploreSign             `json:",omitempty"`
}

// ExploreExit leads to area To while every If holds: the hero walks to small tile At, then stands at the exit of To
// that leads back. Object is the anchor of the building it belongs to, whose picture can be clicked too. An exit
// whose If fails is shut: walking up to it shows Shut. Two exits may share a place with different conditions.
// An exit with Node instead of To leads out of the area into an event: going through it starts the event at Node
// (a city gate the party leaves by, its back node with Leave ending the walk).
type ExploreExit struct {
	Name   string
	At     [2]int
	Object *[2]int  `json:",omitempty"`
	To     string   `json:",omitempty"`
	Node   string   `json:",omitempty"`
	If     []string `json:",omitempty"`
	Shut   []Line   `json:",omitempty"`
}

// Line is one line of a talk: [speaker, text]; an empty speaker is narration. The speaker is a character id or
// a shown name.
type Line [2]string

// ExplorePerson stands in an area while every If holds. The key in ExploreArea.People is the shown name, what
// follows an @ telling apart the same person's stands under other flags; a key with # is an extra. A character's
// name (or id) brings that character's scene sprite and portrait; Sprite and Portrait stand in for others.
type ExplorePerson struct {
	At               [2]int
	Face             string
	Sprite, Portrait string         `json:",omitempty"`
	Faction, Pose    string         `json:",omitempty"`
	If               []string       `json:",omitempty"`
	Move             *PersonMove    `json:",omitempty"` // standing when nil
	Chatter          string         `json:",omitempty"` // short bubbles now and then, "|" between them
	Play             string         `json:",omitempty"` // a motion played over and over (drilling, working)
	Talk             []Line         `json:",omitempty"` // what talking says when no event is due
	Event            []ExploreEvent `json:",omitempty"` // the first whose If holds starts on talking
	Sortie           *ExploreSortie `json:",omitempty"` // the confidant
	Shop             string         `json:",omitempty"` // a shop id of Data.Shops
}

// PersonMove is how a person walks about: to and fro along Patrol (small tiles from At on), or now and then to an
// open small tile within Wander of At.
type PersonMove struct {
	Patrol [][2]int `json:",omitempty"`
	Wander int      `json:",omitempty"`
}

// ExploreEvent is an event a person starts on being talked to, while every If holds. A Side event is no part of the
// main story: the walk's guide shows it in its own colour, and the sortie does not wait for it.
type ExploreEvent struct {
	If   []string `json:",omitempty"`
	Node string
	Side bool `json:",omitempty"`
}

// ExploreTime is an area's time of day while every If holds.
type ExploreTime struct {
	If   []string `json:",omitempty"`
	Time string
}

type ExploreSortie struct {
	If  []string `json:",omitempty"`
	Ask string   // the question, answered yes or no
}

type ExploreEffect struct {
	Effect string
	At     [2]int
}

type ExploreSign struct {
	Name string
	At   [2]int
	Talk []Line
}

// Shop sells Items (equipment or item ids) for their Price, or for Rate percent of it when Rate is set (미축 selling
// to his own lord below the market).
type Shop struct {
	Name  string
	Items []string
	Rate  int    `json:",omitempty"`
	Mark  string `json:",omitempty"` // the guide plate's hue: "" the usual shop yellow, "red" a shop with new stock
}

// PersonName is the shown name of a person key: the part before @ and #.
func PersonName(key string) string {
	if i := strings.IndexAny(key, "@#"); i >= 0 {
		return key[:i]
	}
	return key
}

// Arrival is where the hero stands entering area to from area from: at the exit of to that leads back to from,
// facing into the area (NW), else at the area's At facing SE.
func (x *Explore) Arrival(to, from string) Spot {
	a := x.Areas[to]
	for _, e := range a.Exits {
		if e.To == from && from != "" {
			return Spot{to, e.At, "NW"}
		}
	}
	return Spot{to, a.At, "SE"}
}

// Gainable reports whether a grant or reward can give id: money, an equipment or an item.
func (d *Data) Gainable(id string) bool {
	_, eq := d.Equipment[id]
	_, item := d.Items[id]
	return id == Money || eq || item
}

// Money is the id that grants and rewards give money by.
const Money = "money"

// Price is what a shop asks for an equipment or item id; 0 when it is not sold.
// Ask is what shop asks for id: its Price, at the shop's Rate.
func (d *Data) Ask(shop, id string) int {
	p := d.Price(id)
	if r := d.Shops[shop].Rate; r > 0 {
		p = p * r / 100
	}
	return p
}

func (d *Data) Price(id string) int {
	if e, ok := d.Equipment[id]; ok {
		return e.Price
	}
	return d.Items[id].Price
}

var faces = map[string]bool{"NE": true, "NW": true, "SE": true, "SW": true}

func (d *Data) validateExplores(nodes map[string]Node) error {
	for id, s := range d.Shops {
		if s.Name == "" {
			return fmt.Errorf("shop %s: no name", id)
		}
		if !in(s.Mark, "", "red") {
			return fmt.Errorf("shop %s: mark %q", id, s.Mark)
		}
		for _, item := range s.Items {
			if d.Price(item) <= 0 {
				return fmt.Errorf("shop %s: %s has no price", id, item)
			}
		}
	}
	conds := func(where string, cs []string) error {
		for _, c := range cs {
			if err := d.validateCond(c, nil); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
		}
		return nil
	}
	for name, x := range d.Explores {
		if !d.Has(x.Hero) || x.Areas[x.Start.Area] == nil || !faces[x.Start.Face] {
			return fmt.Errorf("%s: hero %q or start %q", name, x.Hero, x.Start.Area)
		}
		for aid, a := range x.Areas {
			where := name + " " + aid
			if a.Map == "" || a.Name == "" {
				return fmt.Errorf("%s: map and name", where)
			}
			for _, t := range a.Times {
				if err := conds(where+" time "+t.Time, t.If); err != nil {
					return err
				}
			}
			for _, e := range a.Exits {
				if (e.Node == "") == (e.To == "") || e.To != "" && x.Areas[e.To] == nil || e.Name == "" {
					return fmt.Errorf("%s: exit %q to %q", where, e.Name, e.To)
				}
				if e.Node != "" {
					if err := d.endsWithBack(e.Node, nodes); err != nil {
						return fmt.Errorf("%s exit %s: %w", where, e.Name, err)
					}
				}
				if err := conds(where+" exit "+e.Name, e.If); err != nil {
					return err
				}
			}
			for key, p := range a.People {
				where := where + " " + key
				if PersonName(key) == "" || !faces[p.Face] {
					return fmt.Errorf("%s: name or face", where)
				}
				if !strings.Contains(key, "#") && len(p.Talk) == 0 && len(p.Event) == 0 && p.Shop == "" && p.Sortie == nil {
					return fmt.Errorf("%s: a named person says nothing (no Talk, Event, Shop or Sortie)", where)
				}
				if p.Move != nil && (len(p.Move.Patrol) == 0) == (p.Move.Wander <= 0) {
					return fmt.Errorf("%s: move is a patrol or a wander", where)
				}
				if err := conds(where, p.If); err != nil {
					return err
				}
				for _, ev := range p.Event {
					if err := conds(where, ev.If); err != nil {
						return err
					}
					if err := d.endsWithBack(ev.Node, nodes); err != nil {
						return fmt.Errorf("%s: %w", where, err)
					}
				}
				if p.Sortie != nil {
					if p.Sortie.Ask == "" {
						return fmt.Errorf("%s: sortie asks nothing", where)
					}
					if err := conds(where, p.Sortie.If); err != nil {
						return err
					}
				}
				if _, ok := d.Shops[p.Shop]; p.Shop != "" && !ok {
					return fmt.Errorf("%s: shop %s", where, p.Shop)
				}
			}
		}
	}
	for _, n := range nodes {
		switch n.Kind {
		case "explore":
			if d.Explores[n.Explore] == nil {
				return fmt.Errorf("node %s explores %q", n.ID, n.Explore)
			}
			if n.Opening != "" {
				if err := d.endsWithBack(n.Opening, nodes); err != nil {
					return fmt.Errorf("node %s opening: %w", n.ID, err)
				}
			}
		case "back":
			if n.Area == "" && (n.At != nil || n.Face != "") || n.Face != "" && !faces[n.Face] || n.Leave && n.Area != "" {
				return fmt.Errorf("node %s: back", n.ID)
			}
		}
	}
	return nil
}

// endsWithBack checks that an event starting at node id comes back to the walk on every way it can take, waiting
// for nothing but lines and choices on the way. An end node with a Text (the hero falls) may close a way too.
func (d *Data) endsWithBack(id string, nodes map[string]Node) error {
	seen := map[string]bool{}
	var walk func(id string) error
	walk = func(id string) error {
		n := nodes[id]
		switch {
		case n.ID == "":
			return fmt.Errorf("event at %q runs out before a back node", id)
		case seen[id] || n.Kind == "back" || n.Kind == "end" && n.Text != "": // or the hero falls and the story ends
			return nil
		case n.Kind == "battle" || n.Kind == "explore" || n.Kind == "end":
			return fmt.Errorf("event reaches %s node %s", n.Kind, id)
		}
		seen[id] = true
		next := []string{d.After(id)}
		if n.Kind == "branch" {
			next = []string{n.Else}
			if n.Else == "" {
				next[0] = d.After(id)
			}
			for _, c := range n.Cases {
				next = append(next, c.Next)
			}
		}
		for _, v := range n.Choices {
			next = append(next, v)
		}
		if n.Kind == "choice" {
			next = next[1:]
		}
		for _, v := range next {
			if err := walk(v); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(id)
}
