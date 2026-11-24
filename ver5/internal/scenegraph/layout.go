package scenegraph

import (
	"slices"
)

// Layout sizes: a card and the gaps between cards, in pixels.
const (
	CardW  = 196
	CardH  = 60
	colGap = 54
	rowGap = 22
	pad    = 28
	top    = 56 // room above the first row for a group's title
)

// Group is a box drawn round the cards of one walk's areas.
type Group struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
}

type cell struct{ col, row int }

// layout puts each walk into columns: the start, then the hub area with its branches (one column deeper for each
// exit away from the hub), then the lane of events the areas start, then what the story goes on with. Rows are
// shared: an area and the events it starts share a row.
func (b *builder) layout() {
	g := b.g
	idx := map[string]int{}
	for i, v := range g.Vertices {
		idx[v.ID] = i
	}
	out := map[string][]Edge{}
	for _, e := range g.Edges {
		out[e.From] = append(out[e.From], e)
	}
	placed := map[string]cell{}
	taken := map[cell]bool{}
	put := func(id string, col, row int, lane string) {
		for taken[cell{col, row}] {
			row++
		}
		taken[cell{col, row}] = true
		placed[id] = cell{col, row}
		g.Vertices[idx[id]].Lane = lane
	}
	rowOff := 0
	type box struct {
		id, title        string
		col0, col1, row0 int
		row1             int
	}
	var boxes []box
	maxRow, maxCol := 0, 0
	for _, ev := range g.Vertices {
		if ev.Kind != KindExplore {
			continue
		}
		if _, ok := placed[ev.ID]; ok {
			continue
		}
		// what leads into the walk and stands nowhere yet (the news before a walk) goes in a row of its own above it
		var intro []string
		for at := ev.ID; ; {
			prev := ""
			for _, e := range g.Edges {
				if e.To == at && e.Kind != EdgeBack || e.To == at && e.Label == "탐험 종료" {
					if _, ok := placed[e.From]; !ok && g.Vertices[idx[e.From]].Kind != KindExplore && g.Vertices[idx[e.From]].Kind != KindArea && !slices.Contains(intro, e.From) {
						prev = e.From
						break
					}
				}
			}
			if prev == "" {
				break
			}
			intro = append([]string{prev}, intro...)
			at = prev
		}
		if len(intro) > 0 {
			for i, id := range intro {
				put(id, i, rowOff, "flow")
			}
			rowOff++
		}
		var hub string
		var opening []string
		for _, e := range out[ev.ID] {
			switch e.Kind {
			case EdgeStart:
				hub = e.To
			case EdgeOpen:
				opening = append(opening, e.To)
			}
		}
		if hub == "" {
			continue
		}
		parent, depth := map[string]string{hub: ""}, map[string]int{hub: 0}
		children := map[string][]string{}
		queue := []string{hub}
		for len(queue) > 0 {
			a := queue[0]
			queue = queue[1:]
			for _, e := range out[a] {
				if e.Kind == EdgeExit {
					if _, seen := parent[e.To]; !seen {
						parent[e.To], depth[e.To] = a, depth[a]+1
						children[a] = append(children[a], e.To)
						queue = append(queue, e.To)
						for i := range g.Edges {
							if g.Edges[i].From == a && g.Edges[i].To == e.To && g.Edges[i].Kind == EdgeExit {
								g.Edges[i].Tree = true
								break
							}
						}
					}
				}
			}
		}
		events := map[string][]string{hub: opening}
		for a := range parent {
			for _, e := range out[a] {
				if (e.Kind == EdgeTalk || e.Kind == EdgeGate) && g.Vertices[idx[e.To]].Kind == KindEvent && !slices.Contains(events[a], e.To) {
					if _, ok := placed[e.To]; !ok {
						events[a] = append(events[a], e.To)
					}
				}
			}
		}
		var dfs func(a string) []string
		dfs = func(a string) []string {
			list := []string{a}
			for _, c := range children[a] {
				list = append(list, dfs(c)...)
			}
			return list
		}
		var units [][]string
		for _, c := range children[hub] {
			units = append(units, dfs(c))
		}
		half := len(units) / 2
		var seq []string
		for _, u := range units[:half] {
			seq = append(seq, u...)
		}
		seq = append(seq, hub)
		for _, u := range units[half:] {
			seq = append(seq, u...)
		}
		maxDepth := 0
		for _, d := range depth {
			maxDepth = max(maxDepth, d)
		}
		laneCol := maxDepth + 2
		row, hubRow := rowOff, rowOff
		first := rowOff
		for _, a := range seq {
			if a == hub {
				hubRow = row
			}
			put(a, 1+depth[a], row, "area")
			for i, e := range events[a] {
				put(e, laneCol, row+i, "event")
			}
			row += max(1, len(events[a]))
		}
		put(ev.ID, 0, hubRow, "area")
		boxes = append(boxes, box{id: ev.ID, title: "탐험 구역 · " + g.Vertices[idx[hub]].Title + " 중심", col0: 1, col1: 1 + maxDepth, row0: first, row1: row - 1})
		// what the story goes on with: from the areas (the sortie) and from the events (leaving the walk)
		flowCol := laneCol + 1
		queue = nil
		for _, a := range seq {
			queue = append(queue, a)
			queue = append(queue, events[a]...)
		}
		for len(queue) > 0 {
			from := queue[0]
			queue = queue[1:]
			for _, e := range out[from] {
				if e.Kind == EdgeBack && e.Label != "탐험 종료" || e.Kind == EdgeExit {
					continue
				}
				if _, ok := placed[e.To]; ok || g.Vertices[idx[e.To]].Kind == KindExplore {
					continue
				}
				fc := placed[from]
				col := max(flowCol, fc.col+1)
				if g.Vertices[idx[from]].Lane == "area" {
					col = flowCol
				}
				put(e.To, col, fc.row, "flow")
				queue = append(queue, e.To)
			}
		}
		for _, c := range placed {
			maxRow, maxCol = max(maxRow, c.row), max(maxCol, c.col)
		}
		rowOff = maxRow + 3
	}
	// whatever is left (a story without a walk, the walks a flow leads into)
	for _, v := range g.Vertices {
		if _, ok := placed[v.ID]; ok {
			continue
		}
		col := 0
		for _, e := range g.Edges {
			if e.To == v.ID {
				if c, ok := placed[e.From]; ok {
					col = max(col, c.col+1)
				}
			}
		}
		put(v.ID, col, rowOff, "flow")
		rowOff++
		maxRow = max(maxRow, rowOff)
	}
	for id, c := range placed {
		v := &g.Vertices[idx[id]]
		v.X, v.Y = pad+c.col*(CardW+colGap), top+c.row*(CardH+rowGap)
		g.W, g.H = max(g.W, v.X+CardW+pad), max(g.H, v.Y+CardH+pad)
	}
	for _, bx := range boxes {
		g.Groups = append(g.Groups, Group{ID: bx.id, Title: bx.title,
			X: pad + bx.col0*(CardW+colGap) - 14, Y: top + bx.row0*(CardH+rowGap) - 34,
			W: (bx.col1-bx.col0+1)*(CardW+colGap) - colGap + 28, H: (bx.row1-bx.row0+1)*(CardH+rowGap) - rowGap + 44})
	}
}
