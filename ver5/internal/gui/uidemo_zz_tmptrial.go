package gui

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["trial"] = demoTrial }

// demoTrial plays a trial chapter kept outside assets (-stage cmd/_scene15, see AddTrialChapter) from its first node
// or -node, turning the lines on by itself.
func demoTrial(d *Demo) error {
	g := d.g
	first, err := AddTrialChapter(g.S.Data, d.stage)
	if err != nil {
		return err
	}
	g.names = nameIndex(g.S.Data)
	if err := g.AddScenes(filepath.Join(d.stage, "scenes.json")); err != nil {
		return err
	}
	if err := StartTrial(g.S, first, cmp.Or(d.node, first)); err != nil {
		return err
	}
	g.S.Settings.InstantText = true
	shown := 0
	d.step = func(f int) { // the next line 100 frames after the beats before it have played out
		if os.Getenv("TRIAL_DEBUG") != "" && f%50 == 0 {
			o := g.S.View()
			fmt.Fprintln(os.Stderr, f, o.Node, o.Phase, g.lineIndex, g.sceneBusy(), g.narrationLeaving, g.held != nil, len(g.modals), g.overlay())
		}
		g.turnPage() // as Update ends a narration that faded out and a node whose closing beats are done
		if g.sceneBusy() || g.intro != nil || g.S.View().Phase != "scenario" {
			shown = f
		}
		if f-shown >= 100 {
			shown = f
			g.advanceDialogue()
		}
	}
	return nil
}

// StartTrial starts a new game at the trial chapter's first node and plays on to node (a scene node: to its first line).
func StartTrial(s *session.Session, first, node string) error {
	for i := 0; i < 10 && s.Data.Node(node).Kind == "scene"; i++ {
		node = s.Data.After(node)
	}
	if r := s.Handle(session.Request{Op: "jump", Seed: 1, Node: first, Discard: true}); r.Error != "" {
		return fmt.Errorf("jump to %s: %s", first, r.Error)
	}
	do := func(c core.Command) error {
		rev := s.View().Revision
		if r := s.Handle(session.Request{Op: "command", Revision: &rev, Command: c}); r.Error != "" {
			return fmt.Errorf("%s at %s: %s", c.Kind, s.View().Node, r.Error)
		}
		return nil
	}
	for l := s.View().Learners; len(l) > 0; l = s.View().Learners {
		if err := do(core.Command{Kind: "learn_close", Actor: l[0].ID}); err != nil {
			return err
		}
	}
	for i := 0; i < 500 && s.View().Node != node; i++ {
		if err := do(core.Command{Kind: "next"}); err != nil {
			return err
		}
	}
	if n := s.View().Node; n != node {
		return fmt.Errorf("stopped at %s on the way to %s", n, node)
	}
	return nil
}

// AddTrialChapter puts a trial chapter kept in dir into the content: cast.json's people, and story.json's nodes as
// one more chapter at the end. It returns the chapter's first node; the staging, dir/scenes.json, goes to AddScenes.
func AddTrialChapter(d *content.Data, dir string) (string, error) {
	read := func(name string, v any) error {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, v); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	var cast struct{ Characters map[string]content.Character }
	if err := read("cast.json", &cast); err != nil {
		return "", err
	}
	for id, c := range cast.Characters {
		d.Characters[id] = c
	}
	var story struct{ Nodes []content.Node }
	if err := read("story.json", &story); err != nil {
		return "", err
	}
	if len(story.Nodes) == 0 {
		return "", fmt.Errorf("%s/story.json has no nodes", dir)
	}
	ch := content.Chapter{ID: "trial", File: filepath.Join(dir, "story.json")}
	for _, n := range story.Nodes {
		d.Nodes = append(d.Nodes, n)
		ch.Nodes = append(ch.Nodes, n.ID)
	}
	d.Chapters = append(d.Chapters, ch)
	return ch.Nodes[0], nil
}
