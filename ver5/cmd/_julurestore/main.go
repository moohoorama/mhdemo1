// Temporary: plays B09 with the AI and restores a snapshot after every command to find a CorruptSave.
package main

import (
	"fmt"
	"os"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

func main() {
	d, err := content.Load("assets/data")
	if err != nil {
		panic(err)
	}
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B09" {
			d.Campaign.Start = n.ID
		}
	}
	e := core.New(d, 1)
	if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
		panic(err)
	}
	for step := 0; step < 20000; step++ {
		o := e.Observe()
		if o.Phase != "battle" && o.Phase != "deployment" {
			fmt.Println("end", o.Phase, o.Result)
			return
		}
		if _, err := core.Restore(d, e.Snapshot()); err != nil {
			fmt.Println("step", step, "phase", o.Phase, "round", o.Round, "turn", o.Turn, err)
			os.Exit(1)
		}
		c, err := ai.Next(e)
		if err != nil {
			panic(err)
		}
		if _, err := e.Apply(c); err != nil {
			panic(err)
		}
	}
}

func init() {
	d, _ := content.Load("assets/data")
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B09" {
			d.Campaign.Start = n.ID
		}
	}
	e := core.New(d, 1)
	e.Apply(core.Command{Kind: "start"})
	s := e.Snapshot()
	for _, o := range s.Characters {
		def := d.Characters[o.Def]
		held := def.Traits
		if o.TraitsOverride != nil {
			held = *o.TraitsOverride
		}
		held = append(append([]string{}, held...), o.Learned...)
		for slot, eq := range o.Equipment {
			v, ok := d.Equipment[eq]
			if !ok || v.Slot != slot || !d.Carries(d.Classes[o.Class], held, v) {
				fmt.Println("bad:", o.ID, o.Class, slot, eq, ok, v.Slot, held)
			}
		}
	}
}
