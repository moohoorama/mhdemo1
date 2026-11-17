package autoplay

import (
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srpg/internal/content"
	"srpg/internal/core"
)

func TestLearningSavesForPreferredTrait(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	g := DefaultGenome()
	e, err := core.NewAt(d, 1, "c01-battle", "hard")
	require.NoError(t, err)
	snap := e.Snapshot()
	snap.Warehouse = map[string]int{}
	for i := range snap.Characters {
		if snap.Characters[i].Def == "guanyu" {
			snap.Characters[i].Level = 10
			snap.Characters[i].Learned = nil
			snap.Characters[i].Equipment = map[string]string{}
		}
	}
	for _, points := range []int{500, 700, 1000} {
		t.Run(strconv.Itoa(points), func(t *testing.T) {
			for i := range snap.Characters {
				if snap.Characters[i].Def == "guanyu" {
					snap.Characters[i].Points = points
				}
			}
			e, err := core.Restore(d, snap)
			require.NoError(t, err)
			got := g.bestTrait(d, e, "guanyu")
			if points < 1000 {
				assert.Empty(t, got, "save rather than buy a cheaper movement trait")
			} else {
				assert.Equal(t, "opening", got)
			}
		})
	}
	t.Run("the immediate-buy genome can still learn a cheaper trait", func(t *testing.T) {
		g[gReserve] = 0
		for i := range snap.Characters {
			if snap.Characters[i].ID == "guanyu" {
				snap.Characters[i].Points = 500
			}
		}
		e, err := core.Restore(d, snap)
		require.NoError(t, err)
		trait := g.bestTrait(d, e, "guanyu")
		require.NotEmpty(t, trait)
		assert.LessOrEqual(t, d.Traits[trait].Cost, 500)
	})
}

func TestExplorationBuysOnlyAtReachableShops(t *testing.T) {
	for _, open := range []bool{false, true} {
		t.Run(map[bool]string{false: "closed", true: "open"}[open], func(t *testing.T) {
			d, err := content.Load("../testdata/mini")
			require.NoError(t, err)
			d.Nodes = append([]content.Node{{ID: "walk", Kind: "explore", Explore: "test-walk"}}, d.Nodes...)
			d.Campaign.Start, d.Campaign.Money = "walk", 1000
			d.Campaign.Inventory = map[string]int{}
			d.Items["ration"] = content.Item{Effect: "hp", Amount: 100, Price: 100}
			d.Shops = map[string]content.Shop{"food": {Items: []string{"ration"}, Rate: 75}}
			d.Explores = map[string]*content.Explore{"test-walk": {
				Start: content.Spot{Area: "street", Face: "SE"},
				Areas: map[string]*content.ExploreArea{
					"street": {Exits: []content.ExploreExit{{To: "shop", If: []string{"flag:open"}}}},
					"shop":   {People: map[string]*content.ExplorePerson{"seller": {Shop: "food"}}},
				},
			}}
			e := core.New(d, 1)
			if open {
				snap := e.Snapshot()
				snap.Flags["open"] = 1
				e, err = core.Restore(d, snap)
				require.NoError(t, err)
			}
			g := DefaultGenome()
			commands := g.Explore(d, e)
			if !open {
				assert.Empty(t, commands)
				return
			}
			require.Len(t, commands, 2)
			assert.Equal(t, "walk", commands[0].Kind)
			buy := commands[1]
			assert.Equal(t, "buy", buy.Kind)
			assert.Equal(t, "seller", buy.Target)
			for _, c := range commands {
				_, err := e.Apply(c)
				require.NoError(t, err)
			}
			assert.Equal(t, buy.Count, e.Observe().Inventory["ration"])
			assert.Equal(t, 1000-75*buy.Count, e.Observe().Money)
			assert.Empty(t, g.Explore(d, e), "once stocked the AI stops shopping")
		})
	}
}

func TestExplorationBuysAPromotionWithoutAlreadyOwningASeal(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := core.NewAt(d, 1, "c12-battle", "hard")
	require.NoError(t, err)
	snap := e.Snapshot()
	snap.Inventory = map[string]int{}
	snap.Money = 2000
	for i := range snap.Characters {
		if snap.Characters[i].ID == "liubei" {
			snap.Characters[i].Class = "light_infantry"
			snap.Characters[i].Level = 20
		}
	}
	e, err = core.Restore(d, snap)
	require.NoError(t, err)
	g := DefaultGenome()
	c, score := g.purchase(d, e, "xuzhou")
	assert.Positive(t, score)
	assert.Equal(t, "insu_mid", c.Item)
	assert.Equal(t, 1, c.Count)
}

func TestSortieReplacesABenchedOfficerButKeepsRequiredOfficers(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := StartAt(d, "c17-battle", 1, "hard", DefaultGenome())
	require.NoError(t, err)
	snap := e.Snapshot()
	bench := ""
	for i := range snap.Characters {
		c := &snap.Characters[i]
		if !c.Temp && d.Characters[c.Def].Playable && !slices.Contains(snap.Deployment, c.ID) {
			bench = c.ID
			c.Level = 50
			break
		}
	}
	require.NotEmpty(t, bench, "fixture has a benched officer")
	e, err = core.Restore(d, snap)
	require.NoError(t, err)
	g := DefaultGenome()
	g.sortie(d, e, func(c core.Command) bool {
		_, err := e.Apply(c)
		require.NoError(t, err)
		return true
	})
	assert.Contains(t, e.Observe().Deployment, bench)
	assert.Contains(t, e.Observe().Deployment, "liubei")
	assert.Contains(t, e.Observe().Deployment, "zhangfei")
	assert.Len(t, e.Observe().Deployment, d.Stages[snap.Stage].Max)
}
