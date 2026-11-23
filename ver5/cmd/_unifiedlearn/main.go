// Temporary tool: one unified numeric model (no per-stage experts) trained for a fixed wall-clock budget on
// first-loss campaigns at the hard difficulty, then evaluated on held-out seeds with per-battle records.
//
//	go run ./cmd/_unifiedlearn train -budget 30m -model out/best.json
//	go run ./cmd/_unifiedlearn eval -model out/best.json -out out/eval.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"srpg/internal/autoplay"
	"srpg/internal/content"
	"srpg/internal/core"
)

// difficulty is the game's "hard" id, shown to players as 매우 어려움.
const difficulty = "hard"

var (
	contentDir  = flag.String("content", "assets/data", "game data directory (scenario is read from its sibling folder)")
	budget      = flag.Duration("budget", 30*time.Minute, "train: wall-clock budget, counted from after setup")
	model       = flag.String("model", "", "train: output directory base path (best.json, latest.json); eval: checkpoint to read")
	out         = flag.String("out", "", "train: log directory; eval: result JSON")
	trainSeed   = flag.Uint64("seed", 220001, "first training new-game seed (never a dev or eval seed)")
	devSeed     = flag.Uint64("dev-seed", 230001, "first development new-game seed")
	devGames    = flag.Int("dev-games", 4, "development new games per checkpoint")
	evalSeed    = flag.Uint64("eval-seed", 240001, "first held-out evaluation seed")
	evalGames   = flag.Int("eval", 8, "held-out evaluation new games")
	par         = flag.Int("par", 4, "campaigns played at once")
	collect     = flag.Int("collect", 8, "training campaigns per update")
	hidden      = flag.Int("hidden", 32, "units in each hidden layer")
	initSeed    = flag.Uint64("init-seed", 7, "weight initialisation seed")
	warm        = flag.Int("warm", 8, "rule-preparation demonstration campaigns")
	warmEpochs  = flag.Int("warm-epochs", 20, "imitation passes over the preparation demonstrations")
	temperature = flag.Float64("battle-temperature", .25, "training battle sampling temperature")
	prep        = flag.String("prep", "learned", "eval: learned keeps the preparation network; rule uses the rule preparation, Learn and explore only")
	policy      = flag.String("policy", "learned", "eval: learned (the unified network) or laya (the Laya bridge chooses every battle action)")
	pyBin       = flag.String("python", "", "laya: python interpreter with torch and transformers>=4.48")
	layaScript  = flag.String("laya-script", "", "laya: path to tools/unified30/laya_bridge.py")
	layaDir     = flag.String("laya-dir", "", "laya: directory of the downloaded Laya checkpoint")
	layaHead    = flag.String("laya-head", "", "laya: trained decision-head weights (empty keeps the released head)")
	exportSeed  = flag.Uint64("export-seed", 220001, "export: first training campaign seed for the teacher decisions")
	exportGames = flag.Int("export-games", 0, "export: teacher campaigns to write as decision rows")
	bcCampaigns = flag.Int("bc", 4, "warm-start: campaigns whose battle decisions the rule search AI makes (imitated)")
	bcEpochs    = flag.Int("bc-epochs", 2, "warm-start: imitation passes per teacher campaign")
	bcSeed      = flag.Uint64("bc-seed", 225001, "first teacher warm-start seed (between training and dev seeds)")
)

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parallel(n int, job func(int)) {
	var wg sync.WaitGroup
	jobs := make(chan int)
	for range min(*par, n) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				job(i)
			}
		}()
	}
	for i := range n {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}

func save(path string, v any) {
	b, err := json.MarshalIndent(v, "", " ")
	check(err)
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	check(os.WriteFile(path, b, 0o644))
}

func main() {
	if len(os.Args) < 2 {
		check(fmt.Errorf("usage: _unifiedlearn train|eval [flags]"))
	}
	command := os.Args[1]
	check(flag.CommandLine.Parse(os.Args[2:]))
	if *par < 1 || *collect < 1 || *devGames < 1 || *evalGames < 1 || *warmEpochs < 1 || *temperature <= 0 || *hidden < 1 || *hidden > 256 {
		check(fmt.Errorf("invalid settings"))
	}
	d, err := content.Load(*contentDir)
	check(err)
	switch command {
	case "train":
		train(d)
	case "eval":
		evaluate(d)
	case "export":
		exportDecisions(d)
	default:
		check(fmt.Errorf("unknown command %q", command))
	}
}

