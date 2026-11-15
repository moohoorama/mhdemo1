// Temporary tool: plays the whole campaign with the autoplay AIs and scores them (internal/autoplay).
//
//	go run ./cmd/_autoplay run -player rule -seeds 1,2,3
//	go run ./cmd/_autoplay evo -gens 8 -pop 16 -out reports/autoplay/genome.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"srpg/internal/autoplay"
	"srpg/internal/content"
)

var (
	dir        = flag.String("content", "assets/data", "content directory")
	difficulty = flag.String("difficulty", "hard", "difficulty id")
	seedsFlag  = flag.String("seeds", "1", "campaign seeds, comma-separated")
	playerFlag = flag.String("player", "rule", "base | rule | evo | search | value | policy")
	genomeFlag = flag.String("genome", "reports/autoplay/genome.json", "the tuned genome (evo and the players built on it)")
	stages     = flag.Int("stages", 0, "battles to fight at most; 0 all")
	out        = flag.String("out", "", "file for the reports (JSON)")
	par        = flag.Int("par", 4, "campaigns played at once")
	kFlag      = flag.Int("k", 4, "search: candidates per decision")
	gens       = flag.Int("gens", 8, "evo: generations")
	models     = flag.String("models", "reports/autoplay", "where the learned models are kept")
	epochs     = flag.Int("epochs", 40, "train: passes over the rows")
	pop        = flag.Int("pop", 16, "evo: genomes per generation")
	elite      = flag.Int("elite", 4, "evo: genomes the next generation is drawn around")
	rFlag      = flag.Int("r", 2, "search: dice per candidate")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: _autoplay run|evo [flags]")
		os.Exit(2)
	}
	cmd := os.Args[1]
	flag.CommandLine.Parse(os.Args[2:])
	d, err := content.Load(*dir)
	check(err)
	switch cmd {
	case "run":
		run(d)
	case "evo":
		evo(d)
	case "summary":
		summary(flag.Args())
	case "train-value":
		trainValue(d)
	case "train-policy":
		trainPolicy(d)
	default:
		check(fmt.Errorf("unknown command %s", cmd))
	}
}

func seeds() []uint64 {
	var out []uint64
	for _, s := range strings.Split(*seedsFlag, ",") {
		n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
		check(err)
		out = append(out, n)
	}
	return out
}

func run(d *content.Data) {
	ss := seeds()
	reps := make([]autoplay.Report, len(ss))
	sem := make(chan bool, *par)
	var wg sync.WaitGroup
	t := time.Now()
	for i, s := range ss {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- true
			defer func() { <-sem }()
			p := player(d, *playerFlag)
			r, err := autoplay.Run(d, p, autoplay.Options{Difficulty: *difficulty, Seed: s, Stages: *stages, Log: os.Stdout})
			check(err)
			reps[i] = r
		}()
	}
	wg.Wait()
	for _, r := range reps {
		fmt.Println(autoplay.Summary(r))
	}
	fmt.Printf("%s: %v\n", *playerFlag, time.Since(t).Round(time.Second))
	if *out != "" {
		b, _ := json.MarshalIndent(reps, "", " ")
		check(os.WriteFile(*out, b, 0o644))
	}
}

func player(d *content.Data, name string) autoplay.Player {
	switch name {
	case "base":
		return autoplay.Baseline(d)
	case "rule":
		return &autoplay.RulePlayer{Label: "rule", D: d, G: autoplay.DefaultGenome()}
	case "evo":
		return &autoplay.RulePlayer{Label: "evo", D: d, G: genome()}
	case "search":
		return searcher(d)
	case "value":
		net, err := autoplay.LoadMLP(filepath.Join(*models, "value.json"))
		check(err)
		return &autoplay.ValuePlayer{Label: "value", D: d, G: genome(), Net: net, K: 6, P: 6}
	case "policy":
		act, err := autoplay.LoadMLP(filepath.Join(*models, "policy.json"))
		check(err)
		prep, err := autoplay.LoadMLP(filepath.Join(*models, "prep.json"))
		check(err)
		return &autoplay.PolicyPlayer{Label: "policy", D: d, G: genome(), ActNet: act, PrepNet: prep, K: *kFlag, P: 6}
	}
	check(fmt.Errorf("unknown player %s", name))
	return nil
}

func searcher(d *content.Data) *autoplay.Searcher {
	return &autoplay.Searcher{Label: "search", D: d, G: genome(), K: *kFlag, R: *rFlag, P: 6, B: 3, Par: 8}
}

func trainValue(d *content.Data) {
	t := time.Now()
	states, starts := autoplay.CollectValue(d, autoplay.BattleNodes(d), seeds(), *difficulty, genome(), .25, 8)
	rows := append(states, starts...)
	fmt.Printf("value rows %d (%d battle starts) in %v\n", len(rows), len(starts), time.Since(t).Round(time.Second))
	net := autoplay.Train(rows, 32, *epochs, 1)
	fmt.Printf("value net: train loss %.3f, validation loss %.3f\n", net.TrainLoss, net.ValLoss)
	check(net.Save(filepath.Join(*models, "value.json")))
}

func trainPolicy(d *content.Data) {
	t := time.Now()
	acts, preps := autoplay.CollectPolicy(d, autoplay.BattleNodes(d), seeds(), *difficulty, *searcher(d), 2)
	fmt.Printf("policy rows %d, preparation rows %d in %v\n", len(acts), len(preps), time.Since(t).Round(time.Second))
	act := autoplay.Train(acts, 48, *epochs, 2)
	fmt.Printf("policy net: train loss %.3f, validation loss %.3f\n", act.TrainLoss, act.ValLoss)
	check(act.Save(filepath.Join(*models, "policy.json")))
	prep := autoplay.Train(preps, 24, *epochs*3, 3)
	fmt.Printf("prep net: train loss %.3f, validation loss %.3f\n", prep.TrainLoss, prep.ValLoss)
	check(prep.Save(filepath.Join(*models, "prep.json")))
}

// summary prints a markdown table of run files.
func summary(files []string) {
	fmt.Println("| AI | 끝까지 깬 판 | 이긴 전투(판당) | 시도 승률 | 첫 시도 승률 | 부목표 달성 | 레벨 점수 |")
	fmt.Println("|---|---|---|---|---|---|---|")
	for _, f := range files {
		b, err := os.ReadFile(f)
		check(err)
		var reps []autoplay.Report
		check(json.Unmarshal(b, &reps))
		fmt.Println(autoplay.Total(reps).Row())
	}
}

// tuned is what evo saves.
type tuned struct {
	Genome      autoplay.Genome
	Fitness     float64
	Generations []autoplay.Generation
}

func genome() autoplay.Genome {
	b, err := os.ReadFile(*genomeFlag)
	check(err)
	var t tuned
	check(json.Unmarshal(b, &t))
	return t.Genome
}

func evo(d *content.Data) {
	t := time.Now()
	g, f, hist := autoplay.Evolve(d, autoplay.EvoOptions{Pop: *pop, Elite: *elite, Gens: *gens, Seeds: seeds(), Difficulty: *difficulty,
		Stages: *stages, Par: 8, Start: autoplay.DefaultGenome(), Log: os.Stdout})
	fmt.Printf("best %.1f %v (%v)\n", f, g, time.Since(t).Round(time.Second))
	b, _ := json.MarshalIndent(tuned{g, f, hist}, "", " ")
	check(os.MkdirAll(filepath.Dir(*genomeFlag), 0o755))
	check(os.WriteFile(*genomeFlag, b, 0o644))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
