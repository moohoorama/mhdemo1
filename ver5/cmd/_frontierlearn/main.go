package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"srpg/internal/autoplay"
	"srpg/internal/content"
)

var (
	contentDir       = flag.String("content", "reports/autoplay/campaign-learning/data-snapshot/assets/data", "frozen data")
	initial          = flag.String("initial", "reports/autoplay/campaign-learning/joint-initial.json", "joint model")
	stage            = flag.String("stage", "B07", "stage to train from genuine undefeated campaign states")
	pool             = flag.String("pool", "", "reuse saved real frontier states")
	model            = flag.String("model", "reports/autoplay/goal10/frontier/model.json", "output model")
	collect          = flag.Int("collect", 8, "new game attempts to collect real frontier states")
	episodes         = flag.Int("episodes", 256, "frontier training battles")
	warm             = flag.Int("warm", 20, "imitation passes over successful teacher battles")
	par              = flag.Int("par", 2, "parallel simulations")
	k                = flag.Int("teacher-k", 1, "teacher search candidates; 1 uses the rule evaluator")
	goalInputs       = flag.Bool("goal-inputs", false, "observe objective areas in the stage expert")
	effectInputs     = flag.Bool("effect-inputs", true, "new experts distinguish buff effects and original item quantities")
	teacherRisk      = flag.Float64("teacher-risk", .03521889423931956, "teacher threat penalty")
	teacherXP        = flag.Float64("teacher-xp", .3367455612453371, "teacher experience value")
	teacherThreat    = flag.String("teacher-threat", "", "override teacher threat mode, e.g. officers for every expected blow")
	teacherExplore   = flag.Float64("teacher-explore", 0, "probability of exploring one of the teacher's three best actions")
	teacherLordLast  = flag.Bool("teacher-lord-last", false, "teacher orders other officers before the lord")
	teacherGlobal    = flag.Bool("teacher-global", false, "teacher compares all ready officers together")
	teacherDuelFirst = flag.Bool("teacher-duel-first", false, "teacher orders an available duelist before other officers")
	teacherLordDelay = flag.Int("teacher-lord-delay", 0, "teacher keeps the lord from moving during these opening rounds")
	teacherGames     = flag.Int("teachers", 0, "teacher battles; zero uses each frontier once")
	teacherData      = flag.String("teacher-data", "", "reuse saved successful demonstrations")
	trainPrep        = flag.Bool("train-preparation", true, "update the already learned preparation policy")
	stagePrep        = flag.Bool("stage-preparation", false, "isolate this stage's preparation weights from earlier stages")
	temperature      = flag.Float64("temperature", .3, "training battle sampling temperature")
	progressReward   = flag.Float64("progress-reward", 0, "auxiliary progress among failed frontier battles, capped at .25")
	rate             = flag.Float64("rate", 1e-4, "PPO learning rate")
	epochs           = flag.Int("epochs", 2, "PPO passes per collection")
	selfImitate      = flag.Int("self-imitate", 0, "extra imitation passes on this batch's neural victories")
	targetStreak     = flag.Int("target-streak", 20, "fresh-game unbeaten stages required for goal checkpoint")
	seed             = flag.Uint64("seed", 290001, "first new game seed")
)

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func save(path string, x any) {
	b, err := json.Marshal(x)
	check(err)
	saveBytes(path, b)
}
func saveBytes(path string, b []byte) {
	check(os.MkdirAll(filepath.Dir(path), 0755))
	check(os.WriteFile(path+".tmp", b, 0644))
	check(os.Rename(path+".tmp", path))
}
func saveSamples(path string, x any) {
	b, err := json.Marshal(x)
	check(err)
	var compressed bytes.Buffer
	z, err := gzip.NewWriterLevel(&compressed, gzip.BestSpeed)
	check(err)
	_, err = z.Write(b)
	check(err)
	check(z.Close())
	saveBytes(path+".gz", compressed.Bytes())
}
func readSamples(path string, x any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.HasSuffix(path, ".gz") {
		z, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return err
		}
		b, err = io.ReadAll(z)
		closeErr := z.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return json.Unmarshal(b, x)
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

func main() {
	flag.Parse()
	if *par < 1 || *collect < 1 || *episodes < 0 || *warm < 0 || *k < 1 || *teacherGames < 0 || *teacherExplore < 0 || *teacherExplore > 1 || *teacherRisk < 0 || *teacherXP < 0 || *temperature <= 0 || *rate <= 0 || *epochs < 1 || *progressReward < 0 || *progressReward > .25 {
		check(fmt.Errorf("invalid settings"))
	}
	d, err := content.Load(*contentDir)
	check(err)
	m, err := autoplay.LoadBattleModel(*initial)
	check(err)
	if d.Hash != m.ContentHash {
		check(fmt.Errorf("content differs"))
	}
	hadExpert := m.StageModels[*stage] != nil
	check(m.AddStageExpert(*stage))
	if *stagePrep {
		check(m.EnableStagePreparation(*stage))
	}
	if *goalInputs {
		m.StageModels[*stage].EnableGoalInputs()
	}
	if *effectInputs && !hadExpert {
		m.StageModels[*stage].EnableEffectInputs()
	}
	check(os.MkdirAll(filepath.Dir(*model), 0755))
	started := time.Now()
	var starts []autoplay.FrontierStart
	if *pool != "" {
		b, err := os.ReadFile(*pool)
		check(err)
		check(json.Unmarshal(b, &starts))
	} else {
		found := make([]*autoplay.FrontierStart, *collect)
		reports := make([]autoplay.Report, *collect)
		parallel(*collect, func(i int) {
			var err error
			found[i], reports[i], err = autoplay.CollectFrontier(d, m, *seed+uint64(i), *stage)
			check(err)
			fmt.Printf("collect %d %s elapsed %v\n", i+1, autoplay.Summary(reports[i]), time.Since(started).Round(time.Second))
		})
		for _, s := range found {
			if s != nil {
				starts = append(starts, *s)
			}
		}
		save(*model+".collection.json", reports)
		save(*model+".frontiers.json", starts)
	}
	if len(starts) == 0 {
		check(fmt.Errorf("no real undefeated frontier reached"))
	}
	fmt.Printf("real frontiers %d; resource/level/RNG state retained\n", len(starts))
	nteachers := *teacherGames
	if nteachers == 0 {
		nteachers = len(starts)
	}
	teachers := make([]autoplay.CampaignEpisode, nteachers)
	if *teacherData != "" {
		check(readSamples(*teacherData, &teachers))
		if *stagePrep {
			for i := range teachers {
				if rows := teachers[i].Samples["prep"]; len(rows) > 0 {
					teachers[i].Samples["prep:"+*stage] = rows
					delete(teachers[i].Samples, "prep")
				}
			}
		}
	} else {
		parallel(nteachers, func(i int) {
			var err error
			teachers[i], err = autoplay.PlayFrontier(d, m, starts[i%len(starts)], *seed+uint64(i), autoplay.FrontierOptions{Teacher: &autoplay.FrontierTeacher{K: *k, Risk: *teacherRisk, XP: *teacherXP, Threat: *teacherThreat, Explore: *teacherExplore, LordLast: *teacherLordLast, Global: *teacherGlobal, DuelFirst: *teacherDuelFirst, LordDelay: *teacherLordDelay}})
			check(err)
			if !teachers[i].Report.Stages[0].Won {
				teachers[i].Samples = nil
			}
			fmt.Printf("teacher %d/%d victory=%v round=%d elapsed %v\n", i+1, nteachers, teachers[i].Report.Stages[0].Won, teachers[i].Report.Stages[0].Rounds, time.Since(started).Round(time.Second))
		})
		saveSamples(*model+".teachers.json", teachers)
	}
	t := autoplay.NewCampaignTrainer(m, *seed)
	if !*trainPrep {
		delete(t.Models, "prep")
		delete(t.Models, "prep:"+*stage)
	}
	keys := make([]string, 0, len(t.Models))
	for key := range t.Models {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		var rows []autoplay.BattleSample
		for _, e := range teachers {
			if e.Report.Stages[0].Won {
				rows = append(rows, e.Samples[key]...)
			}
		}
		fmt.Printf("teacher decisions %q %d\n", key, len(rows))
		if len(rows) > 0 {
			for _, row := range rows {
				for _, features := range row.Inputs.Features {
					if len(features) != t.Models[key].Model.Actor.Sizes[0] {
						check(fmt.Errorf("demonstration dimensions differ for %s; collect matching inputs", key))
					}
				}
			}
			t.Models[key].Imitate(rows, *warm, 32, 1e-3)
		}
	}
	check(m.Save(*model + ".warm.json"))
	var best *autoplay.CampaignAssessment
	validate := func(done int) {
		var score autoplay.CampaignAssessment
		for i := range 4 {
			e := autoplay.PlayCampaignEpisode(d, m, 310001+uint64(i), false, false, 1)
			if e.Error != "" {
				check(fmt.Errorf("eval: %s", e.Error))
			}
			score.Add(e.Report)
		}
		fmt.Printf("full campaign dev after %d mean %.2f min %d max %d\n", done, score.Mean(), score.Min, score.Max)
		if best == nil || score.Better(*best) {
			copy := score
			best = &copy
			check(m.Save(*model + ".best.json"))
			save(*model+".best.eval.json", score)
		}
		if score.Max >= *targetStreak {
			check(m.Save(*model + ".goal.json"))
			save(*model+".goal.eval.json", score)
		}
	}
	validate(0)
	savedSuccesses := 0
	for done := 0; done < *episodes; {
		n := min(8, *episodes-done)
		results := make([]autoplay.CampaignEpisode, n)
		parallel(n, func(i int) {
			var err error
			results[i], err = autoplay.PlayFrontier(d, m, starts[(done+i)%len(starts)], *seed+uint64(done+i), autoplay.FrontierOptions{Training: true, Temperature: *temperature, ProgressReward: *progressReward})
			check(err)
		})
		wins := 0
		var successes []autoplay.CampaignEpisode
		for _, e := range results {
			if e.Report.Stages[0].Won {
				wins++
				successes = append(successes, e)
			}
		}
		t.Update(results, false, *epochs, 64, *rate)
		saveSuccess := len(successes) > 0 && savedSuccesses < 4
		if len(successes) > 0 {
			if saveSuccess {
				saveSamples(fmt.Sprintf("%s.success.%06d.json", *model, done+n), successes[:1])
				savedSuccesses++
			}
			if *selfImitate > 0 {
				for _, key := range keys {
					var rows []autoplay.BattleSample
					for _, e := range successes {
						rows = append(rows, e.Samples[key]...)
					}
					if len(rows) > 0 {
						t.Models[key].Imitate(rows, *selfImitate, 32, *rate)
					}
				}
			}
		}
		done += n
		check(m.Save(*model))
		if saveSuccess {
			check(m.Save(fmt.Sprintf("%s.%06d.json", *model, done)))
		}
		if done%64 == 0 {
			check(m.Save(*model + ".checkpoint.json"))
		}
		fmt.Printf("frontier %d/%d victory %d/%d elapsed %v\n", done, *episodes, wins, n, time.Since(started).Round(time.Second))
		if done%64 == 0 || done == *episodes {
			validate(done)
		}
	}
}
