// Temporary: measures how strong each trait is by pitting a class against itself on an open field —
// 1:1, 3:3, 5:5 — with the trait on one side, the game's AI on both sides. Every seed is fought twice,
// the trait once with the player's army (moving first) and once with the enemy, so that neither the
// first move nor the two AIs favour it. Prints win rate, HP margin (the trait side's HP share left minus
// the other side's) and rounds, against the same fights without the trait.
//
//	go run ./cmd/_traitsim -class light_cavalry -traits opening,charge_aim -seeds 50
//	go run ./cmd/_traitsim -class light_infantry,light_cavalry,archer,scholar -j 6 -json out.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// template is the soldier template a class is fought with, for its stats and gear: of the templates serving in
// the first class of the class's promotion line, the one the stages field most.
func template(d *content.Data, promotes map[string][]string, class string) string {
	root := class
	for changed := true; changed; {
		changed = false
		for from, to := range promotes {
			if slices.Contains(to, root) {
				root, changed = from, true
			}
		}
	}
	used := map[string]int{}
	for _, st := range d.Stages {
		for _, sp := range append(slices.Clone(st.Enemies), st.Party...) {
			used[sp.Character]++
		}
	}
	best := ""
	for _, id := range slices.Sorted(maps.Keys(d.Characters)) {
		if c := d.Characters[id]; c.Template && c.Class == root && (best == "" || used[id] > used[best]) {
			best = id
		}
	}
	return best
}

var fullMP, buffIdle bool

type tally struct {
	Wins, Losses, Draws, Fights int
	FirstWins, FirstFights      int // the fights where the measured side is the player's army, which moves first
	Margin, Rounds              float64
}

func (t *tally) add(o tally) {
	t.Wins += o.Wins
	t.FirstWins += o.FirstWins
	t.FirstFights += o.FirstFights
	t.Losses += o.Losses
	t.Draws += o.Draws
	t.Fights += o.Fights
	t.Margin += o.Margin
	t.Rounds += o.Rounds
}
func (t tally) WinRate() float64 { return float64(t.Wins) / float64(max(1, t.Fights)) }
func (t tally) AvgMargin() float64 {
	return t.Margin / float64(max(1, t.Fights))
}
func (t tally) FirstRate() float64 { return float64(t.FirstWins) / float64(max(1, t.FirstFights)) }
func (t tally) SecondRate() float64 {
	return float64(t.Wins-t.FirstWins) / float64(max(1, t.Fights-t.FirstFights))
}
func (t tally) AvgRounds() float64 { return t.Rounds / float64(max(1, t.Fights)) }

type Row struct {
	Class, Trait, Name string
	Size               int
	Formation          string
	Gaps               string
	WinRate, Margin    float64
	FirstRate          float64 // win rate moving first
	SecondRate         float64 // win rate moving second
	Rounds             float64
	Draws              int
	Fights             int
}

// setup is what every fight shares: the content, the level, the start gaps, the seeds.
type setup struct {
	d     *content.Data
	level int
	gaps  []int
	seeds int
}

// job is one line of the table: a class, a side size, a formation and a trait ("" for none).
type job struct {
	class, tpl, trait, formation string
	n                            int
}

