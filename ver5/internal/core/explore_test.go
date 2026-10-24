package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// walkEngine stands at a walk of two areas: a street with a man who starts the oath event, a shopkeeper and the
// confidant (who sorties only after the oath), and a grove behind an exit that opens after the oath.
func walkEngine(t *testing.T) *Engine {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	flow := []content.Node{
		{ID: "walk", Kind: "explore", Explore: "explore01"},
		{ID: "pre", Kind: "narrate", Text: "off we go"},
		{ID: "fight", Kind: "battle", Stage: "S1"},
	}
	events := []content.Node{
		{ID: "oath", Kind: "scene", Place: "숲"},
		{ID: "oath-1", Kind: "say", Speaker: "hero", Text: "brothers"},
		{ID: "oath-2", Kind: "set", Flags: map[string]int{"oath": 1}},
		{ID: "oath-3", Kind: "grant", Item: "money", Count: 50, Event: "oath-money"},
		{ID: "oath-4", Kind: "back", Area: "grove", At: &[2]int{3, 4}, Face: "SW"},
	}
	ids := func(ns []content.Node) (out []string) {
		for _, n := range ns {
			out = append(out, n.ID)
		}
		return
	}
	d.Nodes = append(append([]content.Node{}, flow...), events...)
	d.Chapters = []content.Chapter{{ID: "1", File: "x.json", Nodes: ids(flow), Events: ids(events)}}
	d.Campaign.Start, d.Campaign.Money = "walk", 100
	tonic, spear := d.Items["tonic"], d.Equipment["spear"]
	tonic.Price, spear.Price = 30, 80
	d.Items["tonic"], d.Equipment["spear"] = tonic, spear
	d.Shops = map[string]content.Shop{"tools": {Name: "도구점", Items: []string{"tonic", "spear"}}}
	d.Explores = map[string]*content.Explore{"explore01": {
		Hero:  "hero",
		Start: content.Spot{Area: "street", At: [2]int{5, 5}, Face: "SE"},
		Areas: map[string]*content.ExploreArea{
			"street": {Map: "S_a", Name: "거리", At: [2]int{1, 1}, Exits: []content.ExploreExit{{Name: "숲", At: [2]int{0, 9}, To: "grove", If: []string{"flag:oath"}}},
				People: map[string]*content.ExplorePerson{
					"장비":       {Face: "SW", Event: []content.ExploreEvent{{If: []string{"!flag:oath"}, Node: "oath"}}, Talk: []content.Line{{"장비", "가유"}}},
					"관우":       {Face: "SW", Sortie: &content.ExploreSortie{If: []string{"flag:oath"}, Ask: "출진?"}},
					"상인":       {Face: "SW", Shop: "tools"},
					"장비@after": {Face: "SW", If: []string{"flag:oath"}},
				}},
			"grove": {Map: "S_b", Name: "숲", At: [2]int{2, 2}, Exits: []content.ExploreExit{{Name: "거리", At: [2]int{9, 0}, To: "street"}}},
		},
	}}
	require.NoError(t, d.Validate())
	return New(d, 1)
}

func visitors(e *Engine) map[string]string {
	out := map[string]string{}
	for _, v := range e.Visitors() {
		out[v.Key] = v.Do
	}
	return out
}

