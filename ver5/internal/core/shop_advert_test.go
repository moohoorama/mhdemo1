package core

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// walkNodeOf is the story node whose walk opens explore file.
func walkNodeOf(d *content.Data, file string) string {
	for _, n := range d.Nodes {
		if n.Kind == "explore" && n.Explore == file {
			return n.ID
		}
	}
	return ""
}

// walkAt starts a game at the start of the chapter that has the walk of explore file and plays the story on to it.
func walkAt(t *testing.T, d *content.Data, file string) *Engine {
	t.Helper()
	node := walkNodeOf(d, file)
	start := d.Chapters[d.ChapterIndex(node)].Nodes[0]
	e, err := NewAt(d, 1, start, "normal")
	require.NoError(t, err)
	for i := 0; i < 80 && !(e.Walking() && e.state.Explore == node); i++ {
		for _, id := range e.Observe().Learning {
			applyOK(t, e, Command{Kind: "learn_close", Actor: id})
		}
		next(t, e)
	}
	require.True(t, e.Walking() && e.state.Explore == node, "the story reaches %s", node)
	return e
}

// areaPath is the areas to pass from one area of a walk to another through its exits, the first being from.
func areaPath(x *content.Explore, from, to string) []string {
	prev := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		if at == to {
			var path []string
			for a := to; a != ""; a = prev[a] {
				path = append([]string{a}, path...)
			}
			return path
		}
		for _, e := range x.Areas[at].Exits {
			if _, seen := prev[e.To]; e.To != "" && !seen {
				prev[e.To] = at
				queue = append(queue, e.To)
			}
		}
	}
	return nil
}

// walkTo takes the hero through the exits to area.
func walkTo(t *testing.T, e *Engine, area string) {
	t.Helper()
	x := e.ExploreFile()
	path := areaPath(x, e.Observe().Spot.Area, area)
	require.NotEmpty(t, path, "no way to %s", area)
	for i := 1; i < len(path); i++ {
		at := x.Arrival(path[i], path[i-1])
		applyOK(t, e, Command{Kind: "walk", Option: path[i], X: at.At[0], Y: at.At[1], Target: at.Face})
	}
}

// cryingShop is a shop with new stock in a walk, who tells of it, and where.
type cryingShop struct{ file, shop, area, room string }

func cryingShops(d *content.Data) []cryingShop {
	var out []cryingShop
	files := make([]string, 0, len(d.Explores))
	for f := range d.Explores {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		for name, a := range d.Explores[f].Areas {
			for _, p := range a.People {
				if shop, ok := strings.CutPrefix(p.Notice, "shop:"); ok {
					out = append(out, cryingShop{file: f, shop: shop, area: name})
				}
			}
		}
	}
	for i, c := range out {
		for name, a := range d.Explores[c.file].Areas {
			for _, p := range a.People {
				if p.Shop == c.shop {
					out[i].room = name
				}
			}
		}
	}
	return out
}

// TestNewStockIsToldOutdoorsAndSoldInside plays the real story's walks: where a shop sells goods no earlier walk
// sold, the hero finds someone outdoors near its door whose bubbles name every new good, who answers when talked to
// (explaining what each one is for, with no price or amount that a rebalance would make wrong), and the shop, entered, sells each of them.
func TestNewStockIsToldOutdoorsAndSoldInside(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)

	want := map[string][]string{
		"explore03/luoyang_tavern":  {"콩", "보리", "술", "특급주"},
		"explore08/tools_cure":      {"만능약"},
		"explore14/xuzhou":          {"중급 인수"},
		"explore14/xuzhou_tavern":   {"빼갈"},
		"explore19_ye/weapons_iron": {"철검", "철창", "대궁", "가죽장갑", "구리갑옷", "가죽옷"},
	}
	var seen []string
	for _, c := range cryingShops(d) {
		key := c.file + "/" + c.shop
		seen = append(seen, key)
		names, ok := want[key]
		require.True(t, ok, "%s tells of a shop that has nothing new in this walk", key)
		t.Run(key, func(t *testing.T) {
			stock := d.NewStock(c.file, c.shop)
			require.Len(t, stock, len(names))
			for i, id := range stock {
				assert.Equal(t, names[i], d.StockName(id))
			}

			e := walkAt(t, d, c.file)

			walkTo(t, e, c.area)
			street := e.ExploreFile().Areas[c.area]
			var crier string
			for k, p := range street.People {
				if p.Notice == "shop:"+c.shop {
					crier = k
				}
			}
			require.NotEmpty(t, crier)
			assert.Equal(t, "talk", visitors(e)[crier], "the crier is there to be talked to")
			_, err = e.Apply(Command{Kind: "talk", Target: crier})
			assert.EqualError(t, err, "NothingToSay", "what the crier says is the walk's talk, not an event")

			p := street.People[crier]
			var said []string
			for _, l := range p.Talk {
				said = append(said, l[1])
			}
			for _, id := range stock {
				name := d.StockName(id)
				assert.Contains(t, strings.ReplaceAll(p.Chatter, " ", ""), strings.ReplaceAll(name, " ", ""), "bubbles name %s", name)
				i := slices.IndexFunc(said, func(s string) bool {
					return strings.Contains(strings.ReplaceAll(s, " ", ""), strings.ReplaceAll(name, " ", ""))
				})
				require.GreaterOrEqual(t, i, 0, "talking explains %s", name)
			}

			for _, text := range append(slices.Clone(said), strings.Split(p.Chatter, "|")...) {
				assert.NotRegexp(t, `[0-9０-９]`, text, "a crier's words state no number")
			}

			walkTo(t, e, c.room)
			var keeper string
			for k, p := range e.ExploreFile().Areas[c.room].People {
				if p.Shop == c.shop {
					keeper = k
				}
			}
			assert.Equal(t, "shop", visitors(e)[keeper])
			e.state.Money = 1_000_000
			for _, id := range stock {
				applyOK(t, e, Command{Kind: "buy", Target: keeper, Item: id, Count: 1})
			}
		})
	}
	var wantKeys []string
	for k := range want {
		wantKeys = append(wantKeys, k)
	}
	sort.Strings(wantKeys)
	sort.Strings(seen)
	assert.Equal(t, wantKeys, seen, "every first sale has its crier, and nothing else has")
}

// TestAShopSeenAgainIsNotCriedAgain: Xuzhou is walked again in chapter 18 with the same shops, whose goods the
// chapter 14 walk sold already.
func TestAShopSeenAgainIsNotCriedAgain(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	assert.NotEmpty(t, d.NewStock("explore14", "xuzhou"))
	assert.Empty(t, d.NewStock("explore18", "xuzhou"))
	assert.Empty(t, d.NewStock("explore18", "xuzhou_tavern"))
	for _, a := range d.Explores["explore18"].Areas {
		for k, p := range a.People {
			assert.False(t, strings.HasPrefix(p.Notice, "shop:"), "%s cries goods the walk before sold", k)
		}
	}
}
