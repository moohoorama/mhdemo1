package scenemap

import (
	"container/heap"
	"image"
	"sort"
)

// Key is what the standing layer is sorted by: the ground an item covers (Box: u0, v0, u1, v1 in small tiles), how
// high it stands (Lift, canvas pixels), its kind (Rank) and the screen area it covers.
type Key struct {
	Box  [4]float64      `json:"box"`
	Lift int             `json:"lift"`
	Rank int             `json:"rank"`
	X    float64         `json:"-"` // the anchor's screen x
	Rect image.Rectangle `json:"-"`
}

// Ranks of the kinds of standing items, for items on the same ground: a block's faces, then walls and gates, then
// objects, then people.
const (
	RankFace = iota
	RankWall
	RankObject
	RankPerson
)

func (k Key) front() float64 { return k.Box[2] + k.Box[3] }

func (k Key) wide() bool { return k.Box[2]-k.Box[0] > 1 || k.Box[3]-k.Box[1] > 1 }

// before is the plain order: by the front corner of the footprint (u1 + v1), then height, kind, screen x.
func (k Key) before(o Key) bool {
	if k.front() != o.front() {
		return k.front() < o.front()
	}
	if k.Lift != o.Lift {
		return k.Lift < o.Lift
	}
	if k.Rank != o.Rank {
		return k.Rank < o.Rank
	}
	return k.X < o.X
}

// behind reports whether k's footprint lies wholly behind o's: on the smaller side along u or along v, and not
// wholly in front of it along the other axis.
func (k Key) behind(o Key) bool {
	byU := k.Box[2] <= o.Box[0] && !(o.Box[3] <= k.Box[1])
	byV := k.Box[3] <= o.Box[1] && !(o.Box[2] <= k.Box[0])
	return byU || byV
}

// Order is the drawing order of keys: the plain order, except that an item covering more than one small tile is
// drawn after every item it overlaps on screen whose footprint lies wholly behind its own, and before every one
// lying wholly in front.
func Order(keys []Key) []int {
	plain := make([]int, len(keys))
	for i := range plain {
		plain[i] = i
	}
	sort.SliceStable(plain, func(a, b int) bool { return keys[plain[a]].before(keys[plain[b]]) })
	rank := make([]int, len(keys)) // position in the plain order
	for p, i := range plain {
		rank[i] = p
	}
	after := make([][]int, len(keys)) // after[i]: items that must come after i
	needs := make([]int, len(keys))
	for i, k := range keys {
		if !k.wide() {
			continue
		}
		for j, o := range keys {
			if i == j || !k.Rect.Overlaps(o.Rect) || o.wide() && j < i {
				continue // a pair of wide items is looked at once
			}
			switch {
			case k.behind(o) && !o.behind(k):
				after[i] = append(after[i], j)
				needs[j]++
			case o.behind(k) && !k.behind(o):
				after[j] = append(after[j], i)
				needs[i]++
			}
		}
	}
	ready := &byRank{rank: rank}
	for i := range keys {
		if needs[i] == 0 {
			heap.Push(ready, i)
		}
	}
	out := make([]int, 0, len(keys))
	done := make([]bool, len(keys))
	next := 0 // the plain order breaks a cycle: the first item not drawn yet goes next
	for len(out) < len(keys) {
		var i int
		if ready.Len() > 0 {
			i = heap.Pop(ready).(int)
		} else {
			for done[plain[next]] {
				next++
			}
			i = plain[next]
		}
		if done[i] {
			continue
		}
		done[i] = true
		out = append(out, i)
		for _, j := range after[i] {
			if needs[j]--; needs[j] == 0 && !done[j] {
				heap.Push(ready, j)
			}
		}
	}
	return out
}

type byRank struct {
	items []int
	rank  []int
}

func (h byRank) Len() int           { return len(h.items) }
func (h byRank) Less(a, b int) bool { return h.rank[h.items[a]] < h.rank[h.items[b]] }
func (h byRank) Swap(a, b int)      { h.items[a], h.items[b] = h.items[b], h.items[a] }
func (h *byRank) Push(x any)        { h.items = append(h.items, x.(int)) }
func (h *byRank) Pop() any {
	n := len(h.items) - 1
	x := h.items[n]
	h.items = h.items[:n]
	return x
}