func main() {
	dir := flag.String("content", "assets/data", "content directory")
	classes := flag.String("class", "", "classes, comma separated (required)")
	traitList := flag.String("traits", "", "traits, comma separated; none takes the class line's pools and every officer's Learn of that line")
	sizes := flag.String("sizes", "1,3,5", "units per side")
	formations := flag.String("formations", "tight,spread", "tight: side by side; spread: one empty cell between neighbours")
	seeds := flag.Int("seeds", 30, "seeds; each is fought twice, the trait on either side")
	level := flag.Int("level", 30, "level of every unit")
	gaps := flag.String("gaps", "6,12", "empty columns between the two lines at the start, comma separated, pooled")
	workers := flag.Int("j", max(1, runtime.NumCPU()/2), "fights run at once")
	flag.BoolVar(&fullMP, "fullmp", true, "every unit starts with full MP, so that spells and arts count in a short fight")
	flag.BoolVar(&buffIdle, "buff", true, "a unit that would end its action without attacking casts a self buff it does not have yet")
	show := flag.Bool("show", false, "print both sides' numbers and the blows of the first fight and stop")
	out := flag.String("json", "", "also write the rows to this file")
	flag.Parse()
	d, err := content.Load(*dir)
	check(err)
	ai.Player = nil // both sides by the enemy's greedy AI
	d.Rules.StageLevelBase = 0
	d.Rules.EnemyStock, d.Rules.AllyStock = nil, nil
	d.Campaign.Inventory = nil
	lines := map[string][]string{}
	for id, c := range d.Classes {
		lines[id] = c.Promotes
	}
	d.Classes = maps.Clone(d.Classes)
	for id, c := range d.Classes {
		c.Promotes = nil // enemies would be promoted by their level
		c.Plan = nil     // and hold their class's traits
		d.Classes[id] = c
	}
	su := setup{d: d, level: *level, gaps: splitInts(*gaps), seeds: *seeds}
	var jobs []job
	for _, class := range splitList(*classes) {
		if _, ok := d.Classes[class]; !ok {
			check(fmt.Errorf("no class %s", class))
		}
		tpl := template(d, lines, class)
		if tpl == "" {
			check(fmt.Errorf("no soldier template for class %s", class))
		}
		home := map[string]string{} // trait -> the class it is fought in
		for _, tr := range splitList(*traitList) {
			home[tr] = class
		}
		if len(home) == 0 {
			home = lineTraits(d, lines, class)
		}
		fought := []string{}
		for _, tr := range slices.Sorted(maps.Keys(home)) {
			if !slices.Contains(fought, home[tr]) {
				fought = append(fought, home[tr])
			}
		}
		for _, n := range splitInts(*sizes) {
			for _, f := range splitList(*formations) {
				for _, c := range fought {
					jobs = append(jobs, job{class: c, tpl: tpl, formation: f, n: n})
				}
				for _, tr := range slices.Sorted(maps.Keys(home)) {
					jobs = append(jobs, job{class: home[tr], tpl: tpl, trait: tr, formation: f, n: n})
				}
			}
		}
	}
	if *show {
		j := jobs[min(1, len(jobs)-1)]
		fight(su, j, "ally", su.gaps[0], 1, true)
		return
	}
	results := make([]tally, len(jobs))
	next, done := make(chan int), make(chan int)
	for w := 0; w < *workers; w++ {
		go func() {
			for i := range next {
				results[i] = run(su, jobs[i])
				done <- i
			}
		}()
	}
	go func() {
		for i := range jobs {
			next <- i
		}
		close(next)
	}()
	for k := 1; k <= len(jobs); k++ {
		i := <-done
		j, t := jobs[i], results[i]
		fmt.Printf("[%d/%d] %-15s %d:%d %-6s %-12s win %5.1f%% (선공 %5.1f%% 후공 %5.1f%%)  margin %+6.1f%%p  rounds %4.1f  draws %d\n",
			k, len(jobs), j.class, j.n, j.n, j.formation, name(d, j.trait), t.WinRate()*100, t.FirstRate()*100, t.SecondRate()*100, t.AvgMargin()*100, t.AvgRounds(), t.Draws)
	}
	if *out != "" {
		rows := make([]Row, len(jobs))
		for i, j := range jobs {
			t := results[i]
			rows[i] = Row{Class: j.class, Trait: j.trait, Name: name(d, j.trait), Size: j.n, Formation: j.formation, Gaps: *gaps, WinRate: t.WinRate(),
				FirstRate: t.FirstRate(), SecondRate: t.SecondRate(), Margin: t.AvgMargin(), Rounds: t.AvgRounds(), Draws: t.Draws, Fights: t.Fights}
		}
		b, err := json.MarshalIndent(rows, "", " ")
		check(err)
		check(os.WriteFile(*out, b, 0o644))
	}
}

func name(d *content.Data, trait string) string {
	if trait == "" {
		return "(없음)"
	}
	return d.Traits[trait].Name
}

// lineTraits are the learnable traits that make sense for the class, each with the class it is fought in: the
// pools of its whole promotion line, in the class whose pool it is (a 무쌍 needs a chariot's MP), and the Learn
// lists of the officers who start in that line, in the class they start in.
func lineTraits(d *content.Data, promotes map[string][]string, class string) map[string]string {
	line := map[string]bool{}
	var walk func(string)
	walk = func(c string) {
		if line[c] {
			return
		}
		line[c] = true
		for _, p := range promotes[c] {
			walk(p)
		}
	}
	walk(class)
	out := map[string]string{}
	addAll := func(ids []string, in string) {
		for _, id := range ids {
			if _, ok := out[id]; !ok && d.Traits[id].Cost > 0 && !offField(d.Traits[id]) {
				out[id] = in
			}
		}
	}
	for _, c := range slices.Sorted(maps.Keys(line)) {
		addAll(d.Classes[c].Pool, c)
	}
	for _, id := range slices.Sorted(maps.Keys(d.Characters)) {
		if c := d.Characters[id]; c.Playable && line[c.Class] {
			addAll(c.Learn, class)
		}
	}
	return out
}

