package sim

import (
	"encoding/json"
	"reflect"
	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
	"testing"
)

func TestCampaignRestoresEveryCommandBoundary(t *testing.T) {
	d, err := content.Load("../../assets/content/campaign.json")
	if err != nil {
		t.Fatal(err)
	}
	e := core.New(d, 2) // any seed whose campaign is won
	counts := map[string]int{}
	for i := 0; i < 5000; i++ {
		o := e.Observe()
		restored, err := core.Restore(d, e.Snapshot())
		if err != nil {
			t.Fatalf("restore phase %s stage %d: %v", o.Phase, o.Stage, err)
		}
		if o.Phase == "complete" {
			if counts["join"] != 1 || counts["grant"] != 1 || counts["victory"] != 3 {
				t.Fatal("duplicate events", counts)
			}
			for _, id := range []string{"reward-B01", "reward-B02", "reward-B03", "join-간옹", "grant-고정도"} {
				if !o.Executed[id] {
					t.Fatal("missing ledger", id)
				}
			}
			if o.Warehouse["item_024"] != 0 {
				t.Fatal("sword not equipped")
			}
			return
		}
		var c core.Command
		if o.Phase == "scenario" {
			switch o.Dialogue.Kind {
			case "dialogue":
				c.Kind = "next"
			case "choice":
				c = core.Command{Kind: "choose", Option: "결의"}
			case "preparation":
				if o.Warehouse["item_024"] > 0 {
					c = core.Command{Kind: "equip", Actor: "유비", Item: "item_024"}
				} else {
					c.Kind = "start"
				}
			}
		} else if o.Phase == "result" {
			if o.Result != "victory" {
				t.Fatal("campaign lost", o.Stage, o.Round)
			}
			c.Kind = "continue"
		} else {
			found := false
			for _, u := range o.UnitViews {
				if u.Faction == o.Turn && u.HP > 0 && !u.Done {
					c, err = ai.Choose(e, u.ID)
					if err != nil {
						t.Fatal(err)
					}
					found = true
					break
				}
			}
			if !found {
				c.Kind = "end"
			}
		}
		a, err := e.Apply(c)
		if err != nil {
			t.Fatal(err)
		}
		b, err := restored.Apply(c)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a, b) {
			t.Fatal("event replay mismatch", i, c)
		}
		as, _ := json.Marshal(e.Snapshot())
		bs, _ := json.Marshal(restored.Snapshot())
		if string(as) != string(bs) {
			t.Fatal("state replay mismatch", i, c)
		}
		for _, v := range a.Events {
			counts[v.Kind]++
		}
	}
	t.Fatal("campaign did not finish")
}
func TestMultipleSeeds(t *testing.T) {
	d, err := content.Load("../../assets/content/campaign.json")
	if err != nil {
		t.Fatal(err)
	}
	won := 0
	for seed := uint64(1); seed <= 10; seed++ {
		r, err := Run(d, seed)
		if err != nil {
			t.Fatal(err)
		}
		if r.Result == "complete" {
			won++
			if len(r.Rounds) != 3 || r.Levels["유비"] < 2 || r.Levels["간옹"] < 2 {
				t.Fatal("growth", r)
			}
		}
	}
	if won < 5 {
		t.Fatalf("only %d/10 seeds won", won)
	}
}
