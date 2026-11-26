package content

import "slices"

// walkRef is where a walk (an explore file) stands in the story: its chapter, and its place in that chapter's nodes.
type walkRef struct {
	file           string
	chapter, order int
}

// walkRefs lists the walks the chapters' explore nodes open, in story order.
func (d *Data) walkRefs() []walkRef {
	var out []walkRef
	for ci, c := range d.Chapters {
		for oi, id := range c.Nodes {
			if n := d.Node(id); n.Kind == "explore" {
				out = append(out, walkRef{file: n.Explore, chapter: ci, order: oi})
			}
		}
	}
	return out
}

// isTelling reports whether an act is one of the two tellings that close the second act; a chapter of one telling
// never comes before a chapter of the other, since the player plays only one of them.
func isTelling(act string) bool { return act == "2막-연의" || act == "2막-가상" }

// before reports whether the player can have walked a before walking b: an earlier chapter, or an earlier node of
// the same chapter, of the same telling.
func (d *Data) before(a, b walkRef) bool {
	if a.chapter == b.chapter {
		return a.order < b.order
	}
	if a.chapter > b.chapter {
		return false
	}
	ca, cb := d.Chapters[a.chapter], d.Chapters[b.chapter]
	return !(isTelling(ca.Act) && isTelling(cb.Act) && ca.Act != cb.Act)
}

// soldIn is the equipment and item ids the shops of a walk sell, in any of its areas.
func (d *Data) soldIn(file string) map[string]bool {
	out := map[string]bool{}
	if x := d.Explores[file]; x != nil {
		for _, a := range x.Areas {
			for _, p := range a.People {
				if p.Shop != "" {
					for _, id := range d.Shops[p.Shop].Items {
						out[id] = true
					}
				}
			}
		}
	}
	return out
}

// NewStock is what shop sells in the walk of explore file that no walk the player can have taken before it sold:
// the goods a crier near the shop's door tells of. Nothing is new in the walks of the first chapter, which hold the
// opening stock, and a walk that no chapter opens has nothing new. The ids come in the shop's order.
func (d *Data) NewStock(file, shop string) []string {
	if len(d.Chapters) == 0 {
		return nil
	}
	walks := d.walkRefs()
	at := slices.IndexFunc(walks, func(w walkRef) bool { return w.file == file })
	if at < 0 || walks[at].chapter == 0 {
		return nil
	}
	sold := map[string]bool{}
	for _, w := range walks {
		if w.file != file && d.before(w, walks[at]) {
			for id := range d.soldIn(w.file) {
				sold[id] = true
			}
		}
	}
	var out []string
	for _, id := range d.Shops[shop].Items {
		if !sold[id] {
			out = append(out, id)
		}
	}
	return out
}

// StockName is the name an equipment or item id is shown by.
func (d *Data) StockName(id string) string {
	if e, ok := d.Equipment[id]; ok {
		return e.Name
	}
	return d.Items[id].Name
}
