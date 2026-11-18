package autoplay

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"

	"srpg/internal/content"
	"srpg/internal/core"
)

// Variety prepares the party for the end-to-end test so that one run of the story tries as much of the game as it
// can, from one seed: the sortie and the deputies drawn at random, the pieces in the warehouse no one has worn yet
// put on first, 부관임명 bought before anything else, and then for every officer a trait drawn at random as its goal,
// its prerequisites bought on the way and the points saved until it can be had. A retry draws again; from LastTry on
// it prepares as Fallback does (the tuned rules when its playouts are off), so that the draw never ends the story.
type Variety struct {
	D        *content.Data
	Fallback *Searcher
	LastTry  int // the try that prepares as Fallback does
	rng      *rand.Rand
	goals    map[string]string // officer → the trait it saves for
	worn     map[string]bool   // pieces put on at least once
	Learned  map[string]int    // trait → how many times an officer took it
	Equipped map[string]int    // piece → how many times it was put on
	Deputies map[string]int    // "officer+deputy" → battles prepared so
	Sorties  map[string]int    // officer → battles it was sent to
	Log      []string          // what each preparation chose, for a failure's report
}

// NewVariety is a Variety drawing from seed.
func NewVariety(d *content.Data, fallback *Searcher, seed uint64, lastTry int) *Variety {
	return &Variety{D: d, Fallback: fallback, LastTry: lastTry, rng: rand.New(rand.NewPCG(seed, seed^0x5eed)),
		goals: map[string]string{}, worn: map[string]bool{}, Learned: map[string]int{}, Equipped: map[string]int{},
		Deputies: map[string]int{}, Sorties: map[string]int{}}
}

// PrepareCommands readies the party at a battle node and gives the commands it took, as Searcher.PrepareCommands.
func (v *Variety) PrepareCommands(e *core.Engine, try int) []core.Command {
	if try >= v.LastTry && v.Fallback != nil {
		v.Log = append(v.Log, fmt.Sprintf("try %d: the search player's preparation", try))
		return v.Fallback.PrepareCommands(e, try)
	}
	var out []core.Command
	apply := func(c core.Command) bool {
		if _, err := e.Apply(c); err != nil {
			return false
		}
		out = append(out, c)
		return true
	}
	var g Genome
	copy(g[:], CampaignPlayer(v.D).G[:])
	v.sortie(e, apply)
	g.promote(v.D, e, apply)
	for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
		if !apply(c) {
			break
		}
	}
	v.deputies(e, apply)
	v.gear(e, apply)
	for _, c := range Officers(v.D, e) {
		for range 10 {
			t := v.next(e, c.ID)
			if t == "" || !apply(core.Command{Kind: "learn", Actor: c.ID, Trait: t}) {
				break
			}
			v.Learned[t]++
		}
	}
	v.Log = append(v.Log, fmt.Sprintf("try %d: %v", try, e.Observe().Deployment))
	return out
}

// Learn answers a learning window: the first waiting officer buys the next step to its goal, or saves.
func (v *Variety) Learn(e *core.Engine) core.Command {
	id := e.Observe().Learning[0]
	if t := v.next(e, id); t != "" {
		v.Learned[t]++
		return core.Command{Kind: "learn", Actor: id, Trait: t}
	}
	return core.Command{Kind: "learn_close", Actor: id}
}

// sortie sends the lord and those the stage requires, then officers drawn at random up to the stage's count.
func (v *Variety) sortie(e *core.Engine, apply func(core.Command) bool) {
	st := v.D.Stages[e.Observe().Stage]
	var ids, rest []string
	for _, c := range Officers(v.D, e) {
		if slices.Contains(st.Required, c.ID) || v.D.Characters[c.Def].Lord {
			ids = append(ids, c.ID)
		} else if !slices.Contains(st.Reserve, c.ID) { // the reserve goes on top of the count
			rest = append(rest, c.ID)
		}
	}
	v.rng.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	for _, id := range rest {
		if len(ids) >= st.Max {
			break
		}
		ids = append(ids, id)
	}
	if apply(core.Command{Kind: "deploy", Deployment: ids}) {
		for _, id := range ids {
			v.Sorties[id]++
		}
	}
}

// deputies gives each officer who may name one an officer left on the bench, drawn at random.
func (v *Variety) deputies(e *core.Engine, apply func(core.Command) bool) {
	for _, c := range Officers(v.D, e) {
		dep := e.Observe().Deployment
		if !e.CanDeputize(c.ID) || charOf(e, c.ID).Deputy != "" || !slices.Contains(dep, c.ID) {
			continue
		}
		var bench []string
		for _, x := range Officers(v.D, e) {
			if x.ID != c.ID && !slices.Contains(dep, x.ID) && e.DeputyCheck(c.ID, x.ID) == nil {
				bench = append(bench, x.ID)
			}
		}
		if len(bench) == 0 {
			continue
		}
		x := bench[v.rng.IntN(len(bench))]
		if apply(core.Command{Kind: "deputy", Actor: c.ID, Target: x}) {
			v.Deputies[c.ID+"+"+x]++
		}
	}
}

