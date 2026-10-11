package duel

import (
	"encoding/json"
	"srpg/internal/content"
	"testing"
)

func TestCampaignScripts(t *testing.T) {
	scripts, err := LoadScripts("../../assets/duels.json")
	if err != nil {
		t.Fatal(err)
	}
	data, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	count := 0
	for _, stage := range data.Stages {
		for _, d := range stage.Duels {
			count++
			t.Run(d.ID, func(t *testing.T) {
				script, ok := scripts[d.ID]
				if !ok || script.Outcome != d.Outcome {
					t.Fatalf("missing or conflicting script for %+v", d)
				}
				encoded, _ := json.Marshal(script.Beats)
				if seen[string(encoded)] {
					t.Fatal("reused another duel's story")
				}
				seen[string(encoded)] = true
				story := scripts.Story(d.ID, d.Outcome)
				if !story.Sample(story.Duration()).Done {
					t.Fatal("story never ends")
				}
				for time := 0.; time < story.Duration(); time += .07 {
					before := story.Sample(time)
					story.Sample(story.Duration())
					if before != story.Sample(time) {
						t.Fatal("sampling depends on history")
					}
				}
			})
		}
	}
	if count != len(scripts) {
		t.Fatalf("%d scripts for %d campaign duels", len(scripts), count)
	}
	t.Run("one exchange kills", func(t *testing.T) {
		for _, id := range []string{"B01-duel-1", "B01-duel-2"} {
			strikes := 0
			for _, b := range scripts[id].Beats {
				switch b.Kind {
				case "hit", "block", "dodge", "crossing-charge", "critical", "backhit":
					strikes++
				}
			}
			if strikes != 1 {
				t.Fatalf("%s has %d exchanges", id, strikes)
			}
		}
	})
	t.Run("Dengmao confronts Zhang Fei before the first exchange", func(t *testing.T) {
		story := scripts.Story("B01-duel-1", "victory")
		var strike int
		for i, b := range story.Beats {
			if b.Kind == "critical" {
				strike = i
				break
			}
		}
		if strike == 0 {
			t.Fatal("missing first exchange")
		}
		ready := story.Sample(story.Start(strike) - .01)
		if ready.Actors[0].Position != -60 || ready.Actors[1].Position != 60 || ready.Actors[1].Facing != "NW" {
			t.Fatalf("fighters did not confront each other in the center: %+v", ready)
		}
		before := story.Sample(story.Start(strike) + .55)
		impact := story.Sample(story.Start(strike) + .57)
		if before.Flash != 0 || impact.Flash <= 0 || !impact.Critical {
			t.Fatal("first exchange must flash on contact")
		}
	})
	t.Run("long evenly matched duel", func(t *testing.T) {
		if scripts.Story("hulao-zhang-fei", "victory").Duration() <= 2*scripts.Story("B01-duel-1", "victory").Duration() {
			t.Fatal("draw should be a long duel")
		}
	})
	t.Run("Huaxiong is pursued and cut down", func(t *testing.T) {
		story := scripts.Story("sishui-guan-yu", "victory")
		for time := 0.; time < story.Duration(); time += .03 {
			r := story.Sample(time)
			if r.Actors[0].Animation == "exhausted" || r.Actors[0].Animation == "hit" {
				t.Fatalf("Guan Yu should remain composed at %g: %+v", time, r)
			}
		}
		before := at(story, "critical-block", .55)
		contact := at(story, "critical-block", .57)
		if before.Flash != 0 || contact.Flash <= 0 || !contact.Critical || contact.Actors[0].Animation != "block" || contact.Actors[0].Position != -60 || contact.ShakeSide != 1 {
			t.Fatalf("powerful blow was not calmly blocked: %+v", contact)
		}
		previous := [2]float64{-60, 120}
		for _, time := range []float64{.1, .6, 1.2, 1.8, 2.3} {
			r := at(story, "straight-chase", time)
			for i, a := range r.Actors[:2] {
				if a.Lane != 0 || a.Facing != "SE" || a.Position < previous[i] {
					t.Fatalf("chase must run straight toward enemy ranks: %+v", r)
				}
				previous[i] = a.Position
			}
		}
		cut := at(story, "backhit", .57)
		if cut.Actors[1].Animation != "hit" || cut.Actors[0].Facing != cut.Actors[1].Facing || !cut.Critical {
			t.Fatalf("missing pursuit strike: %+v", cut)
		}
		last := story.Sample(story.Duration())
		if last.Actors[0].Animation != "attack" || last.Actors[1].Animation != "exhausted" || !last.Done {
			t.Fatalf("wrong finish: %+v", last)
		}
		found := false
		for _, stage := range data.Stages {
			for _, d := range stage.Duels {
				if d.ID == "sishui-guan-yu" {
					found = d.Outcome == "victory" && d.EnemyResult == "death" && d.AllyResult == "stay"
				}
			}
		}
		if !found {
			t.Fatal("map outcome must match Huaxiong's death")
		}
	})
}

