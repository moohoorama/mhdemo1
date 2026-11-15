package autoplay

import (
	"fmt"
	"sort"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// baseline plays as cmd/_campaign does: seals into the first class offered, owners' pieces, the dearest trait,
// and the game's own evaluator in battle.
type baseline struct{ d *content.Data }

// Baseline is the game's current auto play, for comparison.
func Baseline(d *content.Data) Player { return baseline{d} }

func (b baseline) Name() string { return "base" }
func (b baseline) Prepare(e *core.Engine, try int) {
	var seals []string
	for id, it := range b.d.Items {
		if it.Effect == content.ItemPromote {
			seals = append(seals, id)
		}
	}
	sort.Strings(seals)
	for _, seal := range seals {
		for _, c := range Officers(b.d, e) {
			if e.Observe().Inventory[seal] < 1 {
				break
			}
			if classes, err := e.PromotionsAway(seal, c.ID); err == nil {
				e.Apply(core.Command{Kind: "item", Target: c.ID, Item: seal, Option: classes[0]})
			}
		}
	}
	for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
		if _, err := e.Apply(c); err != nil {
			break
		}
	}
}
func (b baseline) Learn(e *core.Engine) core.Command {
	c, _ := ai.Learning(e)
	return c
}
func (b baseline) Act(e *core.Engine, actor string) core.Command {
	c, err := ai.Choose(e, actor)
	if err != nil {
		return core.Command{Kind: "wait", Actor: actor}
	}
	return c
}

// Summary is one line for a campaign: how far it went, the battles won at the first try, the bonus objectives
// met and the level score.
func Summary(r Report) string {
	first, subs, total, tries := 0, 0, 0, 0
	for _, s := range r.Stages {
		tries += s.Tries
		if s.FirstWin {
			first++
		}
		if s.Won {
			subs += s.Subs
			total += s.SubsTotal
		}
	}
	return fmt.Sprintf("%-8s seed %d  cleared %-5v battles %2d  first-try %2d/%-2d  tries %3d  subs %2d/%-2d  level score %5.1f",
		r.Player, r.Seed, r.Cleared, len(r.Stages), first, len(r.Stages), tries, subs, total, r.LevelScore)
}

// Totals sums campaigns of one player.
type Totals struct {
	Player                          string
	Runs, Cleared                   int
	Battles, Wins, FirstWins, Tries int
	Subs, SubsTotal                 int
	LevelScore                      float64 // mean over the runs
}

// Total sums the reports.
func Total(reps []Report) Totals {
	t := Totals{Runs: len(reps)}
	for _, r := range reps {
		t.Player = r.Player
		if r.Cleared {
			t.Cleared++
		}
		for _, s := range r.Stages {
			t.Battles++
			t.Tries += s.Tries
			if s.Won {
				t.Wins++
				t.Subs += s.Subs
				t.SubsTotal += s.SubsTotal
			}
			if s.FirstWin {
				t.FirstWins++
			}
		}
		t.LevelScore += r.LevelScore / float64(len(reps))
	}
	return t
}

// Row is the totals as a markdown table row: runs cleared, battles won per run, the share of tries won, of
// battles won at the first try, of bonus objectives met, and the level score.
func (t Totals) Row() string {
	pct := func(a, b int) string {
		if b == 0 {
			return "-"
		}
		return fmt.Sprintf("%.0f%% (%d/%d)", 100*float64(a)/float64(b), a, b)
	}
	return fmt.Sprintf("| %s | %d/%d | %.1f | %s | %s | %s | %.1f |", t.Player, t.Cleared, t.Runs, float64(t.Wins)/float64(max(1, t.Runs)),
		pct(t.Wins, t.Tries), pct(t.FirstWins, t.Battles), pct(t.Subs, t.SubsTotal), t.LevelScore)
}