func run(su setup, j job) tally {
	var t tally
	for _, gap := range su.gaps {
		for seed := 1; seed <= su.seeds; seed++ {
			for _, side := range []string{"ally", "enemy"} {
				t.add(fight(su, j, side, gap, uint64(seed), false))
			}
		}
	}
	return t
}

// line is where the side's units stand, top to bottom, in column x of a map h cells high.
func line(formation string, n, x, h int) []content.Point {
	step := 2
	if formation == "tight" {
		step = 1
	}
	top := (h - (n-1)*step) / 2
	var out []content.Point
	for i := 0; i < n; i++ {
		out = append(out, content.Point{X: x, Y: top + i*step})
	}
	return out
}

// fight plays one battle and scores it for the side holding the trait (the player's side when there is none).
func fight(su setup, j job, side string, gap int, seed uint64, show bool) tally {
	d := su.d
	dd := *d
	w, h := 2+gap+2+4, max(8, 2*j.n+4)
	ax, bx := 3, 3+gap+1
	tiles := make([]string, h)
	for y := range tiles {
		tiles[y] = strings.Repeat("i--", w)
	}
	traits := func(s string) *[]string {
		list := []string{}
		if j.trait != "" && s == side {
			list = withRequires(d, j.trait, list)
		}
		return &list
	}
	var enemies []content.Spawn
	var roster []string
	dd.Characters = maps.Clone(d.Characters)
	allies, foes := line(j.formation, j.n, ax, h), line(j.formation, j.n, bx, h)
	kit := gear(d, j.class, j.tpl)
	for i := 0; i < j.n; i++ {
		c := d.Characters[j.tpl]
		c.Template, c.Playable, c.Lord = false, true, false
		c.Class, c.Level, c.Traits, c.Learn, c.Equipment = j.class, su.level, nil, nil, kit
		id := "zz_side" + strconv.Itoa(i+1)
		dd.Characters[id] = c
		roster = append(roster, id)
		foe := c
		foe.Playable = false
		fid := "zz_foe" + strconv.Itoa(i+1)
		dd.Characters[fid] = foe
		enemies = append(enemies, content.Spawn{Character: fid, Level: su.level, X: foes[i].X, Y: foes[i].Y, Facing: "NW", Policy: "assault", Traits: traits("enemy")})
	}
	st := content.Stage{ID: "zz-traitsim", Name: "traitsim", Width: w, Height: h, Max: j.n, Tiles: tiles, Allies: allies,
		Enemies: enemies, TurnLimit: 30, Forced: true,
		Victory: []content.Goal{{If: []string{"alldefeated:enemy"}, Text: "rout"}}}
	dd.Stages = []content.Stage{st}
	dd.Nodes = append(slices.Clone(d.Nodes), content.Node{ID: "zz-traitsim", Kind: "battle", Stage: st.ID})
	dd.Campaign.Start = "zz-traitsim"
	dd.Campaign.Roster, dd.Campaign.Deployment = roster, roster
	snap := core.New(&dd, seed).Snapshot() // officers in the party hold only what they learned
	for i := range snap.Characters {
		if strings.HasPrefix(snap.Characters[i].ID, "zz_side") {
			snap.Characters[i].Learned = *traits("ally")
		}
	}
	e, err := core.Restore(&dd, snap)
	check(err)
	_, err = e.Apply(core.Command{Kind: "start"})
	check(err)
	if fullMP {
		snap, maxMP := e.Snapshot(), map[string]int{}
		for _, u := range e.Observe().UnitViews {
			maxMP[u.ID] = u.Stats.MaxMP
		}
		for i := range snap.Units {
			snap.Units[i].MP = maxMP[snap.Units[i].ID]
		}
		e, err = core.Restore(&dd, snap)
		check(err)
	}
	shown := false
	for step := 0; step < 20000; step++ {
		o := e.Observe()
		if o.Phase != "battle" && o.Phase != "deployment" {
			break
		}
		if show && o.Phase == "battle" && !shown {
			shown = true
			for _, u := range o.UnitViews {
				fmt.Printf("%-6s %-16s %-14s Lv%-2d (%d,%d) HP%5d 공%4d 방%4d 정%4d 민%4d 사%4d MP%d/%d 특성 %v\n", u.Faction, u.ID, u.Class, u.Level, u.X, u.Y, u.Stats.MaxHP, u.Stats.Attack, u.Stats.Defense, u.Stats.Mind, u.Stats.Agility, u.Stats.Morale, u.MP, u.Stats.MaxMP, u.Traits)
			}
		}
		c, err := ai.Next(e)
		check(err)
		if c.Kind == "wait" && buffIdle {
			if b, ok := selfBuff(e, c.Actor); ok {
				c = b
			}
		}
		r, err := e.Apply(c)
		check(err)
		if show {
			if c.Kind != "end" && c.Kind != "wait" {
				fmt.Printf("  r%d %s %s %s -> %s (%d,%d)\n", e.Observe().Round, c.Kind, c.Actor, c.Skill, c.Target, c.X, c.Y)
			}
			for _, ev := range r.Events {
				if ev.Kind == "damage" || ev.Kind == "retreat" || ev.Kind == "miss" {
					fmt.Printf("     %s %s→%s %d %s\n", ev.Kind, ev.Actor, ev.Target, ev.Amount, ev.Text)
				}
			}
		}
	}
	o := e.Observe()
	hp := map[string][2]int{}
	for _, u := range o.UnitViews {
		s := hp[u.Faction]
		s[0] += max(0, u.HP)
		s[1] += u.Stats.MaxHP
		hp[u.Faction] = s
	}
	share := func(f string) float64 { return float64(hp[f][0]) / float64(max(1, hp[f][1])) }
	other := map[string]string{"ally": "enemy", "enemy": "ally"}[side]
	r := tally{FirstFights: map[bool]int{true: 1}[side == "ally"], Fights: 1, Rounds: float64(o.Round), Margin: share(side) - share(other)}
	won := o.Result == "victory"
	switch {
	case o.Result != "victory" && o.Result != "defeat":
		r.Draws = 1
	case won == (side == "ally"):
		r.Wins = 1
		if side == "ally" {
			r.FirstWins = 1
		}
	default:
		r.Losses = 1
	}
	if show {
		fmt.Println("result", o.Result, "round", o.Round, "margin", strconv.FormatFloat(r.Margin, 'f', 3, 64))
	}
	return r
}