// gear puts on, slot by slot for each officer sent out, a piece from the warehouse no one has worn yet, drawn at
// random among those the officer may carry; an owner keeps its own piece.
func (v *Variety) gear(e *core.Engine, apply func(core.Command) bool) {
	for _, id := range e.Observe().Deployment {
		for _, slot := range []string{"weapon", "armor", "accessory"} {
			o := charOf(e, id)
			if cur := o.Equipment[slot]; cur != "" && v.D.Equipment[cur].Owner == o.Def {
				continue
			}
			var fresh []string
			for _, item := range sortedKeys(e.Observe().Warehouse) {
				eq := v.D.Equipment[item]
				if e.Observe().Warehouse[item] < 1 || eq.Slot != slot || v.worn[item] || eq.Owner != "" && eq.Owner != o.Def ||
					!v.D.Carries(v.D.Classes[o.Class], o.Learned, eq) {
					continue
				}
				fresh = append(fresh, item)
			}
			if len(fresh) == 0 {
				continue
			}
			item := fresh[v.rng.IntN(len(fresh))]
			if apply(core.Command{Kind: "equip", Actor: id, Item: item}) {
				v.worn[item] = true
				v.Equipped[item]++
			}
		}
	}
}

// next is the trait officer id buys now on the way to its goal, or "" while it saves its points. 부관임명 comes
// first; then a goal drawn among the traits the officer may come to learn, those no officer has taken yet first;
// a goal's missing prerequisites are bought before it.
func (v *Variety) next(e *core.Engine, id string) string {
	o := charOf(e, id)
	owned := func(t string) bool {
		if slices.Contains(o.Learned, t) || o.TraitsOverride != nil && slices.Contains(*o.TraitsOverride, t) {
			return true
		}
		for _, item := range o.Equipment {
			if slices.Contains(v.D.Equipment[item].Traits, t) {
				return true
			}
		}
		for _, deputy := range e.Observe().Characters {
			if deputy.ID == o.Deputy && !deputy.Dead {
				return slices.Contains(deputy.Learned, t) || deputy.TraitsOverride != nil && slices.Contains(*deputy.TraitsOverride, t)
			}
		}
		return false
	}
	pool := v.pool(o)
	goal := v.goals[id]
	for _, t := range pool { // the trait that lets an officer name a deputy comes first
		if v.D.Traits[t].Effects[content.FxDeputyAppoint] > 0 && !owned(t) {
			goal = t
		}
	}
	if goal == "" || owned(goal) || !v.reachable(goal, pool, owned, 0) {
		var fresh, any []string
		for _, t := range pool {
			if owned(t) || !v.reachable(t, pool, owned, 0) {
				continue
			}
			any = append(any, t)
			if v.Learned[t] == 0 {
				fresh = append(fresh, t)
			}
		}
		if len(fresh) > 0 {
			any = fresh
		}
		if len(any) == 0 {
			return ""
		}
		goal = any[v.rng.IntN(len(any))]
	}
	v.goals[id] = goal
	step := v.step(goal, owned)
	if o.Points < v.D.Traits[step].Cost || !slices.Contains(e.Learnable(id), step) {
		return "" // saving for it
	}
	return step
}

// pool is every trait the officer may learn some day: its own and its class line's, with no cost-free gifts.
func (v *Variety) pool(o core.Character) []string {
	var out []string
	for _, t := range append(slices.Clone(v.D.Characters[o.Def].Learn), v.D.Classes[o.Class].AllPool()...) {
		if v.D.Traits[t].Cost > 0 && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

// reachable reports whether trait t and every prerequisite it lacks lie in the pool.
func (v *Variety) reachable(t string, pool []string, owned func(string) bool, depth int) bool {
	if owned(t) {
		return true
	}
	if depth > 8 || !slices.Contains(pool, t) {
		return false
	}
	for _, q := range v.D.Traits[t].Requires {
		if !v.reachable(q, pool, owned, depth+1) {
			return false
		}
	}
	return true
}

// step is the first trait to buy toward t: a missing prerequisite, deepest first, or t itself.
func (v *Variety) step(t string, owned func(string) bool) string {
	for _, q := range v.D.Traits[t].Requires {
		if !owned(q) {
			return v.step(q, owned)
		}
	}
	return t
}
