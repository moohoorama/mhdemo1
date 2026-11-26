package content

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stockData is a story of walks a1 and a2 (chapter 1), b (2), c (3), then two tellings of the ending: t1 (4) and t2 (5).
func stockData() *Data {
	walk := func(shops ...string) *Explore {
		people := map[string]*ExplorePerson{}
		for i, s := range shops {
			people[string(rune('a'+i))] = &ExplorePerson{Shop: s}
		}
		return &Explore{Areas: map[string]*ExploreArea{"town": {People: people}}}
	}
	d := &Data{
		Chapters: []Chapter{
			{ID: "1", Nodes: []string{"a1", "a2"}},
			{ID: "2", Nodes: []string{"b"}},
			{ID: "3", Nodes: []string{"c"}},
			{ID: "4", Act: "2막-연의", Nodes: []string{"t1"}},
			{ID: "5", Act: "2막-가상", Nodes: []string{"t2"}},
		},
		Nodes: []Node{
			{ID: "a1", Kind: "explore", Explore: "xa1"}, {ID: "a2", Kind: "explore", Explore: "xa2"},
			{ID: "b", Kind: "explore", Explore: "xb"}, {ID: "c", Kind: "explore", Explore: "xc"},
			{ID: "t1", Kind: "explore", Explore: "xt1"}, {ID: "t2", Kind: "explore", Explore: "xt2"},
		},
		Explores: map[string]*Explore{
			"xa1": walk("base"), "xa2": walk("base"),
			"xb": walk("base_plus"), "xc": walk("base_plus", "later"),
			"xt1": walk("tell1"), "xt2": walk("tell2"),
		},
		Shops: map[string]Shop{
			"base":      {Items: []string{"rice", "tea"}},
			"base_plus": {Items: []string{"rice", "tea", "wine"}},
			"later":     {Items: []string{"rice", "wine", "silk"}},
			"tell1":     {Items: []string{"rice", "gem", "jade"}},
			"tell2":     {Items: []string{"rice", "gem", "jade"}},
		},
	}
	return d
}

func TestNewStockIsWhatNoEarlierWalkSold(t *testing.T) {
	d := stockData()
	cases := []struct {
		name, file, shop string
		want             []string
	}{
		{"the opening stock is not news", "xa1", "base", nil},
		{"the first chapter's later walk is no news either", "xa2", "base", nil},
		{"a good first sold in chapter 2", "xb", "base_plus", []string{"wine"}},
		{"the same shop again is not news", "xc", "base_plus", nil},
		{"another shop with an old and a new good", "xc", "later", []string{"silk"}},
		{"a telling of the ending", "xt1", "tell1", []string{"gem", "jade"}},
		{"the other telling does not count the first as earlier", "xt2", "tell2", []string{"gem", "jade"}},
		{"a walk no chapter opens", "nowhere", "base_plus", nil},
	}
	d.Explores["nowhere"] = d.Explores["xb"]
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, d.NewStock(c.file, c.shop))
		})
	}
}

func TestNewStockOfAWalkWithTwoShopsOfNewGoods(t *testing.T) {
	d := stockData()
	d.Explores["xc"].Areas["town"].People["c"] = &ExplorePerson{Shop: "tell1"}
	assert.Equal(t, []string{"gem", "jade"}, d.NewStock("xc", "tell1"))
	assert.Equal(t, []string{"silk"}, d.NewStock("xc", "later"), "a good both of the walk's shops sell is new in both")
}

func TestNewStockNodeOrderInAChapter(t *testing.T) {
	d := stockData()
	d.Chapters[1].Nodes = []string{"b", "c"}
	d.Chapters[2].Nodes = nil
	assert.Equal(t, []string{"wine"}, d.NewStock("xb", "base_plus"))
	assert.Equal(t, []string{"silk"}, d.NewStock("xc", "later"), "wine was sold by the walk before it in the chapter")
}

func TestNewStockInTheRealStory(t *testing.T) {
	d, err := LoadScenario("../../assets/data", "hero")
	require.NoError(t, err)
	got := map[string][]string{}
	for file := range d.Explores {
		for _, a := range d.Explores[file].Areas {
			for _, p := range a.People {
				if p.Shop != "" {
					if n := d.NewStock(file, p.Shop); len(n) > 0 {
						got[file+"/"+p.Shop] = n
					}
				}
			}
		}
	}
	var keys []string
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := map[string][]string{
		"explore03/luoyang_tavern":  {"bean", "barley", "liquor", "fine_liquor"},
		"explore08/tools_cure":      {"cure_all"},
		"explore14/xuzhou":          {"insu_mid"},
		"explore14/xuzhou_tavern":   {"baijiu"},
		"explore19_ye/weapons_iron": {"iron_sword", "iron_spear", "long_bow", "leather_gloves", "bronze_armor", "leather_clothes"},
	}
	assert.Equal(t, want, got, "every first sale of the story; keys: %v", keys)
}