// newUnified is one battle network for every stage, with the preparation network the campaign learner uses.
func newUnified(d *content.Data) *autoplay.BattleModel {
	m := autoplay.NewBattleModel(*hidden, *initSeed)
	m.ContentHash = d.Hash
	m.EnableCampaignLearning(*hidden, *initSeed)
	check(m.Validate())
	return m
}

type event struct {
	Event       string  `json:"event"`
	Episodes    int     `json:"episodes"`
	ElapsedSec  float64 `json:"elapsed_sec"`
	TrainMean   float64 `json:"train_mean_wins,omitempty"`
	DevMean     float64 `json:"dev_mean_wins"`
	DevMin      int     `json:"dev_min_wins"`
	DevMax      int     `json:"dev_max_wins"`
	DevComplete int     `json:"dev_completed"`
	DevAttempts int     `json:"dev_attempts"`
	BestUpdated bool    `json:"best_updated"`
}

func train(d *content.Data) {
	if *model == "" || *out == "" {
		check(fmt.Errorf("train needs -model and -out"))
	}
	m := newUnified(d)
	t := autoplay.NewCampaignTrainer(m, *trainSeed)
	logf, err := os.Create(filepath.Join(*out, "events.jsonl"))
	check(err)
	defer logf.Close()
	enc := json.NewEncoder(logf)
	started := time.Now()
	fmt.Printf("train start content=%s budget=%v\n", d.Hash, *budget)

	var demos []autoplay.CampaignEpisode
	for i := range *warm {
		e := autoplay.PlayCampaignEpisode(d, m, *trainSeed+uint64(i), false, true, 1)
		check(errOf(e))
		demos = append(demos, e)
	}
	if len(demos) > 0 {
		t.Update(demos, true, *warmEpochs, 64, 1e-3)
	}
	for i := range *bcCampaigns {
		ep := teacherEpisode(d, m, *bcSeed+uint64(i))
		check(errOf(ep))
		rows := ep.Samples[""]
		t.Models[""].Imitate(rows, *bcEpochs, 64, 1e-3)
		fmt.Printf("warm-start teacher %d/%d %s rows %d\n", i+1, *bcCampaigns, autoplay.Summary(ep.Report), len(rows))
	}
	check(m.Save(*model + ".latest.json"))

	score := devScore(d, m)
	best, initial := score, score
	check(m.Save(*model + ".best.json"))
	check(enc.Encode(toEvent("dev-initial", 0, started, score, true)))
	fmt.Printf("initial dev mean %.2f min %d max %d completed %d/%d\n", score.Mean(), score.Min, score.Max, score.Completed, score.Attempts)

	var progress autoplay.CampaignAssessment
	done, longest := 0, time.Duration(0)
	for {
		if time.Since(started)+longest > *budget {
			break
		}
		cycle := time.Now()
		n := *collect
		results := make([]autoplay.CampaignEpisode, n)
		base := done
		parallel(n, func(i int) {
			results[i] = autoplay.PlayCampaignEpisode(d, m, *trainSeed+uint64(*warm+base+i), true, false, *temperature)
		})
		for _, e := range results {
			check(errOf(e))
			progress.Add(e.Report)
		}
		t.Update(results, false, 2, 64, 1e-4)
		done += n
		m.CampaignEpisodes += n
		m.Preparation.CampaignEpisodes += n
		check(m.Save(*model + ".latest.json"))

		score = devScore(d, m)
		updated := score.Better(best)
		if updated {
			best = score
			check(m.Save(*model + ".best.json"))
		}
		ev := toEvent("cycle", done, started, score, updated)
		ev.TrainMean = progress.Mean()
		check(enc.Encode(ev))
		fmt.Printf("train %d episodes train-mean %.2f dev mean %.2f min %d completed %d/%d best=%v elapsed %v\n",
			done, progress.Mean(), score.Mean(), score.Min, score.Completed, score.Attempts, updated, time.Since(started).Round(time.Second))
		longest = max(longest, time.Since(cycle))
	}
	fmt.Printf("train end episodes %d elapsed %v (budget %v)\n", done, time.Since(started).Round(time.Second), *budget)
	save(filepath.Join(*out, "train-summary.json"), map[string]any{
		"content_hash": d.Hash, "budget": budget.String(), "train_episodes": done, "warm_campaigns": *warm,
		"train_seed_first": *trainSeed, "dev_seed_first": *devSeed, "dev_games": *devGames,
		"eval_seed_first_unused": *evalSeed, "collect": *collect, "par": *par, "hidden": *hidden,
		"battle_temperature": *temperature, "initial_dev_mean": initial.Mean(), "best_dev_mean": best.Mean(),
		"best_dev_min": best.Min, "best_dev_completed": best.Completed, "train_elapsed_sec": time.Since(started).Seconds(),
		"train_campaign_mean_wins": progress.Mean(),
	})
}

