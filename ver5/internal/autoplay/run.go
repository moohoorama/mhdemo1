// Package autoplay plays the whole campaign as the player would — the preparation before each battle
// (promotions, gear, deputies, traits, the sortie), the battles and the retries — for the AIs that try to clear
// the hardest difficulty with the party grown as far as it can be, and scores how they did.
package autoplay

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/storage"
)

// Player makes the player's decisions. Battle commands of the player's own units go to Act; the AI allies and
// the enemy play by the game's own policy.
type Player interface {
	Name() string
	// Prepare readies the party at a battle node (try counts from 1; a retry starts again from the checkpoint).
	Prepare(e *core.Engine, try int)
	// Learn answers an open learning window with learn or learn_close.
	Learn(e *core.Engine) core.Command
	// Act is the command of the player's unit actor, whose turn it is.
	Act(e *core.Engine, actor string) core.Command
}

// Options sets how a campaign is played.
type Options struct {
	Difficulty   string
	Seed         uint64
	Tries        int                // attempts per battle before the campaign counts as stuck (5)
	Stages       int                // battles to fight at most; 0 all
	Log          io.Writer          // one line per battle; nil quiet
	Trace        io.Writer          // optional JSONL: prepared snapshot, commands, events and units after each command
	TraceStage   string             // stage ID to trace; empty traces every stage
	BeforeBattle func(*core.Engine) // observe the real, unprepared campaign state; must not mutate it
}

// StageReport is how one battle went.
type StageReport struct {
	Stage           string
	Order           int
	Tries           int
	Won, FirstWin   bool
	Subs, SubsTotal int // bonus objectives met in the battle won, of those the stage has
	Rounds          int
	Levels          map[string]int // playable officers' levels after it
}

// Report is how a whole campaign went.
type Report struct {
	Player          string
	Seed            uint64
	ContentHash     string
	ExpectedBattles int
	EndNode         string
	Stages          []StageReport
	Cleared         bool
	Levels          map[string]int
	LevelScore      float64
	UnbeatenStages  int
	FirstDefeat     string
}

