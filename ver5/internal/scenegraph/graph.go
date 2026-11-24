// Package scenegraph turns the story's nodes and explore files into a graph of what the player can reach from a
// start node — areas joined by exits, events started by talking, the sortie, the battle — and checks the graph, so a
// scenario can be read and verified without playing it. It stops at the first battle it meets.
package scenegraph

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/scenemap"
)

// Vertex kinds.
const (
	KindExplore = "explore" // a walk's start: the explore file and the event that opens it
	KindArea    = "area"
	KindEvent   = "event" // a run of story nodes
	KindBattle  = "battle"
	KindEnd     = "end"
)

// Vertex is one place of the graph; X and Y are where the layout puts its card.
type Vertex struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Sub   string `json:"sub,omitempty"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Lane  string `json:"lane,omitempty"` // "area", "event" or "flow": the column it stands in
}

// Edge kinds.
const (
	EdgeOpen  = "open"  // the walk's opening event
	EdgeStart = "start" // where the walk begins
	EdgeExit  = "exit"  // an exit to another area
	EdgeGate  = "gate"  // an exit that leads into an event
	EdgeTalk  = "talk"  // talking to someone starts an event
	EdgeSort  = "sortie"
	EdgeBack  = "back"
	EdgeNext  = "next" // the story goes on
	EdgeCase  = "case" // a branch
)

// Edge joins two vertices; Cond are the story conditions that must all hold for it to open.
type Edge struct {
	From  string   `json:"from"`
	To    string   `json:"to"`
	Kind  string   `json:"kind"`
	Label string   `json:"label,omitempty"`
	Cond  []string `json:"cond,omitempty"`
	Side  bool     `json:"side,omitempty"` // a side event: the sortie does not wait for it
	Tree  bool     `json:"tree,omitempty"` // an exit the layout hangs a branch from; the other exits of an area are drawn when it is picked
}

// Issue is a finding of the checks.
type Issue struct {
	Level string `json:"level"` // error or warn
	Where string `json:"where"` // a vertex id or ""
	Msg   string `json:"msg"`
	Rule  string `json:"rule,omitempty"` // the scenario rule it breaks (rules.go)
	// Subject is who the finding is about (a character id), so a list of findings can be told apart and counted.
	Subject string `json:"subject,omitempty"`
}

// FlagUse tells where a story flag is set and where a condition reads it.
type FlagUse struct {
	Name   string   `json:"name"`
	SetBy  []string `json:"setBy"`
	UsedBy []string `json:"usedBy"`
}

// Graph is everything reachable from Start up to the first battle, or the n-th when Options.Battles says.
type Graph struct {
	Start    string            `json:"start"`
	Vertices []Vertex          `json:"vertices"`
	Edges    []Edge            `json:"edges"`
	Issues   []Issue           `json:"issues"`
	Flags    []FlagUse         `json:"flags"`
	Groups   []Group           `json:"groups"`
	Steps    []Step            `json:"steps"`
	Texts    map[string]string `json:"texts"` // a condition string -> how it reads in Korean
	Assumed  []string          `json:"assumed,omitempty"`
	W        int               `json:"w"`
	H        int               `json:"h"`
}

// Options say how far the graph reaches and what it takes as already done.
type Options struct {
	Battles int      // how many battles to pass through before stopping; 0 or 1 stops at the first
	Assume  []string // flags taken as set before the start
}

// Env is what the checks need beyond the story: the scene maps and their rules, and which sprite sheets exist.
type Env struct {
	Map      func(id string) *scenemap.Map
	Catalog  *scenemap.Catalog
	HasSheet func(id string) bool
	Reach    func(mapID string, at [2]int) int // the small tiles walkable from at; when nil, counted from Map and Catalog
	Portrait func(name string) bool            // whether graphics/portraits/<name>.png exists; when nil, portraits are not checked
}

// Run is a run of story nodes: from Start to the node that ends it.
type Run struct {
	Start string
	Nodes []content.Node
	End   content.Node // the node the run stops at (back, branch, battle, explore, end); zero when it ran out
}

type builder struct {
	stopped bool // a battle's aftermath was left out, so the rest of its chapter is not in the graph
	opt     Options
	battles int
	setsOf  map[string]map[string]int // event vertex -> the flags its set nodes store
	rosters map[string][]content.Node // event vertex -> its join and leave nodes
	d       *content.Data
	env     Env
	g       *Graph
	have    map[string]bool
	queue   []func()
	ctx     map[string]string   // event vertex -> the explore node whose walk it belongs to
	from    map[string][]string // event vertex -> the area vertices that start it
	backs   []backRef
	visited map[string]bool // story node ids walked
	sets    map[string][]string
	uses    map[string][]string
	walk    map[string]*scenemap.Walk
}

type backRef struct {
	vertex  string
	node    content.Node
	explore string
}

// AreaID is the vertex id of an area of an explore node's file.
func AreaID(file, area string) string { return "area:" + file + "/" + area }

// Build lays out the graph from the story node start.
func Build(d *content.Data, env Env, start string, opt Options) *Graph {
	b := &builder{opt: opt, setsOf: map[string]map[string]int{}, rosters: map[string][]content.Node{}, d: d, env: env, g: &Graph{Start: start, Vertices: []Vertex{}, Edges: []Edge{}, Issues: []Issue{}, Flags: []FlagUse{}, Groups: []Group{}, Steps: []Step{}, Texts: map[string]string{}}, have: map[string]bool{}, ctx: map[string]string{}, from: map[string][]string{},
		visited: map[string]bool{}, sets: map[string][]string{}, uses: map[string][]string{}, walk: map[string]*scenemap.Walk{}}
	if d.Node(start).ID == "" {
		b.issue("error", "", fmt.Sprintf("시작 노드 %q 가 없습니다", start))
		return b.g
	}
	b.node(start, "")
	for len(b.queue) > 0 {
		f := b.queue[0]
		b.queue = b.queue[1:]
		f()
	}
	b.resolveBacks()
	b.finish()
	b.g.Issues = uniqIssues(b.g.Issues)
	b.g.Assumed = slices.Sorted(slices.Values(opt.Assume))
	b.describe()
	b.follow()
	b.layout()
	return b.g
}

func (b *builder) issue(level, where, msg string) {
	b.g.Issues = append(b.g.Issues, Issue{Level: level, Where: where, Msg: msg})
}

func (b *builder) vertex(v Vertex) bool {
	if b.have[v.ID] {
		return false
	}
	b.have[v.ID] = true
	b.g.Vertices = append(b.g.Vertices, v)
	return true
}

func (b *builder) edge(from, to, kind, label string, cond []string) {
	b.g.Edges = append(b.g.Edges, Edge{From: from, To: to, Kind: kind, Label: label, Cond: cond})
	for _, c := range cond {
		b.use(c, from)
	}
}

// use records the flags a condition reads.
func (b *builder) use(cond, where string) {
	c, err := content.ParseCond(cond)
	if err != nil {
		b.issue("error", where, fmt.Sprintf("조건 %q 를 읽을 수 없습니다", cond))
		return
	}
	var walk func(c content.Cond)
	walk = func(c content.Cond) {
		if c.Op == "flag" {
			b.uses[c.Arg] = append(b.uses[c.Arg], where)
		}
		for _, a := range c.Any {
			walk(a)
		}
	}
	walk(c)
}

// node enters the story at a node: an event run, a battle, a walk. It returns the vertex id, "" for none.
func (b *builder) node(id, ctx string) string {
	n := b.d.Node(id)
	if n.ID == "" {
		b.issue("error", "", fmt.Sprintf("노드 %q 가 없습니다", id))
		return ""
	}
	switch n.Kind {
	case "battle":
		vid := "battle:" + id
		b.visited[id] = true
		fresh := b.vertex(Vertex{ID: vid, Kind: KindBattle, Title: b.stageName(n.Stage), Sub: n.Stage + " · " + id})
		if fresh {
			b.battleSets(vid, n.Stage)
			for _, id := range stageJoins(b.d, n.Stage) {
				b.rosters[vid] = append(b.rosters[vid], content.Node{Kind: "join", Character: id})
			}
			b.battles++
			if next := b.d.After(id); b.battles < b.opt.Battles && next != "" {
				if to := b.node(next, ctx); to != "" {
					b.edge(vid, to, EdgeNext, "전투 뒤", nil)
				}
			} else if next != "" {
				b.stopped = true
			}
		}
		return vid
	case "explore":
		return b.explore(n)
	case "end":
		vid := "end:" + id
		b.visited[id] = true
		b.vertex(Vertex{ID: vid, Kind: KindEnd, Title: "끝", Sub: id})
		return vid
	}
	vid := "event:" + id
	if b.have[vid] {
		return vid
	}
	run := b.Walk(id)
	title := b.eventTitle(run)
	b.vertex(Vertex{ID: vid, Kind: KindEvent, Title: title, Sub: fmt.Sprintf("%s · %d줄", id, len(run.Nodes))})
	b.g.Issues = append(b.g.Issues, nodeLineIssues(b.d, b.env, vid, run)...)
	if ctx != "" {
		b.ctx[vid] = ctx
	}
	for _, rn := range run.Nodes {
		b.visited[rn.ID] = true
		if rn.Kind == "join" || rn.Kind == "leave" {
			b.rosters[vid] = append(b.rosters[vid], rn)
		}
		if rn.Kind == "set" {
			if b.setsOf[vid] == nil {
				b.setsOf[vid] = map[string]int{}
			}
			for k, v := range rn.Flags {
				b.sets[k] = append(b.sets[k], vid)
				b.setsOf[vid][k] = v
			}
		}
	}
	end := run.End
	if end.ID == "" {
		if len(run.Nodes) > 0 {
			b.issue("error", vid, fmt.Sprintf("%s 뒤에 이어지는 노드가 없습니다", run.Nodes[len(run.Nodes)-1].ID))
		}
		return vid
	}
	b.visited[end.ID] = true
	switch end.Kind {
	case "back":
		b.backs = append(b.backs, backRef{vid, end, b.ctx[vid]})
	case "choice":
		opts := make([]string, 0, len(end.Choices))
		for o := range end.Choices {
			opts = append(opts, o)
		}
		slices.Sort(opts)
		for _, o := range opts {
			if to := b.node(end.Choices[o], b.ctx[vid]); to != "" {
				b.edge(vid, to, EdgeCase, "선택: "+o, nil)
			}
		}
	case "branch":
		for _, c := range end.Cases {
			if to := b.node(c.Next, b.ctx[vid]); to != "" {
				b.edge(vid, to, EdgeCase, "", c.If)
			}
		}
		next := end.Else
		if next == "" {
			next = b.d.After(end.ID)
		}
		if to := b.node(next, b.ctx[vid]); to != "" {
			b.edge(vid, to, EdgeCase, "그 밖", nil)
		}
	default: // battle, explore, end
		if to := b.node(end.ID, b.ctx[vid]); to != "" {
			b.edge(vid, to, EdgeNext, "", nil)
		}
	}
	return vid
}

// Walk reads the run of nodes from id on: say, narrate, scene, set, join and the like are the run; back, branch,
// battle, explore and end stop it.
func (b *builder) Walk(id string) Run { return Walk(b.d, id) }

// Walk reads the run of story nodes from id.
func Walk(d *content.Data, id string) Run {
	run := Run{Start: id}
	seen := map[string]bool{}
	for id != "" && !seen[id] {
		seen[id] = true
		n := d.Node(id)
		if n.ID == "" {
			return run
		}
		switch n.Kind {
		case "back", "branch", "choice", "battle", "explore", "end":
			run.End = n
			return run
		}
		run.Nodes = append(run.Nodes, n)
		id = d.After(n.ID)
	}
	return run
}

// StageFlags are the flags a won battle stores: its events' flag actions, and the flag of each bonus objective (the
// stage's own and those a goals action puts in place), taken as achieved.
func StageFlags(s content.Stage) map[string]int {
	out := map[string]int{}
	for _, o := range s.Sub {
		if o.Flag != "" {
			out[o.Flag] = 1
		}
	}
	for _, ev := range s.Events {
		for _, a := range ev.Do {
			for k, v := range a.Flags {
				out[k] = v
			}
			for _, o := range a.Sub {
				if o.Flag != "" {
					out[o.Flag] = 1
				}
			}
		}
	}
	return out
}

func (b *builder) battleSets(vid, stage string) {
	i := b.d.StageIndex(stage)
	if i < 0 {
		return
	}
	for k, v := range StageFlags(b.d.Stages[i]) {
		if b.setsOf[vid] == nil {
			b.setsOf[vid] = map[string]int{}
		}
		b.sets[k] = append(b.sets[k], vid)
		b.setsOf[vid][k] = v
	}
}

func (b *builder) eventTitle(run Run) string {
	for _, n := range run.Nodes {
		if n.Kind == "scene" && n.Place != "" {
			return n.Place
		}
	}
	for _, n := range run.Nodes { // no scene: the first thing said
		if (n.Kind == "narrate" || n.Kind == "say") && n.Text != "" {
			r := []rune(n.Text)
			if len(r) > 16 {
				return string(r[:16]) + "…"
			}
			return string(r)
		}
	}
	return "이야기 " + run.Start
}

func (b *builder) stageName(id string) string {
	if i := b.d.StageIndex(id); i >= 0 {
		return b.d.Stages[i].Name
	}
	b.issue("error", "battle:"+id, fmt.Sprintf("스테이지 %q 가 없습니다", id))
	return id
}

// explore opens a walk: its start spot, the event that opens it, and every area reachable by exits.
func (b *builder) explore(n content.Node) string {
	vid := "explore:" + n.ID
	b.visited[n.ID] = true
	x := b.d.Explores[n.Explore]
	if x == nil {
		b.vertex(Vertex{ID: vid, Kind: KindExplore, Title: "탐험 " + n.Explore, Sub: n.ID})
		b.issue("error", vid, fmt.Sprintf("탐험 파일 %q 가 없습니다", n.Explore))
		return vid
	}
	if !b.vertex(Vertex{ID: vid, Kind: KindExplore, Title: "탐험 시작", Sub: n.Explore + " · " + n.ID}) {
		return vid
	}
	b.g.Issues = append(b.g.Issues, ExploreIssues(b.d, b.env, n.Explore, x)...)
	first := AreaID(n.Explore, x.Start.Area)
	b.areaLater(n, x, x.Start.Area)
	if n.Opening != "" {
		if ev := b.node(n.Opening, n.ID); ev != "" {
			b.edge(vid, ev, EdgeOpen, "시작 이벤트", nil)
			b.from[ev] = append(b.from[ev], first)
		}
	}
	b.edge(vid, first, EdgeStart, "", nil)
	return vid
}

func (b *builder) areaLater(n content.Node, x *content.Explore, name string) string {
	vid := AreaID(n.Explore, name)
	if b.have[vid] {
		return vid
	}
	a := x.Areas[name]
	if a == nil {
		b.issue("error", "", fmt.Sprintf("구역 %q 가 %s 에 없습니다", name, n.Explore))
		return ""
	}
	b.vertex(Vertex{ID: vid, Kind: KindArea, Title: a.Name, Sub: n.Explore + "/" + name + " · " + a.Map})
	b.queue = append(b.queue, func() { b.area(n, x, name, vid) })
	return vid
}

func (b *builder) area(n content.Node, x *content.Explore, name, vid string) {
	a := x.Areas[name]
	m := b.env.Map(a.Map)
	var w *scenemap.Walk
	if m == nil {
		b.issue("error", vid, fmt.Sprintf("지도 %q 가 없습니다", a.Map))
	} else {
		w = b.walkOf(a.Map, m)
	}
	for _, t := range a.Times {
		for _, c := range t.If {
			b.use(c, vid)
		}
	}
	for _, e := range a.Exits {
		if e.To != "" {
			if to := b.areaLater(n, x, e.To); to != "" {
				b.edge(vid, to, EdgeExit, e.Name, e.If)
			}
		}
		if e.Node != "" {
			if to := b.node(e.Node, n.ID); to != "" {
				b.edge(vid, to, EdgeGate, e.Name, e.If)
				b.from[to] = append(b.from[to], vid)
			}
		}
		if e.To == "" && e.Node == "" {
			b.issue("error", vid, fmt.Sprintf("출구 %q 가 어디로도 이어지지 않습니다", e.Name))
		}
		b.checkTile(vid, m, w, "출구 "+e.Name, e.At, true)
	}
	keys := make([]string, 0, len(a.People))
	for k := range a.People {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := a.People[k]
		who := content.PersonName(k)
		for _, c := range p.If {
			b.use(c, vid)
		}
		b.checkTile(vid, m, w, "사람 "+k, p.At, false)
		for _, ev := range p.Event {
			label := who
			if ev.Side {
				label += " (곁가지)"
			}
			if to := b.node(ev.Node, n.ID); to != "" {
				b.edge(vid, to, EdgeTalk, label, append(slices.Clone(p.If), ev.If...))
				b.g.Edges[len(b.g.Edges)-1].Side = ev.Side
				b.from[to] = append(b.from[to], vid)
			}
		}
		if p.Sortie != nil {
			next := b.d.After(n.ID)
			if next == "" {
				b.issue("error", vid, fmt.Sprintf("%s 의 출진 뒤에 이어지는 노드가 없습니다", who))
				continue
			}
			if to := b.node(next, n.ID); to != "" {
				b.edge(vid, to, EdgeSort, who+" 출진", append(slices.Clone(p.If), p.Sortie.If...))
			}
		}
		if p.Shop != "" {
			if _, ok := b.d.Shops[p.Shop]; !ok {
				b.issue("error", vid, fmt.Sprintf("%s 의 상점 %q 가 없습니다", who, p.Shop))
			}
		}
	}
	for _, s := range a.Signs {
		b.checkTile(vid, m, nil, "표지 "+s.Name, s.At, false)
	}
	b.checkTile(vid, m, w, "도착 칸", a.At, false)
}

func (b *builder) walkOf(id string, m *scenemap.Map) *scenemap.Walk {
	if w, ok := b.walk[id]; ok {
		return w
	}
	var w *scenemap.Walk
	if b.env.Catalog != nil {
		w = scenemap.Walkable(m, b.env.Catalog)
	}
	b.walk[id] = w
	return w
}

// checkTile warns about a place on the map that cannot be walked or stood on. An exit off the map is a way out.
func (b *builder) checkTile(vid string, m *scenemap.Map, w *scenemap.Walk, what string, at [2]int, exit bool) {
	switch {
	case m == nil:
	case !m.Inside(at[0], at[1]):
		if !exit {
			b.issue("warn", vid, fmt.Sprintf("%s 의 칸 %v 가 지도 밖입니다", what, at))
		}
	case w != nil && !w.Open(at[0], at[1]):
		b.issue("warn", vid, fmt.Sprintf("%s 의 칸 %v 는 걸을 수 없는 칸입니다", what, at))
	}
}

func (b *builder) drain() {
	for len(b.queue) > 0 {
		f := b.queue[0]
		b.queue = b.queue[1:]
		f()
	}
}

// spreadFrom hands the areas that start an event on to the events that follow it, so a back node after a branch
// still knows where to return.
func (b *builder) spreadFrom() {
	for changed := true; changed; {
		changed = false
		for _, e := range b.g.Edges {
			if e.Kind != EdgeNext && e.Kind != EdgeCase {
				continue
			}
			for _, a := range b.from[e.From] {
				if !slices.Contains(b.from[e.To], a) {
					b.from[e.To] = append(b.from[e.To], a)
					changed = true
				}
			}
		}
	}
}

// resolveBacks gives every back node its edge. A back that names an area opens that area if no exit reached it, and
// the people and exits there are followed in turn.
func (b *builder) resolveBacks() {
	b.spreadFrom()
	for i := 0; i < len(b.backs); i++ {
		r := b.backs[i]
		file := ""
		if r.explore != "" {
			file = b.d.Node(r.explore).Explore
		}
		switch {
		case r.node.Leave:
			next := ""
			if r.explore != "" {
				next = b.d.After(r.explore)
			}
			if next == "" {
				b.issue("error", r.vertex, "탐험을 끝내는 back 뒤에 이어지는 노드가 없습니다")
				continue
			}
			if to := b.node(next, ""); to != "" {
				b.edge(r.vertex, to, EdgeBack, "탐험 종료", nil)
			}
		case r.node.Area != "":
			x := b.d.Explores[file]
			if x == nil || x.Areas[r.node.Area] == nil {
				b.issue("error", r.vertex, fmt.Sprintf("back 이 가리키는 구역 %q 가 없습니다", r.node.Area))
				continue
			}
			to := b.areaLater(b.d.Node(r.explore), x, r.node.Area)
			b.edge(r.vertex, to, EdgeBack, "복귀", nil)
		default:
			srcs := b.from[r.vertex]
			if len(srcs) == 0 {
				b.issue("warn", r.vertex, "back 이 돌아갈 구역을 알 수 없습니다 (말을 건 구역이 없음)")
			}
			for _, s := range slices.Compact(slices.Sorted(slices.Values(srcs))) {
				b.edge(r.vertex, s, EdgeBack, "복귀", nil)
			}
		}
		b.drain()
		b.spreadFrom()
	}
}

func (b *builder) finish() {
	names := map[string]bool{}
	for k := range b.sets {
		names[k] = true
	}
	for k := range b.uses {
		names[k] = true
	}
	for _, k := range slices.Sorted(func(yield func(string) bool) {
		for k := range names {
			if !yield(k) {
				return
			}
		}
	}) {
		u := FlagUse{Name: k, SetBy: uniq(b.sets[k]), UsedBy: uniq(b.uses[k])}
		if slices.Contains(b.opt.Assume, k) {
			u.SetBy = append(u.SetBy, "(앞 장에서 세운 것으로 가정)")
		}
		b.g.Flags = append(b.g.Flags, u)
		switch {
		case len(u.UsedBy) > 0 && len(u.SetBy) == 0:
			b.issue("error", u.UsedBy[0], fmt.Sprintf("플래그 %q 를 읽지만 어디서도 세우지 않습니다", k))
		case len(u.SetBy) > 0 && len(u.UsedBy) == 0:
			b.issue("warn", u.SetBy[0], fmt.Sprintf("플래그 %q 를 세우지만 읽는 곳이 없습니다", k))
		}
	}
	b.unreached()
}

func uniq(s []string) []string {
	s = slices.Clone(s)
	slices.Sort(s)
	return append([]string{}, slices.Compact(s)...)
}

// unreached lists the nodes of the chapters the graph touches that no walk of it came to.
func (b *builder) unreached() {
	if b.stopped {
		return
	}
	for _, c := range b.d.Chapters {
		touched := false
		for _, id := range append(slices.Clone(c.Nodes), c.Events...) {
			touched = touched || b.visited[id]
		}
		if !touched {
			continue
		}
		var miss []string
		for _, id := range c.Events {
			if !b.visited[id] && b.d.Node(id).Kind != "back" {
				miss = append(miss, id)
			}
		}
		if len(miss) > 0 {
			shown := strings.Join(miss[:min(len(miss), 4)], ", ")
			b.issue("warn", "", fmt.Sprintf("%s장 이벤트 노드 %d개에 도달하지 못했습니다 (%s …)", c.ID, len(miss), shown))
		}
	}
}

func walkOf(env Env, m *scenemap.Map) *scenemap.Walk {
	if env.Catalog == nil {
		return nil
	}
	return scenemap.Walkable(m, env.Catalog)
}

// uniqIssues drops the repeats of an issue (a speaker without a portrait who speaks on every page), keeping the order.
func uniqIssues(is []Issue) []Issue {
	seen := map[Issue]bool{}
	out := is[:0]
	for _, i := range is {
		if !seen[i] {
			seen[i] = true
			out = append(out, i)
		}
	}
	return out
}