func TestScriptValidation(t *testing.T) {
	for name, script := range map[string]Script{
		"empty":            {Outcome: "draw"},
		"unknown outcome":  {Outcome: "escape", Beats: []Beat{{Kind: "draw", Duration: 1}}},
		"unknown motion":   {Outcome: "draw", Beats: []Beat{{Kind: "fly", Duration: 1}, {Kind: "draw", Duration: 1}}},
		"invalid side":     {Outcome: "draw", Beats: []Beat{{Kind: "draw", Side: 2, Duration: 1}}},
		"zero duration":    {Outcome: "draw", Beats: []Beat{{Kind: "draw"}}},
		"opposite winner":  {Outcome: "victory", Beats: []Beat{{Kind: "win", Side: 1, Duration: 1}}},
		"unfinished":       {Outcome: "draw", Beats: []Beat{{Kind: "talk", Duration: 1}}},
		"premature ending": {Outcome: "draw", Beats: []Beat{{Kind: "draw", Duration: 1}, {Kind: "draw", Duration: 1}}},
	} {
		t.Run(name, func(t *testing.T) {
			if script.Validate() == nil {
				t.Fatal("accepted invalid script")
			}
		})
	}
	t.Run("missing ID and mismatched outcome use resolved result", func(t *testing.T) {
		scripts := Scripts{"test": {Outcome: "victory", Beats: []Beat{{Kind: "win", Duration: 1}}}}
		for _, id := range []string{"test", "missing"} {
			s := scripts.Story(id, "defeat")
			last := s.Beats[len(s.Beats)-1]
			if last.Kind != "win" || last.Side != 1 {
				t.Fatal("fallback contradicts resolved defeat")
			}
		}
	})
}

func TestSingleAdvance(t *testing.T) {
	for side := range 2 {
		t.Run([]string{"ally attacks", "enemy attacks"}[side], func(t *testing.T) {
			s := Story{Beats: []Beat{{Kind: "advance", Side: side, Duration: 1}, {Kind: "critical", Side: 0, Duration: 1.3}, {Kind: "win", Side: 0, Duration: 2}}}
			r := s.Sample(.5)
			if r.Actors[1-side].Position != []float64{-260, 260}[1-side] {
				t.Fatal("defender moved during single advance")
			}
			before := s.Sample(1 + .55)
			after := s.Sample(1 + .57)
			if before.Impact != 0 || after.Impact <= 0 {
				t.Fatal("single exchange impact timing")
			}
		})
	}
}