// Run plays the campaign from its start with p as the player.
func Run(d *content.Data, p Player, opt Options) (Report, error) {
	if opt.Tries == 0 {
		opt.Tries = 5
	}
	rep := Report{Player: p.Name(), Seed: opt.Seed, ContentHash: d.Hash, ExpectedBattles: len(BattleNodes(d))}
	e, err := core.NewGame(d, opt.Seed, opt.Difficulty)
	if err != nil {
		return rep, err
	}
	e.SetDamagePct(storage.DefaultScalePct) // the settings the game starts with
	e.SetXPPct(storage.DefaultScalePct)
	apply := func(c core.Command) error {
		if _, err := e.Apply(c); err != nil {
			return fmt.Errorf("%s %+v: %w", e.Observe().Node, c, err)
		}
		return nil
	}
	for range 20000 {
		o := e.Observe()
		if o.Node != "" {
			rep.EndNode = o.Node
		}
		switch {
		case o.Phase == "complete":
			rep.Cleared = len(rep.Stages) == rep.ExpectedBattles
			if o.Ending != "" {
				rep.EndNode = o.Ending
			}
			rep.Levels = Levels(d, e)
			rep.LevelScore = LevelScore(d, rep.Levels)
			return rep, nil
		case len(o.Learning) > 0:
			if err := apply(p.Learn(e)); err != nil {
				return rep, err
			}
		case o.Phase == "scenario" && (o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate"):
			if err := apply(core.Command{Kind: "next"}); err != nil {
				return rep, err
			}
		case e.Walking():
			commands := e.ExploreNext()
			if p, ok := p.(interface {
				Explore(*core.Engine) []core.Command
			}); ok {
				commands = p.Explore(e)
			}
			for _, c := range commands {
				if err := apply(c); err != nil {
					return rep, err
				}
			}
		case o.Phase == "scenario" && o.Dialogue.Kind == "choice":
			if err := apply(ChooseStory(e, p)); err != nil {
				return rep, err
			}
		case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
			if opt.Stages > 0 && len(rep.Stages) >= opt.Stages {
				rep.Levels = Levels(d, e)
				rep.LevelScore = LevelScore(d, rep.Levels)
				return rep, nil
			}
			st := d.Stages[o.Stage]
			if opt.BeforeBattle != nil {
				opt.BeforeBattle(e)
			}
			sr := StageReport{Stage: st.ID, Order: st.Order}
			before := e.Snapshot()
			for try := 1; ; try++ {
				sr.Tries = try
				p.Prepare(e, try)
				if checked, ok := p.(interface{ AutoplayError() error }); ok && checked.AutoplayError() != nil {
					return rep, checked.AutoplayError()
				}
				var trace *json.Encoder
				if opt.Trace != nil && (opt.TraceStage == "" || opt.TraceStage == st.ID) {
					trace = json.NewEncoder(opt.Trace)
					if err := trace.Encode(struct {
						Stage    string
						Try      int
						Snapshot core.State
					}{st.ID, try, e.Snapshot()}); err != nil {
						return rep, err
					}
				}
				if err := apply(core.Command{Kind: "start"}); err != nil {
					return rep, err
				}
				if err := battle(e, p, 20000, trace); err != nil {
					return rep, err
				}
				if e.Observe().Result != "victory" && rep.FirstDefeat == "" {
					rep.FirstDefeat = st.ID
				}
				if e.Observe().Result == "victory" || try == opt.Tries {
					break
				}
				before.RNG += 0x9e3779b97f4a7c15 // a retry plays out differently
				if e, err = core.Restore(d, before); err != nil {
					return rep, err
				}
			}
			o = e.Observe()
			sr.Won, sr.FirstWin, sr.Rounds = o.Result == "victory", o.Result == "victory" && sr.Tries == 1, o.Round
			for _, ok := range e.SubDone() {
				sr.SubsTotal++
				if ok {
					sr.Subs++
				}
			}
			for len(e.Observe().Learning) > 0 {
				if err := apply(p.Learn(e)); err != nil {
					return rep, err
				}
			}
			sr.Levels = Levels(d, e)
			rep.Stages = append(rep.Stages, sr)
			if rep.FirstDefeat == "" && sr.FirstWin {
				rep.UnbeatenStages++
			}
			if opt.Log != nil {
				fmt.Fprintf(opt.Log, "%-8s %-4s s%d %-4s tries %d subs %d/%d round %d score %.1f\n", p.Name(), st.ID, opt.Seed,
					map[bool]string{true: "win", false: "LOSS"}[sr.Won], sr.Tries, sr.Subs, sr.SubsTotal, sr.Rounds, LevelScore(d, sr.Levels))
			}
			if !sr.Won {
				rep.Levels = sr.Levels
				rep.LevelScore = LevelScore(d, rep.Levels)
				return rep, nil
			}
			if err := apply(core.Command{Kind: "continue"}); err != nil {
				return rep, err
			}
		default:
			return rep, fmt.Errorf("stuck in phase %s at node %s (%s)", o.Phase, o.Node, o.Dialogue.Kind)
		}
	}
	return rep, fmt.Errorf("campaign did not end")
}

// Battle plays the battle on until its result, the player's units by p and every other unit by the game's AI.
func Battle(e *core.Engine, p Player, limit int) error {
	return battle(e, p, limit, nil)
}

func battle(e *core.Engine, p Player, limit int, trace *json.Encoder) error {
	for range limit {
		o := e.Observe()
		if o.Phase == "result" {
			return nil
		}
		c, err := next(e, o, p)
		if err != nil {
			return err
		}
		r, err := e.Apply(c)
		if err != nil {
			return fmt.Errorf("%s round %d %+v: %w", o.Node, o.Round, c, err)
		}
		if trace != nil {
			after := e.Observe()
			if err := trace.Encode(struct {
				Command                   core.Command
				Round                     int
				Turn, Army, Phase, Result string
				Units                     []core.UnitView
				Events                    []core.Event
			}{c, after.Round, after.Turn, after.Army, after.Phase, after.Result, after.UnitViews, r.Events}); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("battle did not end in %d commands", limit)
}

// next is the command the battle waits for: a learning answer, the deployment, the first ready unit's choice or
// the end of the faction's turn.
func next(e *core.Engine, o core.Observation, p Player) (core.Command, error) {
	if len(o.Learning) > 0 {
		return p.Learn(e), nil
	}
	if o.Phase == "deployment" {
		cs, err := e.DeploymentPlan()
		if err != nil {
			return core.Command{}, err
		}
		return cs[0], nil
	}
	for _, u := range o.UnitViews {
		if u.ActsIn(o.Turn, o.Army) && u.HP > 0 && !u.Done {
			if mine(u) {
				return p.Act(e, u.ID), nil
			}
			return ai.ChooseWithDuels(e, u.ID, true)
		}
	}
	return core.Command{Kind: "end"}, nil
}

// mine reports whether the player commands the unit.
func mine(u core.UnitView) bool { return u.Faction == "ally" && !u.AI }

// Officers are the living playable officers of the party, highest level first (ties by experience, then id).
func Officers(d *content.Data, e *core.Engine) []core.Character {
	var out []core.Character
	for _, c := range e.Observe().Characters {
		if !c.Temp && !c.Dead && d.Characters[c.Def].Playable {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Level != out[j].Level {
			return out[i].Level > out[j].Level
		}
		if out[i].XP != out[j].XP {
			return out[i].XP > out[j].XP
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Levels are the playable officers' levels.
func Levels(d *content.Data, e *core.Engine) map[string]int {
	out := map[string]int{}
	for _, c := range Officers(d, e) {
		out[c.ID] = c.Level
	}
	return out
}

// ScoreN is the level score's whole: the five highest officers share half of it, the next five a quarter and
// the next ten an eighth, each officer's share counted at the level it stands at against the last battle's
// stage level.
const ScoreN = 100.0

// RankWeight is one officer's share of ScoreN by its place in the party's level order (from 0).
func RankWeight(rank int) float64 {
	switch {
	case rank < 5:
		return ScoreN / 2 / 5
	case rank < 10:
		return ScoreN / 4 / 5
	case rank < 20:
		return ScoreN / 8 / 10
	}
	return 0
}

// LevelScore weighs the levels by rank: Σ RankWeight(rank) × level / the last battle's stage level.
func LevelScore(d *content.Data, levels map[string]int) float64 {
	ls := make([]int, 0, len(levels))
	for _, l := range levels {
		ls = append(ls, l)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ls)))
	ref := float64(RefLevel(d))
	s := 0.0
	for i, l := range ls {
		s += RankWeight(i) * float64(l) / ref
	}
	return s
}

// RefLevel is the stage level of the campaign's last battle.
func RefLevel(d *content.Data) int {
	last := 0
	for _, st := range d.Stages {
		last = max(last, st.Order)
	}
	return d.Rules.StageLevel(last)
}

// Grow is how much each officer's experience counts toward the level score now: its rank weight against the top's.
func Grow(d *content.Data, e *core.Engine) map[string]float64 {
	out := map[string]float64{}
	for i, c := range Officers(d, e) {
		out[c.ID] = RankWeight(i) / RankWeight(0)
	}
	return out
}
