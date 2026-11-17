// Temporary tool: learns battle actions from real campaign traces and PPO episodes, then evaluates fixed preparations.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"srpg/internal/autoplay"
	"srpg/internal/content"
)

var (
	traces      = flag.String("traces", "reports/autoplay/watch/trace/search-12.jsonl,reports/autoplay/watch/trace-repeat/search-12.jsonl,reports/autoplay/watch/trace-repeat/search-13.jsonl", "real campaign JSONL files, comma separated")
	stages      = flag.String("stages", "B01", "stages to learn/evaluate, comma separated")
	model       = flag.String("model", "reports/autoplay/rl/battle.json", "model output/input")
	initial     = flag.String("initial", "", "resume training from this model")
	objective   = flag.String("objective", "", "training objective: empty preserves current, growth rewards clearing enemies and saving items")
	rate        = flag.Float64("rate", 2e-4, "PPO learning rate")
	entropy     = flag.Float64("entropy", .01, "PPO entropy coefficient")
	devEvery    = flag.Int("dev-every", 0, "validate and keep best every n episodes; 0 disables")
	devGames    = flag.Int("dev-games", 32, "validation battles per checkpoint")
	devSeed     = flag.Uint64("dev-seed", 60001, "first validation dice; keep separate from final evaluation")
	selfImitate = flag.Int("self-imitate", 0, "extra imitation epochs on own won, no-item, ordinary-enemies<=1 training episodes")
	episodes    = flag.Int("episodes", 256, "reinforcement-learning episodes")
	warm        = flag.Int("warm", 20, "behavior-cloning epochs")
	hidden      = flag.Int("hidden", 32, "units in each of two hidden layers")
	batch       = flag.Int("batch", 32, "policy decisions per update batch")
	epochs      = flag.Int("epochs", 3, "PPO epochs per collection")
	par         = flag.Int("par", 4, "parallel episode workers")
	collect     = flag.Int("collect", 8, "episodes per PPO update")
	seed        = flag.Uint64("seed", 7000, "training RNG seed")
	evalSeeds   = flag.Uint64("eval-seed", 20001, "first evaluation dice")
	evalN       = flag.Int("eval", 8, "evaluation games per stage")
	baseline    = flag.Bool("baseline", true, "also evaluate the existing search policy")
	out         = flag.String("out", "reports/autoplay/rl/eval.json", "evaluation report")
)

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		check(fmt.Errorf("usage: _battlelearn train|eval [flags]"))
	}
	command := os.Args[1]
	check(flag.CommandLine.Parse(os.Args[2:]))
	if *par < 1 || *batch < 1 || *collect < 1 || *epochs < 1 || *hidden < 1 || *hidden > 256 || *evalN < 1 {
		check(fmt.Errorf("par, batch, collect, epochs, hidden and eval must be positive; hidden <= 256"))
	}
	if *rate <= 0 || *entropy < 0 || *devEvery < 0 || *devGames < 1 || *episodes < 0 || *warm < 0 || *selfImitate < 0 {
		check(fmt.Errorf("invalid training settings"))
	}
	d, err := content.Load("assets/data")
	check(err)
	startTime := time.Now()
	starts, rows, err := autoplay.ReadBattleExamples(d, strings.Split(*traces, ","), strings.Split(*stages, ","))
	check(err)
	if len(starts) == 0 || len(rows) == 0 {
		check(fmt.Errorf("no complete battle examples"))
	}
	fmt.Printf("real campaign examples: %d starts, %d choices (%v)\n", len(starts), len(rows), time.Since(startTime).Round(time.Second))
	var m *autoplay.BattleModel
	if command == "train" {
		m = autoplay.NewBattleModel(*hidden, *seed)
		if *initial != "" {
			m, err = autoplay.LoadBattleModel(*initial)
			check(err)
		}
		if len(m.StageModels) != 0 {
			check(fmt.Errorf("train an individual checkpoint, not a model bundle"))
		}
		if *objective != "" {
			m.Objective = *objective
		}
		m.ContentHash, m.Training = d.Hash, strings.Split(*traces, ",")
		t := autoplay.NewBattleTrainer(m, *seed)
		t.Entropy = *entropy
		for epoch := range *warm {
			loss := t.Imitate(rows, 1, *batch, 1e-3)
			fmt.Printf("warm %d/%d loss %.4f (%v)\n", epoch+1, *warm, loss, time.Since(startTime).Round(time.Second))
		}
		check(os.MkdirAll(filepath.Dir(*model), 0o755))
		check(m.Save(*model))
		var best *assessment
		validate := func(done int) {
			score := assess(d, starts, m)
			fmt.Printf("dev %d wins %d/%d goal %d/%d xp %.4f items %.3f ordinary-left %.3f\n", done, score.Wins, score.Games, score.Qualified, score.Games, score.XP/float64(score.Games), float64(score.Items)/float64(score.Games), float64(score.Others)/float64(score.Games))
			check(m.Save(fmt.Sprintf("%s.%06d.json", *model, m.Episodes)))
			if best == nil || score.better(*best) {
				copy := score
				best = &copy
				check(m.Save(*model + ".best.json"))
				b, err := json.MarshalIndent(score, "", " ")
				check(err)
				check(os.WriteFile(*model+".best.eval.json", b, 0o644))
				fmt.Println("new best checkpoint")
			}
		}
		if *devEvery > 0 {
			validate(0)
		}
		rng := rand.New(rand.NewPCG(*seed, 1))
		for done := 0; done < *episodes; {
			n := min(*collect, *episodes-done)
			collected := make([][]autoplay.BattleSample, n)
			results := make([]autoplay.BattleOutcome, n)
			chosen, dice := make([]int, n), make([]uint64, n)
			for i := range n {
				chosen[i], dice[i] = rng.IntN(len(starts)), rng.Uint64()
			}
			parallel(n, func(i int) {
				collected[i], results[i] = autoplay.PlayBattleModel(d, starts[chosen[i]], m, dice[i], true)
			})
			var samples []autoplay.BattleSample
			var successful []autoplay.BattleSample
			wins, xp, rounds, items, others, qualified := 0, 0.0, 0, 0, 0, 0
			for i, r := range results {
				samples = append(samples, collected[i]...)
				if r.Won {
					wins++
				}
				xp += r.XP
				rounds += r.Round
				items += r.ItemsUsed
				others += r.OthersLeft
				if r.Won && r.ItemsUsed == 0 && r.OthersLeft <= 1 {
					qualified++
					successful = append(successful, collected[i]...)
				}
				if r.Error != "" {
					fmt.Printf("episode %d: %s\n", done+i, r.Error)
				}
			}
			loss := t.Improve(samples, *epochs, *batch, *rate)
			if *selfImitate > 0 {
				t.Imitate(successful, *selfImitate, *batch, *rate)
			}
			done += n
			m.Episodes += n
			fmt.Printf("ppo %d/%d wins %d/%d goal %d xp %.3f items %.2f ordinary %.2f round %.1f decisions %d chosen_nll %.4f (%v)\n", done, *episodes, wins, n, qualified, xp/float64(n), float64(items)/float64(n), float64(others)/float64(n), float64(rounds)/float64(n), len(samples), loss, time.Since(startTime).Round(time.Second))
			check(m.Save(*model))
			if *devEvery > 0 && (done%*devEvery < n || done == *episodes) {
				validate(done)
			}
		}
	} else if command == "eval" {
		m, err = autoplay.LoadBattleModel(*model)
		check(err)
	} else {
		check(fmt.Errorf("unknown command %q", command))
	}
	evaluate(d, starts, m)
}