// teacherEpisode plays a campaign where the rule search AI makes every battle decision; its choices are recorded
// as imitation rows for the unified network.
func teacherEpisode(d *content.Data, m *autoplay.BattleModel, seed uint64) autoplay.CampaignEpisode {
	out := autoplay.CampaignEpisode{Samples: map[string][]autoplay.BattleSample{}}
	p := &autoplay.NeuralPlayer{Searcher: autoplay.CampaignPlayer(d), Model: m, BattleTeacher: true, WarmPreparation: true, BattleTemperature: 1}
	p.Record = func(key string, row autoplay.BattleSample) { out.Samples[key] = append(out.Samples[key], row) }
	r, err := autoplay.Run(d, p, autoplay.Options{Difficulty: difficulty, Seed: seed, Tries: 1})
	out.Report = r
	if err != nil {
		out.Error = err.Error()
		return out
	}
	wins, _ := r.FirstLossProgress()
	for key, rows := range out.Samples {
		for i := range rows {
			rows[i].Return = float64(wins)
		}
		out.Samples[key] = rows
	}
	return out
}

func errOf(e autoplay.CampaignEpisode) error {
	if e.Error != "" {
		return fmt.Errorf("campaign: %s", e.Error)
	}
	return nil
}

func devScore(d *content.Data, m *autoplay.BattleModel) autoplay.CampaignAssessment {
	results := make([]autoplay.CampaignEpisode, *devGames)
	parallel(*devGames, func(i int) {
		results[i] = autoplay.PlayCampaignEpisode(d, m, *devSeed+uint64(i), false, false, 1)
	})
	var a autoplay.CampaignAssessment
	for _, e := range results {
		check(errOf(e))
		a.Add(e.Report)
	}
	return a
}

func toEvent(name string, episodes int, started time.Time, a autoplay.CampaignAssessment, updated bool) event {
	return event{Event: name, Episodes: episodes, ElapsedSec: time.Since(started).Seconds(), DevMean: a.Mean(),
		DevMin: a.Min, DevMax: a.Max, DevComplete: a.Completed, DevAttempts: a.Attempts, BestUpdated: updated}
}

// Officer XP, items and money are read from the engine before and after each battle's preparation.
type snap struct {
	Stage  string
	Money  int
	Inv    map[string]int
	Merit  core.Merit
	Levels map[string]int
}

// recorder wraps the neural player to take a snapshot before and after its preparation at each battle.
type recorder struct {
	*autoplay.NeuralPlayer
	d      *content.Data
	pre    []snap
	post   []snap
	lastE  *core.Engine
	bridge *layaBridge
	err    error
}

func (r *recorder) Act(e *core.Engine, actor string) core.Command {
	if r.bridge == nil {
		return r.NeuralPlayer.Act(e, actor)
	}
	cmds := autoplay.BattleCommands(r.d, noPrep(), e)
	if len(cmds) == 0 {
		return core.Command{Kind: "end"}
	}
	i, err := r.bridge.choose(stateText(r.d, e), optionTexts(e, cmds))
	if err != nil {
		r.err = err
		return cmds[0]
	}
	return cmds[i]
}

func (r *recorder) Prepare(e *core.Engine, try int) {
	r.lastE = e
	r.pre = append(r.pre, r.take(e))
	r.NeuralPlayer.Prepare(e, try)
	r.post = append(r.post, r.take(e))
}

func (r *recorder) take(e *core.Engine) snap {
	o := e.Observe()
	stage := ""
	if o.Stage >= 0 && o.Stage < len(r.d.Stages) {
		stage = r.d.Stages[o.Stage].ID
	}
	s := snap{Stage: stage, Money: o.Money, Inv: maps.Clone(o.Inventory), Levels: autoplay.Levels(r.d, e)}
	if o.Merit != nil {
		s.Merit = core.Merit{XP: maps.Clone(o.Merit.XP), Spoils: maps.Clone(o.Merit.Spoils)}
	}
	return s
}

