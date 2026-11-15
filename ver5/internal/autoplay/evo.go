package autoplay

import (
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"sort"
	"sync"

	"srpg/internal/content"
)

// Fitness ranks campaigns for the tuning: battles won first, then won at the first try and the bonus
// objectives, then the level score; tries spent cost a little.
func Fitness(reps []Report) float64 {
	sum := 0.0
	for _, r := range reps {
		for _, s := range r.Stages {
			if s.Won {
				sum += 100 + 10*float64(s.Subs)
			}
			if s.FirstWin {
				sum += 20
			}
			sum -= 5 * float64(s.Tries-1)
		}
		sum += r.LevelScore
	}
	return sum / float64(max(1, len(reps)))
}

// EvoOptions sets the tuning: the cross-entropy method over the genome, each candidate playing the campaign
// with every seed (the same seeds for all, so they are compared on the same dice).
type EvoOptions struct {
	Pop, Elite, Gens int
	Seeds            []uint64
	Difficulty       string
	Stages           int
	Par              int
	Start            Genome
	Log              io.Writer
}

// Generation is one generation's outcome.
type Generation struct {
	Best, Mean float64
	Genome     Genome
}

// Evolve tunes the genome and gives the best found with each generation's record.
func Evolve(d *content.Data, opt EvoOptions) (Genome, float64, []Generation) {
	rng := rand.New(rand.NewPCG(7, 11))
	mean := opt.Start
	var sigma Genome
	for i := range sigma {
		sigma[i] = .5 * max(.5, math.Abs(mean[i]))
	}
	best, bestFit := mean, math.Inf(-1)
	var gens []Generation
	for gen := range opt.Gens {
		pop := make([]Genome, opt.Pop)
		pop[0] = mean
		for k := 1; k < opt.Pop; k++ {
			for i := range pop[k] {
				pop[k][i] = mean[i] + sigma[i]*rng.NormFloat64()
			}
		}
		fit := make([]float64, opt.Pop)
		var mu sync.Mutex
		parallel(opt.Pop*len(opt.Seeds), opt.Par, func(job int) {
			k, s := job/len(opt.Seeds), opt.Seeds[job%len(opt.Seeds)]
			p := &RulePlayer{Label: "evo", D: d, G: pop[k]}
			r, err := Run(d, p, Options{Difficulty: opt.Difficulty, Seed: s, Stages: opt.Stages})
			f := Fitness([]Report{r})
			if err != nil {
				f = -1e6
			}
			mu.Lock()
			fit[k] += f / float64(len(opt.Seeds))
			mu.Unlock()
		})
		order := make([]int, opt.Pop)
		for i := range order {
			order[i] = i
		}
		sort.SliceStable(order, func(a, b int) bool { return fit[order[a]] > fit[order[b]] })
		if fit[order[0]] > bestFit {
			best, bestFit = pop[order[0]], fit[order[0]]
		}
		avg := 0.0
		for _, f := range fit {
			avg += f / float64(opt.Pop)
		}
		var next, spread Genome
		for _, k := range order[:opt.Elite] {
			for i := range next {
				next[i] += pop[k][i] / float64(opt.Elite)
			}
		}
		for _, k := range order[:opt.Elite] {
			for i := range spread {
				spread[i] += (pop[k][i] - next[i]) * (pop[k][i] - next[i]) / float64(opt.Elite)
			}
		}
		for i := range sigma {
			sigma[i] = max(.05, .3*sigma[i]+.7*math.Sqrt(spread[i]))
		}
		mean = next
		gens = append(gens, Generation{Best: fit[order[0]], Mean: avg, Genome: pop[order[0]]})
		if opt.Log != nil {
			fmt.Fprintf(opt.Log, "gen %d best %.1f mean %.1f (best ever %.1f)\n", gen+1, fit[order[0]], avg, bestFit)
		}
	}
	return best, bestFit, gens
}
