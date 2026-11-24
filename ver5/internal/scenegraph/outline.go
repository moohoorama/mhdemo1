package scenegraph

import (
	"slices"

	"srpg/internal/content"
)

// ChapterInfo is one chapter for the table of contents: where it starts and the battles in it.
type ChapterInfo struct {
	ID      string       `json:"id"`
	Title   string       `json:"title"`
	Start   string       `json:"start"`
	Battles []BattleInfo `json:"battles"`
}

// BattleInfo is a battle of a chapter, with the node after it: the place to start "from here".
type BattleInfo struct {
	Node  string `json:"node"`
	Stage string `json:"stage"`
	Name  string `json:"name"`
	After string `json:"after,omitempty"`
}

// Outline lists the chapters and their battles.
func Outline(d *content.Data) []ChapterInfo {
	var out []ChapterInfo
	for _, c := range d.Chapters {
		info := ChapterInfo{ID: c.ID, Battles: []BattleInfo{}}
		if len(c.Nodes) > 0 {
			info.Start = c.Nodes[0]
		}
		info.Title = c.ID + "장"
		if p := firstPlace(d, c); p != "" {
			info.Title += " · " + p
		}
		for _, id := range c.Nodes {
			n := d.Node(id)
			if n.Kind != "battle" {
				continue
			}
			bi := BattleInfo{Node: id, Stage: n.Stage, After: d.After(id)}
			if i := d.StageIndex(n.Stage); i >= 0 {
				bi.Name = d.Stages[i].Name
			}
			info.Battles = append(info.Battles, bi)
		}
		out = append(out, info)
	}
	return out
}

func firstPlace(d *content.Data, c content.Chapter) string {
	for _, id := range append(slices.Clone(c.Events), c.Nodes...) {
		if n := d.Node(id); n.Kind == "scene" && n.Place != "" {
			return n.Place
		}
	}
	return ""
}

// nodesBefore lists, in play order, the story nodes that have happened by the time the story reaches node start: the
// chapters before the start's chapter (their nodes, then their events), the earlier nodes of the start's own chapter,
// and its events when a battle came before the start (the walk before that battle is over).
func nodesBefore(d *content.Data, start string) []string {
	var out []string
	// the story has two tellings of its last chapters (연의 and 가상): only the nodes that lead to the start count
	var leads map[string]bool
	if len(d.Chapters) > 0 && len(d.Chapters[0].Nodes) > 0 && d.Chapters[0].Nodes[0] != start {
		leads = d.Leads(start)
	}
	counts := func(id string) bool { return leads == nil || leads[id] }
	for _, c := range d.Chapters {
		at := slices.Index(c.Nodes, start)
		if at < 0 && slices.Contains(c.Events, start) {
			at = len(c.Nodes)
		}
		if at < 0 {
			walked := false
			for _, id := range c.Nodes {
				if counts(id) {
					out = append(out, id)
					walked = walked || d.Node(id).Kind == "explore"
				}
			}
			if walked || leads == nil {
				out = append(out, c.Events...)
			}
			continue
		}
		fought := false
		for _, id := range c.Nodes[:min(at, len(c.Nodes))] {
			out = append(out, id)
			fought = fought || d.Node(id).Kind == "battle"
		}
		if fought {
			out = append(out, c.Events...)
		}
		break
	}
	return out
}

// AssumeBefore are the flags the story has set by the time it reaches node start.
func AssumeBefore(d *content.Data, start string) []string {
	set := map[string]bool{}
	for _, id := range nodesBefore(d, start) {
		n := d.Node(id)
		switch n.Kind {
		case "set":
			for k := range n.Flags {
				set[k] = true
			}
		case "battle": // what a won battle stores
			if i := d.StageIndex(n.Stage); i >= 0 {
				for k := range StageFlags(d.Stages[i]) {
					set[k] = true
				}
			}
		}
	}
	return slices.Sorted(func(yield func(string) bool) {
		for k := range set {
			if !yield(k) {
				return
			}
		}
	})
}

// RosterBefore are the officers in the party by the time the story reaches node start, in the order the story
// brings them: the campaign's roster, then each node's join or leave; the events of a chapter come with its first
// walk (they are played during it); a won battle brings those its events send in as allies (they stay in the party).
// Only the nodes that lead to the start count, since the last chapters are told twice.
func RosterBefore(d *content.Data, start string) []string {
	have := slices.Clone(d.Campaign.Roster)
	var leads map[string]bool
	if len(d.Chapters) > 0 && len(d.Chapters[0].Nodes) > 0 && d.Chapters[0].Nodes[0] != start {
		leads = d.Leads(start)
	}
	for _, c := range d.Chapters {
		walked := false
		for _, id := range c.Nodes {
			if id == start {
				return have
			}
			if leads != nil && !leads[id] {
				continue
			}
			n := d.Node(id)
			switch n.Kind {
			case "join", "leave":
				have = applyRoster(have, n)
			case "explore":
				if !walked {
					walked = true
					for _, ev := range c.Events {
						have = applyRoster(have, d.Node(ev))
					}
				}
			case "battle":
				for _, id := range stageJoins(d, n.Stage) {
					have = applyRoster(have, content.Node{Kind: "join", Character: id})
				}
			}
		}
		if slices.Contains(c.Nodes, start) {
			return have
		}
	}
	return have
}

func applyRoster(have []string, n content.Node) []string {
	switch n.Kind {
	case "join":
		if !slices.Contains(have, n.Character) {
			have = append(have, n.Character)
		}
	case "leave":
		have = slices.DeleteFunc(have, func(c string) bool { return c == n.Character })
	}
	return have
}

// stageJoins are the officers a battle's events send in as allies; they stay in the party once it is won.
func stageJoins(d *content.Data, stage string) []string {
	i := d.StageIndex(stage)
	if i < 0 {
		return nil
	}
	var out []string
	for _, ev := range d.Stages[i].Events {
		for _, a := range ev.Do {
			if a.Kind != "join" {
				continue
			}
			for _, sp := range a.Spawns {
				if c, ok := d.Characters[sp.Character]; ok && !c.Template && !slices.Contains(out, sp.Character) {
					out = append(out, sp.Character)
				}
			}
		}
	}
	return out
}
