// Temporary tool: starts a game at a battle's waypoint, lets every officer learn the cheapest traits it can
// afford (as a player would at the learning window), optionally names deputies, and prints the party as JSON
// for UI mockups: each officer's attributes, battle stats, equipment, trait catalog by source and learnables.
//
//	go run ./cmd/_uiexport -list
//	go run ./cmd/_uiexport -node c14-battle -deputy gaoshun:sunqian -keep 600 > out.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
)

type trait struct {
	ID, Name, Desc, Kind, Tone, Source string
	Cost                               int
	Have, Learnable                    bool
	Requires                           []string
}

type equip struct {
	Slot, ID, Name, Icon string
	Bonus                [6]float64
	Traits, Learn        []string
}

type officer struct {
	ID, Name, Class, ClassID, Portrait, Bio, Deputy, DeputyOf string
	Lord                                                      bool
	Level, Points, XP                                         int
	Attributes                                                [6]int
	Stats                                                     core.Stats
	StatsAlone                                                *core.Stats `json:",omitempty"`
	Move, Range                                               int
	Equipment                                                 []equip
	Traits                                                    []trait
	AttrsAlone                                                [6]int
	Candidates                                                []candidate `json:",omitempty"`
}

// candidate is another officer as this one's deputy: the weight, what the unit would be and why it cannot be.
type candidate struct {
	ID, Name, Class, Portrait, Error string
	Level                            int
	Weight                           float64
	Charm                            int
	Stats                            core.Stats
	Attrs                            [6]int
	Traits                           []string
}

func main() {
	dir := flag.String("content", "assets/data", "content directory")
	node := flag.String("node", "", "waypoint node")
	list := flag.Bool("list", false, "list the waypoints")
	deputies := flag.String("deputy", "", "lord:deputy pairs, comma separated")
	keep := flag.Int("keep", 0, "trait points every officer keeps unspent")
	cands := flag.Bool("candidates", false, "list every other officer as each one's deputy")
	flag.Parse()
	d, err := content.Load(*dir)
	check(err)
	if *list {
		for _, w := range d.Waypoints() {
			fmt.Println(w.Chapter, w.Node, w.Stage, w.Place)
		}
		return
	}
	e, err := core.NewAt(d, 1, *node, "")
	check(err)
	o := e.Observe()
	for _, l := range o.Learners {
		for {
			opts := e.Learnable(l.ID)
			sort.Slice(opts, func(i, j int) bool { return d.Traits[opts[i]].Cost < d.Traits[opts[j]].Cost })
			done := true
			for _, t := range opts {
				if e.Observe().Characters[slices.IndexFunc(e.Observe().Characters, func(c core.Character) bool { return c.ID == l.ID })].Points-d.Traits[t].Cost < *keep {
					continue
				}
				if _, err := e.Apply(core.Command{Kind: "learn", Actor: l.ID, Trait: t}); err == nil {
					done = false
					break
				}
			}
			if done {
				break
			}
		}
		e.Apply(core.Command{Kind: "learn_close", Actor: l.ID})
	}
	alone := map[string]core.Stats{}
	for _, pair := range strings.Split(*deputies, ",") {
		if lord, dep, ok := strings.Cut(pair, ":"); ok {
			alone[lord], _ = e.CharacterStats(lord)
			if _, err := e.Apply(core.Command{Kind: "deputy", Actor: lord, Target: dep}); err != nil {
				fmt.Fprintln(os.Stderr, "deputy", pair, err)
			}
		}
	}
	o = e.Observe()
	out := []officer{}
	for _, c := range o.Characters {
		def := d.Characters[c.Def]
		if !def.Playable || c.Dead || c.Temp {
			continue
		}
		cl := d.Classes[c.Class]
		of := officer{ID: c.ID, Name: def.Name, Class: cl.Name, ClassID: c.Class, Portrait: def.Look.Portrait, Bio: def.Biography,
			Deputy: c.Deputy, Lord: def.Lord, Level: c.Level, Points: c.Points, XP: c.XP, Attributes: def.Stats, Move: cl.Move, Range: cl.Range}
		for _, x := range o.Characters {
			if x.Deputy == c.ID {
				of.DeputyOf = x.ID
			}
		}
		of.Stats, _ = e.CharacterStats(c.ID)
		_, of.AttrsAlone, _ = e.DeputyStats(c.ID, "")
		for _, x := range o.Characters {
			xd := d.Characters[x.Def]
			if !*cands || x.ID == c.ID || !xd.Playable || x.Dead || x.Temp {
				continue
			}
			st, at, err := e.DeputyStats(c.ID, x.ID)
			if err != nil {
				continue
			}
			k := candidate{ID: x.ID, Name: xd.Name, Class: d.Classes[x.Class].Name, Portrait: xd.Look.Portrait, Level: x.Level, Stats: st, Attrs: at, Charm: xd.Stats[5]}
			k.Weight, _ = e.DeputyBlend(c.ID, x.ID)
			if err := e.DeputyCheck(c.ID, x.ID); err != nil {
				k.Error = err.Error()
			}
			for _, t := range e.DeputyTraits(x.ID) {
				k.Traits = append(k.Traits, d.Traits[t].Name)
			}
			of.Candidates = append(of.Candidates, k)
		}
		if s, ok := alone[c.ID]; ok {
			of.StatsAlone = &s
		}
		equipIDs := []string{}
		for _, slot := range []string{"weapon", "armor", "accessory"} {
			if id, ok := c.Equipment[slot]; ok {
				eq := d.Equipment[id]
				of.Equipment = append(of.Equipment, equip{Slot: slot, ID: id, Name: eq.Name, Icon: eq.Icon, Bonus: eq.Bonus, Traits: eq.Traits, Learn: eq.Learn})
				equipIDs = append(equipIDs, id)
			}
		}
		have := slices.Clone(c.Learned)
		for _, id := range equipIDs {
			have = append(have, d.Equipment[id].Traits...)
		}
		learnable := e.Learnable(c.ID)
		seen := map[string]bool{}
		add := func(ids []string, src string) {
			for _, id := range ids {
				t, ok := d.Traits[id]
				if !ok || seen[id] {
					continue
				}
				seen[id] = true
				req := []string{}
				for _, q := range t.Requires {
					req = append(req, d.Traits[q].Name)
				}
				of.Traits = append(of.Traits, trait{ID: id, Name: t.Name, Desc: t.Desc, Kind: t.Kind, Tone: t.Tone, Source: src, Cost: t.Cost,
					Have: slices.Contains(have, id), Learnable: slices.Contains(learnable, id), Requires: req})
			}
		}
		add(def.Learn, "고유")
		add(cl.AllPool(), "병종")
		for _, id := range equipIDs {
			add(d.Equipment[id].Traits, "장비")
			add(d.Equipment[id].Learn, "장비")
		}
		add(c.Learned, "고유")
		out = append(out, of)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	check(enc.Encode(map[string]any{"node": *node, "stage": d.Node(*node).Stage, "officers": out}))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