type battleRecord struct {
	Stage       string         `json:"stage"`
	Won         bool           `json:"won"`
	Tries       int            `json:"tries"`
	Turns       int            `json:"turns"`
	SubsDone    int            `json:"subs_done"`
	SubsTotal   int            `json:"subs_total"`
	LevelsAfter map[string]int `json:"levels_after"`
	XPGained    map[string]int `json:"xp_gained"`
	Spoils      map[string]int `json:"spoils"`
	MoneyPrep   int            `json:"money_change_preparation"`
	MoneyBattle int            `json:"money_change_battle_and_after"`
	InvPrep     map[string]int `json:"inventory_change_preparation"`
	InvBattle   map[string]int `json:"inventory_change_battle_and_after"`
}

type seedRecord struct {
	Seed        uint64          `json:"seed"`
	Wins        int             `json:"first_loss_progress_wins"`
	FirstDefeat string          `json:"first_defeat"`
	Cleared     bool            `json:"cleared"`
	LevelScore  float64         `json:"level_score"`
	Battles     []battleRecord  `json:"battles"`
	FinalLevels map[string]int  `json:"final_levels"`
	StartMoney  int             `json:"start_money"`
	Error       string          `json:"error,omitempty"`
	Report      autoplay.Report `json:"report"`
}

