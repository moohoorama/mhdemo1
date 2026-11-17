// Temporary tool: jointly learns preparation and battle decisions from first-loss campaign returns.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"srpg/internal/autoplay"
	"srpg/internal/content"
)

var (
	contentDir      = flag.String("content", "assets/data", "game data directory; snapshot it for comparable training/evaluation")
	transferContent = flag.Bool("transfer-content", false, "explicitly transfer previous weights to changed data, preserving source hash")
	initial         = flag.String("initial", "assets/autoplay/battle.json", "initial battle/model bundle")
	model           = flag.String("model", "reports/autoplay/campaign-learning/model.json", "checkpoint output/input")
	episodes        = flag.Int("episodes", 64, "fresh first-loss campaign training attempts")
	warm            = flag.Int("warm", 8, "rule-preparation demonstration campaigns")
	warmEpochs      = flag.Int("warm-epochs", 20, "imitation passes over valid preparation decisions")
	collect         = flag.Int("collect", 4, "campaigns per PPO update")
	par             = flag.Int("par", 2, "parallel campaigns")
	seed            = flag.Uint64("seed", 220001, "first training new-game seed")
	devSeed         = flag.Uint64("dev-seed", 230001, "first development new-game seed")
	devGames        = flag.Int("dev-games", 4, "development new games")
	evalSeed        = flag.Uint64("eval-seed", 240001, "first held-out evaluation seed")
	evalGames       = flag.Int("eval", 8, "held-out evaluation new games")
	temperature     = flag.Float64("battle-temperature", .25, "training battle sampling temperature")
	out             = flag.String("out", "reports/autoplay/campaign-learning/eval.json", "evaluation output")
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

func save(path string, value any) {
	b, err := json.MarshalIndent(value, "", " ")
	check(err)
	check(os.MkdirAll(filepath.Dir(path), 0755))
	check(os.WriteFile(path, b, 0644))
}

func evaluate(d *content.Data, m *autoplay.BattleModel, first uint64, n int) autoplay.CampaignAssessment {
	results := make([]autoplay.CampaignEpisode, n)
	parallel(n, func(i int) { results[i] = autoplay.PlayCampaignEpisode(d, m, first+uint64(i), false, false, 1) })
	var a autoplay.CampaignAssessment
	for _, e := range results {
		if e.Error != "" {
			check(fmt.Errorf("campaign: %s", e.Error))
		}
		a.Add(e.Report)
	}
	return a
}

func main() {
	if len(os.Args) < 2 {
		check(fmt.Errorf("usage: _campaignlearn train|eval [flags]"))
	}
	command := os.Args[1]
	check(flag.CommandLine.Parse(os.Args[2:]))
	if *par < 1 || *collect < 1 || *devGames < 1 || *evalGames < 1 || *episodes < 0 || *warm < 0 || *warmEpochs < 1 || *temperature <= 0 {
		check(fmt.Errorf("invalid training settings"))
	}
	d, err := content.Load(*contentDir)
	check(err)
	started := time.Now()
	var m *autoplay.BattleModel
	if command == "eval" {
		m, err = autoplay.LoadBattleModel(*model)
		check(err)
		bindContent(m, d.Hash)
	} else if command == "train" {
		m, err = autoplay.LoadBattleModel(*initial)
		check(err)
		bindContent(m, d.Hash)
		m.EnableCampaignLearning(32, *seed)
		check(os.MkdirAll(filepath.Dir(*model), 0755))
		t := autoplay.NewCampaignTrainer(m, *seed)
		var demos []autoplay.CampaignEpisode
		var warmStats autoplay.CampaignAssessment
		for i := range *warm {
			e := autoplay.PlayCampaignEpisode(d, m, *seed+uint64(i), false, true, 1)
			if e.Error != "" {
				check(fmt.Errorf("warm campaign: %s", e.Error))
			}
			demos = append(demos, e)
			warmStats.Add(e.Report)
			fmt.Printf("warm %d/%d %s elapsed %v\n", i+1, *warm, autoplay.Summary(e.Report), time.Since(started).Round(time.Second))
		}
		save(*model+".warm-reports.json", warmStats)
		if len(demos) > 0 {
			t.Update(demos, true, *warmEpochs, 64, 1e-3)
		}
		check(m.Save(*model))
		best := evaluate(d, m, *devSeed, *devGames)
		check(m.Save(*model + ".best.json"))
		save(*model+".best.eval.json", best)
		fmt.Printf("initial dev mean %.2f min %d max %d completed %d/%d\n", best.Mean(), best.Min, best.Max, best.Completed, best.Attempts)
		var progress autoplay.CampaignAssessment
		for done := 0; done < *episodes; {
			n := min(*collect, *episodes-done)
			results := make([]autoplay.CampaignEpisode, n)
			parallel(n, func(i int) {
				results[i] = autoplay.PlayCampaignEpisode(d, m, *seed+uint64(*warm+done+i), true, false, *temperature)
			})
			for _, e := range results {
				if e.Error != "" {
					check(fmt.Errorf("training campaign: %s", e.Error))
				}
				progress.Add(e.Report)
				fmt.Printf("train %d/%d %s elapsed %v\n", progress.Attempts, *episodes, autoplay.Summary(e.Report), time.Since(started).Round(time.Second))
			}
			t.Update(results, false, 2, 64, 1e-4)
			done += n
			m.CampaignEpisodes += n
			m.Preparation.CampaignEpisodes += n
			for _, child := range m.StageModels {
				child.CampaignEpisodes += n
			}
			check(m.Save(*model))
			save(*model+".progress.json", progress)
			score := evaluate(d, m, *devSeed, *devGames)
			fmt.Printf("dev after %d mean %.2f min %d max %d completed %d/%d\n", done, score.Mean(), score.Min, score.Max, score.Completed, score.Attempts)
			if score.Better(best) {
				best = score
				check(m.Save(*model + ".best.json"))
				save(*model+".best.eval.json", best)
				fmt.Println("new best first-loss checkpoint")
			}
		}
		m, err = autoplay.LoadBattleModel(*model + ".best.json")
		check(err)
	} else {
		check(fmt.Errorf("unknown command %q", command))
	}
	if m.ContentHash != d.Hash {
		check(fmt.Errorf("model content differs"))
	}
	result := evaluate(d, m, *evalSeed, *evalGames)
	save(*out, result)
	for _, r := range result.Reports {
		fmt.Println(autoplay.Summary(r))
	}
	fmt.Printf("final mean %.2f min %d max %d completed %d/%d elapsed %v\n", result.Mean(), result.Min, result.Max, result.Completed, result.Attempts, time.Since(started).Round(time.Second))
}

func bindContent(m *autoplay.BattleModel, hash string) {
	if m.ContentHash == hash {
		return
	}
	if !*transferContent {
		check(fmt.Errorf("game data changed: model=%s data=%s; use -transfer-content for an explicit transfer experiment", m.ContentHash, hash))
	}
	old := m.ContentHash
	var bind func(*autoplay.BattleModel)
	bind = func(n *autoplay.BattleModel) {
		if n == nil {
			return
		}
		if n.SourceContentHash == "" {
			n.SourceContentHash = n.ContentHash
		}
		n.ContentHash = hash
		bind(n.Preparation)
		for _, c := range n.StageModels {
			bind(c)
		}
	}
	bind(m)
	fmt.Printf("content transfer %s -> %s\n", old, hash)
}
