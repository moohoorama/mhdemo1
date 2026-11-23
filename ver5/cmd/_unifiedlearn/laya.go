package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"srpg/internal/autoplay"
	"srpg/internal/content"
	"srpg/internal/core"
)

// decision is one battle choice written as text: the public state, the legal options and the teacher's index.
type decision struct {
	Seed    uint64   `json:"seed"`
	Stage   string   `json:"stage"`
	State   string   `json:"state"`
	Options []string `json:"options"`
	Label   int      `json:"label"`
}

// stateText writes what the battle shows: goals with their conditions, the bonus objectives and whether each is
// done, the round and limit, the money, and every unit's id, side, class, level, HP and cell.
func stateText(d *content.Data, e *core.Engine) string {
	o := e.Observe()
	st := d.Stages[o.Stage]
	vic, def, subs := st.Victory, st.Defeat, st.Sub
	if o.Goals != nil {
		vic, def, subs = o.Goals.Victory, o.Goals.Defeat, o.Goals.Sub
	}
	var b strings.Builder
	fmt.Fprintf(&b, "stage %s round %d of %d. money %d.\n", st.ID, o.Round, st.TurnLimit, o.Money)
	b.WriteString("victory:")
	for _, g := range vic {
		fmt.Fprintf(&b, " [%s]", strings.Join(g.If, " "))
	}
	b.WriteString("\ndefeat:")
	for _, g := range def {
		fmt.Fprintf(&b, " [%s]", strings.Join(g.If, " "))
	}
	done := e.SubDone()
	b.WriteString("\nbonus:")
	for i, s := range subs {
		ok := i < len(done) && done[i]
		fmt.Fprintf(&b, " [%s done=%v]", strings.Join(s.If, " "), ok)
	}
	b.WriteString("\n")
	for _, u := range o.UnitViews {
		if u.HP <= 0 {
			continue
		}
		fmt.Fprintf(&b, "unit %s %s %s lv%d hp%d at %d,%d done=%v\n", u.ID, u.Faction, u.Def, u.Level, u.HP, u.X, u.Y, u.Done)
	}
	return b.String()
}

// commandText writes one legal command with the game's own preview numbers for it.
func commandText(e *core.Engine, c core.Command) string {
	var b strings.Builder
	b.WriteString(c.Kind)
	if c.Actor != "" {
		b.WriteString(" " + c.Actor)
	}
	if c.Skill != "" {
		b.WriteString(" skill " + c.Skill)
	}
	if c.Item != "" {
		b.WriteString(" item " + c.Item)
	}
	if c.Target != "" {
		b.WriteString(" target " + c.Target)
	}
	if c.Kind == "move" || c.X != 0 || c.Y != 0 {
		fmt.Fprintf(&b, " to %d,%d", c.X, c.Y)
	}
	if p, err := e.Preview(c); err == nil {
		fmt.Fprintf(&b, " hit %.0f dmg %d heal %d xp %d", p.Hit, p.MaxDamage, p.Healing, p.XP)
	}
	return b.String()
}

func optionTexts(e *core.Engine, cmds []core.Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = commandText(e, c)
	}
	return out
}

// teacherRec plays the rule search AI and writes each of its battle decisions as a text row.
type teacherRec struct {
	*autoplay.NeuralPlayer
	d    *content.Data
	last *autoplay.BattleSample
	rows []decision
	seed uint64
}

func newTeacher(d *content.Data, seed uint64) *teacherRec {
	t := &teacherRec{d: d, seed: seed}
	t.NeuralPlayer = &autoplay.NeuralPlayer{Searcher: autoplay.CampaignPlayer(d), Model: noPrep(), BattleTeacher: true, WarmPreparation: true, BattleTemperature: 1,
		Record: func(key string, s autoplay.BattleSample) {
			if key == "" {
				t.last = &s
			}
		}}
	return t
}

func (t *teacherRec) Act(e *core.Engine, actor string) core.Command {
	state := stateText(t.d, e)
	t.last = nil
	c := t.NeuralPlayer.Act(e, actor)
	if t.last == nil {
		return c
	}
	cmds := t.last.Inputs.Commands
	t.rows = append(t.rows, decision{Seed: t.seed, Stage: t.d.Stages[e.Observe().Stage].ID, State: state, Options: optionTexts(e, cmds), Label: t.last.Chosen})
	return c
}

// noPrep is a battle model with no preparation or stage networks: the preparation, learning and exploration
// then come from the same rule search for every policy compared here.
func noPrep() *autoplay.BattleModel { return autoplay.NewBattleModel(1, 0) }

// layaBridge is a Python process holding the Laya model; it answers one choice per JSON line.
type layaBridge struct {
	cmd      *exec.Cmd
	in       io.WriteCloser
	out      *bufio.Reader
	requests int
	calls    int
	seconds  float64
}

type bridgeRequest struct {
	ID      int      `json:"id"`
	State   string   `json:"state"`
	Options []string `json:"options"`
}

type bridgeResponse struct {
	ID      int     `json:"id"`
	Choice  int     `json:"choice"`
	Calls   int     `json:"calls"`
	Seconds float64 `json:"seconds"`
	Error   string  `json:"error"`
}

func startLaya(python, script, dir, head string) (*layaBridge, error) {
	args := []string{"-I", script, "--laya", dir}
	if head != "" {
		args = append(args, "--head", head)
	}
	cmd := exec.Command(python, args...)
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &layaBridge{cmd: cmd, in: in, out: bufio.NewReaderSize(out, 1<<20)}, nil
}

func (b *layaBridge) choose(state string, options []string) (int, error) {
	req := bridgeRequest{ID: b.requests, State: state, Options: options}
	b.requests++
	if err := json.NewEncoder(b.in).Encode(req); err != nil {
		return 0, err
	}
	line, err := b.out.ReadBytes('\n')
	if err != nil {
		return 0, fmt.Errorf("laya bridge: %w", err)
	}
	var r bridgeResponse
	if err := json.Unmarshal(line, &r); err != nil {
		return 0, err
	}
	if r.Error != "" {
		return 0, errors.New(r.Error)
	}
	if r.Choice < 0 || r.Choice >= len(options) {
		return 0, fmt.Errorf("laya choice %d out of %d options", r.Choice, len(options))
	}
	b.calls += r.Calls
	b.seconds += r.Seconds
	return r.Choice, nil
}

func (b *layaBridge) close() {
	b.in.Close()
	b.cmd.Wait()
}