// gear is what the soldiers of the class carry: the template's own pieces the class can wear, and a plain
// weapon of the class's kind (Bonus 1, no traits) where the template has none it can wield.
func gear(d *content.Data, class, tpl string) []string {
	cl := d.Classes[class]
	var out []string
	armed := false
	for _, id := range d.Characters[tpl].Equipment {
		if eq := d.Equipment[id]; d.Carries(cl, nil, eq) {
			out = append(out, id)
			armed = armed || eq.Slot == "weapon"
		}
	}
	if !armed {
		for _, id := range slices.Sorted(maps.Keys(d.Equipment)) {
			eq := d.Equipment[id]
			if eq.Slot == "weapon" && len(eq.Traits) == 0 && slices.Max(eq.Bonus[:]) <= 1 && d.Carries(cl, nil, eq) {
				return append(out, id)
			}
		}
	}
	return out
}

// selfBuff is a buff the unit may cast on itself now and does not hold yet, for a turn it would end without striking.
func selfBuff(e *core.Engine, actor string) (core.Command, bool) {
	var me *core.UnitView
	o := e.Observe()
	for i := range o.UnitViews {
		if o.UnitViews[i].ID == actor {
			me = &o.UnitViews[i]
		}
	}
	if me == nil {
		return core.Command{}, false
	}
	for _, c := range e.LegalActions(actor).Commands {
		if c.Kind != "skill" {
			continue
		}
		sk := e.Data().Skills[c.Skill]
		if sk.Kind != "buff" || sk.Mode != "self" {
			continue
		}
		for _, st := range strings.Split(sk.Effect, "+") {
			if me.Status[st].Turns == 0 {
				return c, true
			}
		}
	}
	return core.Command{}, false
}

// offField tells a trait a fight between soldiers cannot show: deputies, items, villages, weapons to wear.
func offField(t content.Trait) bool {
	for k := range t.Effects {
		switch {
		case k == content.FxDeputyAppoint, k == content.FxDeputyCharm, k == content.FxItemShare, k == content.FxItemDouble,
			k == content.FxVillageForage, strings.HasPrefix(k, content.FxWield):
			return true
		}
	}
	return false
}

// withRequires adds the trait and, before it, every trait it needs.
func withRequires(d *content.Data, id string, list []string) []string {
	if slices.Contains(list, id) {
		return list
	}
	for _, r := range d.Traits[id].Requires {
		list = withRequires(d, r, list)
	}
	return append(list, id)
}

func splitList(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func splitInts(s string) []int {
	var out []int
	for _, x := range splitList(s) {
		n, err := strconv.Atoi(x)
		check(err)
		out = append(out, n)
	}
	return out
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