func delta(a, b map[string]int) map[string]int {
	out := map[string]int{}
	for k := range b {
		if v := b[k] - a[k]; v != 0 {
			out[k] = v
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok && a[k] != 0 {
			out[k] = -a[k]
		}
	}
	return out
}

func playEval(d *content.Data, m *autoplay.BattleModel, seed uint64, bridge *layaBridge) seedRecord {
	if *prep == "rule" || bridge != nil {
		rule := *m
		rule.Preparation = nil
		m = &rule
	}
	np := &autoplay.NeuralPlayer{Searcher: autoplay.CampaignPlayer(d), Model: m, BattleTemperature: 1}
	r := &recorder{NeuralPlayer: np, d: d, bridge: bridge}
	rep, err := autoplay.Run(d, r, autoplay.Options{Difficulty: difficulty, Seed: seed, Tries: 1})
	sr := seedRecord{Seed: seed, Report: rep}
	if err != nil {
		sr.Error = err.Error()
	}
	if r.err != nil {
		sr.Error = "laya: " + r.err.Error()
	}
	if r.lastE == nil {
		return sr
	}
	end := r.take(r.lastE)
	wins, lost := rep.FirstLossProgress()
	sr.Wins, sr.FirstDefeat, sr.Cleared, sr.LevelScore = wins, lost, rep.Cleared, rep.LevelScore
	sr.FinalLevels = end.Levels
	if len(r.pre) > 0 {
		sr.StartMoney = r.pre[0].Money
	}
	for k, st := range rep.Stages {
		b := battleRecord{Stage: st.Stage, Won: st.Won, Tries: st.Tries, Turns: st.Rounds, SubsDone: st.Subs,
			SubsTotal: st.SubsTotal, LevelsAfter: st.Levels}
		if k < len(r.post) {
			after := end
			if k+1 < len(r.pre) {
				after = r.pre[k+1]
			}
			b.XPGained = after.Merit.XP
			b.Spoils = after.Merit.Spoils
			b.MoneyPrep = r.post[k].Money - r.pre[k].Money
			b.InvPrep = delta(r.pre[k].Inv, r.post[k].Inv)
			b.MoneyBattle = after.Money - r.post[k].Money
			b.InvBattle = delta(r.post[k].Inv, after.Inv)
		}
		sr.Battles = append(sr.Battles, b)
	}
	return sr
}

type evalResult struct {
	ContentHash string         `json:"content_hash"`
	Model       string         `json:"model"`
	Difficulty  string         `json:"difficulty"`
	Prep        string         `json:"preparation"`
	Policy      string         `json:"policy"`
	Stats       map[string]any `json:"laya_stats,omitempty"`
	Seeds       []uint64       `json:"eval_seeds"`
	Runs        []seedRecord   `json:"runs"`
	Summary     summary        `json:"summary"`
	Note        string         `json:"note"`
	WallSec     float64        `json:"eval_wall_sec"`
}

type summary struct {
	Seeds           int     `json:"seeds"`
	Mean            float64 `json:"mean_wins"`
	Median          float64 `json:"median_wins"`
	Min             int     `json:"min_wins"`
	Max             int     `json:"max_wins"`
	Cleared         int     `json:"cleared"`
	BestSeed        uint64  `json:"best_seed"`
	BestWins        int     `json:"best_wins"`
	BestFirstDefeat string  `json:"best_first_defeat"`
}

func exportDecisions(d *content.Data) {
	if *out == "" || *exportGames < 1 {
		check(fmt.Errorf("export needs -out and -export-games"))
	}
	var all []decision
	perSeed := make([][]decision, *exportGames)
	wins := make([]int, *exportGames)
	parallel(*exportGames, func(i int) {
		t := newTeacher(d, *exportSeed+uint64(i))
		r, err := autoplay.Run(d, t, autoplay.Options{Difficulty: difficulty, Seed: *exportSeed + uint64(i), Tries: 1})
		check(err)
		perSeed[i] = t.rows
		wins[i], _ = r.FirstLossProgress()
	})
	f, err := os.Create(*out)
	check(err)
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, rows := range perSeed {
		for _, r := range rows {
			check(enc.Encode(r))
			all = append(all, r)
		}
	}
	fmt.Printf("export rows %d from %d teacher campaigns, wins %v\n", len(all), *exportGames, wins)
}

func evaluate(d *content.Data) {
	if *model == "" || *out == "" {
		check(fmt.Errorf("eval needs -model and -out"))
	}
	m, err := autoplay.LoadBattleModel(*model)
	check(err)
	if m.ContentHash != d.Hash {
		check(fmt.Errorf("model content %s differs from data %s", m.ContentHash, d.Hash))
	}
	if *prep != "learned" && *prep != "rule" {
		check(fmt.Errorf("-prep must be learned or rule"))
	}
	if len(m.StageModels) > 0 {
		check(fmt.Errorf("model has per-stage experts; not a unified model"))
	}
	started := time.Now()
	runs := make([]seedRecord, *evalGames)
	var stats map[string]any
	switch *policy {
	case "learned":
		parallel(*evalGames, func(i int) { runs[i] = playEval(d, m, *evalSeed+uint64(i), nil) })
	case "laya":
		if *pyBin == "" || *layaScript == "" || *layaDir == "" {
			check(fmt.Errorf("laya eval needs -python, -laya-script and -laya-dir"))
		}
		bridge, err := startLaya(*pyBin, *layaScript, *layaDir, *layaHead)
		check(err)
		for i := range runs {
			runs[i] = playEval(d, m, *evalSeed+uint64(i), bridge)
		}
		bridge.close()
		stats = map[string]any{"decisions": bridge.requests, "model_calls": bridge.calls, "model_seconds": bridge.seconds}
	default:
		check(fmt.Errorf("-policy must be learned or laya"))
	}
	res := evalResult{ContentHash: d.Hash, Model: *model, Difficulty: difficulty, Prep: *prep, Policy: *policy, Runs: runs, Stats: stats,
		Note: "eval seeds are unused in training and dev; greedy battle and preparation; tries=1, no retries"}
	res.WallSec = time.Since(started).Seconds()
	wins := []int{}
	for _, r := range runs {
		res.Seeds = append(res.Seeds, r.Seed)
		wins = append(wins, r.Wins)
		if r.Cleared {
			res.Summary.Cleared++
		}
		if r.Error != "" {
			check(fmt.Errorf("seed %d: %s", r.Seed, r.Error))
		}
	}
	sorted := append([]int(nil), wins...)
	sort.Ints(sorted)
	s := &res.Summary
	s.Seeds = len(runs)
	sum := 0
	for _, w := range wins {
		sum += w
	}
	s.Mean = float64(sum) / float64(len(wins))
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		s.Median = float64(sorted[mid])
	} else {
		s.Median = float64(sorted[mid-1]+sorted[mid]) / 2
	}
	s.Min, s.Max = sorted[0], sorted[len(sorted)-1]
	s.BestWins = -1
	for _, r := range runs {
		if r.Wins > s.BestWins {
			s.BestWins, s.BestSeed, s.BestFirstDefeat = r.Wins, r.Seed, r.FirstDefeat
		}
	}
	save(*out, res)
	fmt.Printf("eval seeds %d mean %.2f median %.1f min %d max %d cleared %d best %d (seed %d, first defeat %s) wall %.0fs\n",
		s.Seeds, s.Mean, s.Median, s.Min, s.Max, s.Cleared, s.BestWins, s.BestSeed, s.BestFirstDefeat, res.WallSec)
	for _, r := range runs {
		fmt.Println(autoplay.Summary(r.Report))
	}
}