func TestThreeBrothersAgainstOne(t *testing.T) {
	scripts, err := LoadScripts("../../assets/duels.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"hulao-zhang-fei", "hulao-guan-yu", "hulao-liu-bei"} {
		t.Run(id, func(t *testing.T) {
			story := scripts.Story(id, "victory")
			var arrival int
			for i, b := range story.Beats {
				if b.Kind == "reinforcements" {
					arrival = i
					break
				}
			}
			if arrival == 0 || len(scripts[id].Supports) != 2 {
				t.Fatal("missing brothers")
			}
			before := story.Sample(story.Start(arrival) - .01)
			joined := story.Sample(story.Start(arrival) + story.Beats[arrival].Duration)

			enemyX, enemyY := joined.Actors[1].ScreenPosition()
			for i := 2; i < 4; i++ {
				sx, sy := before.Actors[i].ScreenPosition()
				x, y := joined.Actors[i].ScreenPosition()
				if before.Actors[i].Visible || !joined.Actors[i].Visible || sy < 850 || y >= sy {
					t.Fatal("support must enter from a lower screen corner")
				}
				if i == 2 && (sx >= 0 || x >= enemyX || y <= enemyY) || i == 3 && (sx <= 1280 || x <= enemyX) {
					t.Fatal("support entered or surrounded from the wrong side")
				}
			}
			// The enemy must lie inside the triangle formed by all three brothers.
			vertices := []int{0, 2, 3}
			sign := 0.0
			for i, v := range vertices {
				x, y := joined.Actors[v].ScreenPosition()
				nx, ny := joined.Actors[vertices[(i+1)%3]].ScreenPosition()
				cross := (nx-x)*(enemyY-y) - (ny-y)*(enemyX-x)
				if i == 0 {
					sign = cross
				} else if cross*sign <= 0 {
					t.Fatal("three brothers do not enclose the enemy")
				}
			}
			previousY := [2]float64{850, 850}
			for _, local := range []float64{.01, .5, 1, 1.7} {
				state := at(story, "reinforcements", local)
				for i := 2; i < 4; i++ {
					_, y := state.Actors[i].ScreenPosition()
					if y >= previousY[i-2] && local > .01 {
						t.Fatal("reinforcement path must approach from below")
					}
					previousY[i-2] = y
				}
			}
			for i, b := range story.Beats {
				if b.Side > 1 && b.Kind == "block" {
					blocked := story.Sample(story.Start(i) + .57)
					if blocked.Actors[1].Facing != blocked.Actors[1].facingToward(blocked.Actors[b.Side]) {
						t.Fatal("defender did not turn toward the flank attacker")
					}
				}
			}

			for _, time := range []float64{.1, .8, 1.6, 2.1} {
				state := at(story, "retreat", time)
				if state.Actors[1].Lane != 0 || state.Actors[1].Facing != "SE" || state.Actors[1].Animation != "walk" {
					t.Fatalf("not a straight retreat: %+v", state)
				}
				for _, side := range []int{0, 2, 3} {
					if state.Actors[side].Animation != "idle" {
						t.Fatal("brothers must hold their ground during retreat")
					}
				}
			}
			last := story.Sample(story.Duration())
			if last.Actors[1].Position != 160 || last.Actors[1].Animation == "exhausted" {
				t.Fatal("retreat must stop at the formation entrance without exhaustion")
			}
			x, y := last.Actors[1].ScreenPosition()
			if .5*x+y > 1074-160 {
				t.Fatal("retreat crossed into the front rank")
			}
			for time := 0.; time < story.Duration(); time += .04 {
				state := story.Sample(time)
				for _, a := range state.Actors {
					if a.Animation == "exhausted" || a.Animation == "hit" {
						t.Fatal("these duels must end without a defeated pose")
					}
				}
			}
		})
	}
	t.Run("heavy strike is evaded without hitting", func(t *testing.T) {
		s := scripts.Story("hulao-guan-yu", "victory")
		r := at(s, "critical-dodge", .65)
		if !r.Critical || r.Flash <= 0 || r.Impact != 0 || r.Actors[1].Animation != "walk" || r.Actors[1].Position <= 60 {
			t.Fatalf("heavy evasion: %+v", r)
		}
	})
	t.Run("Liu Bei yields while blocking", func(t *testing.T) {
		s := scripts.Story("hulao-liu-bei", "victory")
		before := at(s, "yielding-block", .55)
		after := at(s, "yielding-block", 1.1)
		if after.Actors[0].Animation != "block" || after.Actors[0].Position >= before.Actors[0].Position {
			t.Fatalf("block does not show disadvantage: %+v", after)
		}
	})
	t.Run("Zhang Fei holds his ground", func(t *testing.T) {
		s := scripts.Story("hulao-zhang-fei", "victory")
		for i, b := range s.Beats {
			if b.Kind == "critical-block" {
				r := s.Sample(s.Start(i) + .57)
				if r.Actors[0].Position < -60 || r.Actors[1].Position > 60 {
					t.Fatal("equal combat yielded ground")
				}
			}
		}
	})
}
