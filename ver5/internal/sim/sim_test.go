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
	d, err := content.Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	for seed := uint64(1); seed <= 30; seed++ {
		if ok := restoresEveryBoundary(t, d, seed); ok {
			return
		}
	}
	t.Fatal("no seed finished the campaign")
}

// restoresEveryBoundary plays a seed to its end, checking at each command that a restored engine replays it
// identically; it reports whether the campaign was won.
func restoresEveryBoundary(t *testing.T, d *content.Data, seed uint64) bool {
	e := core.New(d, seed)
	counts := map[string]int{}
	for i := 0; i < 5000; i++ {
		o := e.Observe()
		restored, err := core.Restore(d, e.Snapshot())
		if err != nil {
			t.Fatalf("restore phase %s stage %d: %v", o.Phase, o.Stage, err)
		}
		if o.Phase == "complete" {
			if counts["join"] != 1 || counts["grant"] != 1 || counts["victory"] != 2 {
				t.Fatal("duplicate events", counts)
			}
			for _, id := range []string{"reward-S1", "reward-S2", "join-sage", "grant-spear"} {
				if !o.Executed[id] {
					t.Fatal("missing ledger", id)
				}
			}
			return true
		}
		var c core.Command
		if o.Phase == "scenario" {
			switch o.Dialogue.Kind {
			case "dialogue":
				c.Kind = "next"
			case "choice":
				c = core.Command{Kind: "choose", Option: "go"}
			case "preparation":
				if limit := d.Stages[o.Stage].Limit; len(o.Deployment) > limit {
					c = core.Command{Kind: "deploy", Deployment: o.Deployment[:limit]}
				} else {
					c.Kind = "start"
				}
			}
		} else if o.Phase == "result" {
			if o.Result != "victory" {
				return false
			}
			c.Kind = "continue"
		} else {
			c, err = ai.NextWithDuels(e, true)
			if err != nil {
				t.Fatal(err)
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
	return false
}
func TestMultipleSeeds(t *testing.T) {
	d, err := content.Load("../testdata/mini")
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
			if len(r.Rounds) != 2 || r.Levels["hero"] < 2 || r.Levels["sage"] < 1 {
				t.Fatal("growth", r)
			}
		}
	}
	if won < 1 {
		t.Fatalf("%d/10 seeds won", won)
	}
}