type assessment struct {
	Games, Wins, Qualified, Items, Others int
	XP                                    float64
	FirstDice                             uint64
}

func (a assessment) better(b assessment) bool {
	if a.Wins != b.Wins {
		return a.Wins > b.Wins
	}
	if a.Qualified != b.Qualified {
		return a.Qualified > b.Qualified
	}
	return a.XP > b.XP
}

func assess(d *content.Data, starts []autoplay.BattleStart, m *autoplay.BattleModel) assessment {
	results := make([]autoplay.BattleOutcome, *devGames)
	parallel(len(results), func(i int) {
		_, results[i] = autoplay.PlayBattleModel(d, starts[i%len(starts)], m, *devSeed+uint64(i), false)
	})
	a := assessment{Games: len(results), FirstDice: *devSeed}
	for _, r := range results {
		if r.Won && r.Error == "" {
			a.Wins++
			a.XP += r.XP
			if r.ItemsUsed == 0 && r.OthersLeft <= 1 {
				a.Qualified++
			}
		}
		a.Items += r.ItemsUsed
		a.Others += r.OthersLeft
	}
	return a
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

func evaluate(d *content.Data, starts []autoplay.BattleStart, m *autoplay.BattleModel) {
	var selected []autoplay.BattleStart
	seen := map[string]bool{}
	for _, start := range starts {
		if !seen[start.Stage] {
			selected = append(selected, start)
			seen[start.Stage] = true
		}
	}
	type report struct {
		ContentHash string
		Model       string
		Kind        string
		Neural      []autoplay.BattleOutcome
		Search      []autoplay.BattleOutcome
	}
	r := report{ContentHash: d.Hash, Model: *model, Kind: "fixed actual campaign preparations; dice varied; not a campaign clear", Neural: make([]autoplay.BattleOutcome, len(selected)**evalN), Search: make([]autoplay.BattleOutcome, len(selected)**evalN)}
	if !*baseline {
		r.Search = nil
	}
	parallel(len(r.Neural), func(i int) {
		start, dice := selected[i / *evalN], *evalSeeds+uint64(i%*evalN)
		_, r.Neural[i] = autoplay.PlayBattleModel(d, start, m, dice, false)
		line := fmt.Sprintf("eval %s dice %d neural=%v xp %.3f r%d", start.Stage, dice, r.Neural[i].Won, r.Neural[i].XP, r.Neural[i].Round)
		if *baseline {
			_, r.Search[i] = autoplay.PlayBattleModel(d, start, nil, dice, false)
			line += fmt.Sprintf("; search=%v xp %.3f r%d", r.Search[i].Won, r.Search[i].XP, r.Search[i].Round)
		}
		fmt.Println(line)
	})
	b, err := json.MarshalIndent(r, "", " ")
	check(err)
	check(os.MkdirAll(filepath.Dir(*out), 0o755))
	check(os.WriteFile(*out, b, 0o644))
}
