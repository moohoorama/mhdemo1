package core

import (
	"testing"

	"srpg/internal/content"
)

func TestSpawnOverrides(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	hp, mp := 10, 0
	eq := []string{}
	traits := []string{}
	sp := content.Spawn{Character: "troop_cavalry", Class: "heavy_cavalry", Level: 20, Facing: "SW", HP: &hp, MP: &mp, Equipment: &eq, Traits: &traits}
	if err = d.ValidateSpawn(sp); err != nil {
		t.Fatal(err)
	}
	e := fromState(d, State{Difficulty: "normal"})
	a := e.spawn(sp.Character, sp.Character, "enemy", 1, 1, sp)
	other := e.spawn(sp.Character, sp.Character, "enemy", 2, 1, content.Spawn{})
	if a.HP != 10 || a.MP != 0 || a.Facing != "SW" || e.char(a.Character).Class != "heavy_cavalry" || a.ID == other.ID || e.char(other.Character).Class != "light_cavalry" {
		t.Fatalf("overrides %+v / %+v", a, other)
	}
	preview := SpawnPreview(d, sp)
	if preview.Stats != e.stats(a) || preview.HP != a.HP {
		t.Fatal("preview differs")
	}
	sp.Class = "archer"
	if d.ValidateSpawn(sp) == nil {
		t.Fatal("cross-family accepted")
	}
}

func TestEditedBattleSaveRestore(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	hp, mp := 17, 0
	traits := []string{}
	eq := []string{}
	d.Stages[0].Enemies = append(d.Stages[0].Enemies, content.Spawn{Character: "troop_footman", Class: "guards", Level: 1, X: 9, Y: 4, Facing: "NE", HP: &hp, MP: &mp, Traits: &traits, Equipment: &eq})
	e := New(d, 1)
	for n := 0; e.Observe().Phase != "battle"; n++ {
		if n > 100 {
			t.Fatal("battle not reached")
		}
		if e.Deploying() {
			finishDeployment(t, e)
			continue
		}
		o := e.Observe()
		c := Command{Kind: "next"}
		switch o.Dialogue.Kind {
		case "choice":
			for k := range o.Dialogue.Choices {
				c = Command{Kind: "choose", Option: k}
				break
			}
		case "preparation":
			c.Kind = "start"
		}
		if _, err = e.Apply(c); err != nil {
			t.Fatal(err)
		}
	}
	restored, err := Restore(d, e.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	u := restored.unit("troop_footman#1")
	if u == nil || u.HP != 17 || u.MP != 0 || u.Facing != "NE" || restored.char(u.Character).Class != "guards" {
		t.Fatalf("restored %+v", u)
	}
}
