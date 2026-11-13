// Command _traitcount lists every named NPC spawn with the level, class and traits it fights with.
package main

import (
	"fmt"
	"sort"
	"strings"

	"srpg/internal/content"
)

func main() {
	d, err := content.Load("assets/data")
	if err != nil {
		panic(err)
	}
	promoted := func(def, class string, level int) string {
		for {
			cl, next := d.Classes[class], d.Promotes(def, class)
			if cl.Tier >= len(d.Rules.PromoteLevels) || level < d.Rules.PromoteLevels[cl.Tier] || len(next) == 0 {
				return class
			}
			class = next[0]
		}
	}
	type row struct {
		order                 int
		stage, side, id, line string
	}
	var rows []row
	for _, s := range d.Stages {
		var all []struct {
			side string
			sp   content.Spawn
		}
		for _, sp := range s.Enemies {
			all = append(all, struct {
				side string
				sp   content.Spawn
			}{"적", sp})
		}
		for _, sp := range s.Party {
			if sp.AI {
				all = append(all, struct {
					side string
					sp   content.Spawn
				}{"동맹", sp})
			}
		}
		for _, ev := range s.Events {
			for _, a := range ev.Do {
				for _, sp := range a.Spawns {
					side := "적+"
					if sp.Side != "" && sp.Side != "enemy" {
						side = "동맹+"
					}
					all = append(all, struct {
						side string
						sp   content.Spawn
					}{side, sp})
				}
			}
		}
		for _, x := range all {
			c := d.Characters[x.sp.Character]
			if c.Template || c.Playable {
				continue
			}
			lv := d.SpawnLevel(s.Order, x.sp)
			class := c.Class
			if x.sp.Class != "" {
				class = x.sp.Class
			}
			class = promoted(x.sp.Character, class, lv)
			ts := d.PlannedTraits(x.sp.Character, class, lv, x.sp.Traits)
			var own, eq []string
			for _, t := range ts {
				own = append(own, d.Traits[t].Name)
			}
			equip := c.Equipment
			if x.sp.Equipment != nil {
				equip = *x.sp.Equipment
			}
			for _, id := range equip {
				for _, t := range d.Equipment[id].Traits {
					eq = append(eq, d.Traits[t].Name)
				}
			}
			rows = append(rows, row{s.Order, s.ID, x.side, x.sp.Character, fmt.Sprintf("%s\t%s\tLv%d\t%s\t%d\t%s\t[장비 %s]", c.Name, d.Classes[class].Name, lv, x.sp.Rank, len(ts), strings.Join(own, "·"), strings.Join(eq, "·"))})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].order < rows[j].order })
	for _, r := range rows {
		fmt.Printf("%d\t%s\t%s\t%s\n", r.order, r.stage, r.side, r.line)
	}
}