func TestWalkTalkEventAndSortie(t *testing.T) {
	e := walkEngine(t)
	o := e.Observe()
	require.Equal(t, "walk", o.Node)
	assert.Equal(t, content.Spot{Area: "street", At: [2]int{5, 5}, Face: "SE"}, o.Spot)
	assert.Equal(t, 100, o.Money)
	assert.Equal(t, map[string]string{"장비": "event", "관우": "talk", "상인": "shop"}, visitors(e))

	t.Run("a shut exit and a person without an event refuse", func(t *testing.T) {
		_, err := e.Apply(Command{Kind: "walk", Option: "grove", Target: "SE"})
		assert.EqualError(t, err, "ExitShut")
		_, err = e.Apply(Command{Kind: "sortie", Target: "관우"})
		assert.EqualError(t, err, "NoSortie")
		_, err = e.Apply(Command{Kind: "talk", Target: "관우"})
		assert.EqualError(t, err, "NothingToSay")
	})

	applyOK(t, e, Command{Kind: "walk", Option: "street", X: 7, Y: 2, Target: "NW"})
	assert.Equal(t, [2]int{7, 2}, e.Observe().Spot.At)
	applyOK(t, e, Command{Kind: "talk", Target: "장비"})
	o = e.Observe()
	assert.Equal(t, "oath-1", o.Node)
	assert.Equal(t, "숲", o.Place)
	assert.Nil(t, e.Visitors(), "nobody is talked to during an event")
	next(t, e)
	o = e.Observe()
	assert.Equal(t, "walk", o.Node, "back returns to the walk")
	assert.Equal(t, content.Spot{Area: "grove", At: [2]int{3, 4}, Face: "SW"}, o.Spot)
	assert.Equal(t, 150, o.Money)

	applyOK(t, e, Command{Kind: "walk", Option: "street", X: 0, Y: 9, Target: "NW"})
	assert.Equal(t, map[string]string{"장비": "talk", "관우": "sortie", "상인": "shop", "장비@after": "talk"}, visitors(e))
	applyOK(t, e, Command{Kind: "sortie", Target: "관우"})
	o = e.Observe()
	assert.Equal(t, "pre", o.Node)
	assert.Empty(t, o.Explore)
}

func TestShopBuysAndSellsForHalf(t *testing.T) {
	e := walkEngine(t)
	applyOK(t, e, Command{Kind: "buy", Target: "상인", Item: "tonic", Count: 2})
	o := e.Observe()
	assert.Equal(t, 40, o.Money)
	assert.Equal(t, 4, o.Inventory["tonic"])
	_, err := e.Apply(Command{Kind: "buy", Target: "상인", Item: "spear", Count: 1})
	assert.EqualError(t, err, "NotEnoughMoney")
	_, err = e.Apply(Command{Kind: "buy", Target: "상인", Item: "ration", Count: 1})
	assert.EqualError(t, err, "InvalidItem", "an item without a price is not sold")
	_, err = e.Apply(Command{Kind: "buy", Target: "장비", Item: "tonic", Count: 1})
	assert.EqualError(t, err, "NoShop")

	applyOK(t, e, Command{Kind: "sell", Target: "상인", Item: "tonic", Count: 4})
	o = e.Observe()
	assert.Equal(t, 100, o.Money)
	assert.Zero(t, o.Inventory["tonic"])
	_, err = e.Apply(Command{Kind: "sell", Target: "상인", Item: "tonic", Count: 1})
	assert.EqualError(t, err, "NotOwned")

	applyOK(t, e, Command{Kind: "buy", Target: "상인", Item: "spear", Count: 1})
	assert.Equal(t, 1, e.Observe().Warehouse["spear"], "bought equipment goes to the warehouse")
}

func TestWalkSavesAndRestores(t *testing.T) {
	e := walkEngine(t)
	applyOK(t, e, Command{Kind: "walk", Option: "street", X: 3, Y: 8, Target: "NE"})
	r, err := Restore(e.Data(), e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, e.Observe().Spot, r.Observe().Spot)

	applyOK(t, e, Command{Kind: "talk", Target: "장비"})
	_, err = Restore(e.Data(), e.Snapshot())
	require.NoError(t, err, "a save inside an event restores")

	s := e.Snapshot()
	s.Spot.Area = "nowhere"
	_, err = Restore(e.Data(), s)
	assert.EqualError(t, err, "CorruptSave")
}

func TestExploreNextFindsTheEventThenTheSortie(t *testing.T) {
	e := walkEngine(t)
	applyOK(t, e, Command{Kind: "unequip", Actor: "hero", Item: "jerkin"})
	for step := 0; step < 20 && e.Walking() || e.Observe().Dialogue.Kind == "say"; step++ {
		if !e.Walking() {
			next(t, e)
			continue
		}
		plan := e.ExploreNext()
		require.NotEmpty(t, plan)
		for _, c := range plan {
			applyOK(t, e, c)
		}
	}
	assert.Equal(t, "pre", e.Observe().Node)
}
