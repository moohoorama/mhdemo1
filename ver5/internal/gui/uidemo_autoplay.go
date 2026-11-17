package gui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

func init() { demos["autoplay"] = demoAutoplay }

type autoplayRecord struct {
	Stage                     string
	Try                       int
	Snapshot                  *core.State
	Command                   core.Command
	Round                     int
	Turn, Army, Phase, Result string
	Units                     []core.UnitView
	Events                    []core.Event
}

type autoplayRecording struct {
	stage    string
	try      int
	snapshot core.State
	steps    []autoplayRecord
}

// readAutoplayRecording selects the stage's last attempt, stopping before later stages.
func readAutoplayRecording(r io.Reader, stage string) (autoplayRecording, error) {
	var out autoplayRecording
	selected := false
	dec := json.NewDecoder(r)
	for {
		var row autoplayRecord
		if err := dec.Decode(&row); err != nil {
			if err == io.EOF {
				break
			}
			return out, fmt.Errorf("read autoplay trace: %w", err)
		}
		if row.Snapshot != nil {
			if row.Stage != stage && selected {
				break
			}
			selected = row.Stage == stage
			if selected {
				out = autoplayRecording{stage: row.Stage, try: row.Try, snapshot: *row.Snapshot}
			}
		} else if selected {
			out.steps = append(out.steps, row)
		}
	}
	if len(out.steps) == 0 || out.steps[len(out.steps)-1].Result == "" {
		return out, fmt.Errorf("trace has no complete attempt for %s", stage)
	}
	return out, nil
}

func applyAutoplayRecord(e *core.Engine, step autoplayRecord) (core.Result, error) {
	r, err := e.Apply(step.Command)
	if err != nil {
		return r, err
	}
	o := e.Observe()
	if o.Round != step.Round || o.Turn != step.Turn || o.Army != step.Army || o.Phase != step.Phase || o.Result != step.Result ||
		!reflect.DeepEqual(o.UnitViews, step.Units) || !reflect.DeepEqual(r.Events, step.Events) {
		return r, fmt.Errorf("autoplay trace differs after %+v", step.Command)
	}
	return r, nil
}

// demoAutoplay renders recorded commands; it never asks an AI to choose new ones.
func demoAutoplay(d *Demo) error {
	f, err := os.Open(d.trace)
	if err != nil {
		return err
	}
	defer f.Close()
	rec, err := readAutoplayRecording(f, d.stage)
	if err != nil {
		return err
	}
	g := d.g
	e, err := core.Restore(g.S.Data, rec.snapshot)
	if err != nil {
		return err
	}
	if g.S.Data.Stages[e.Observe().Stage].ID != rec.stage {
		return fmt.Errorf("trace stage does not match its snapshot")
	}
	r, err := e.Apply(core.Command{Kind: "start"})
	if err != nil {
		return err
	}
	next, opening := 0, core.Command{Kind: "start"}
	for e.Observe().Phase == "deployment" && next < len(rec.steps) {
		opening = rec.steps[next].Command
		r, err = applyAutoplayRecord(e, rec.steps[next])
		if err != nil {
			return err
		}
		next++
	}
	g.S.Engine, g.S.Screen = e, "playing"
	g.S.Settings.Detail = true
	g.speed = 2
	g.ensureField(e.Observe())
	g.sync(e.Observe())
	g.modals = nil
	g.replayOpening(opening, r.Events)
	fmt.Fprintf(os.Stderr, "autoplay replay %s seed %d attempt %d: %d commands, 2x\n", rec.stage, rec.snapshot.Seed, rec.try, len(rec.steps))
	held, finished := 0, 0
	d.step = func(frame int) {
		if len(g.modals) > 0 && (!g.busy() || g.briefed()) {
			held++
			if held >= 45 {
				g.closeModal()
				held = 0
			}
			return
		}
		if g.busy() {
			return
		}
		if next == len(rec.steps) {
			finished++
			if finished == 180 {
				fmt.Fprintf(os.Stderr, "autoplay replay verified: %s %s round %d, %d frames\n", rec.stage, e.Observe().Result, e.Observe().Round, frame)
				d.err = ebiten.Termination
			}
			return
		}
		step := rec.steps[next]
		before := e.Observe()
		r, err := applyAutoplayRecord(e, step)
		if err != nil {
			d.err = err
			return
		}
		next++
		g.logCommand(step.Command, r.Events)
		g.record(&before, r.Events)
		g.replayOne(before, step.Command, r.Events)
	}
	return nil
}
