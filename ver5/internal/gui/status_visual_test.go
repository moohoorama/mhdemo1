package gui

import (
	"encoding/json"
	"os"
	"srpg/internal/core"
	"strings"
	"testing"
)

func TestStatusIconsShowDirectionAndSkipExpired(t *testing.T) {
	icons := unitStatusIcons(map[string]core.Status{
		"attack": {Turns: 2, Value: 20}, "defense": {Turns: 2, Value: -20},
		"attack-down": {Turns: 1, Value: 10}, "mind": {Turns: 2, Value: 15},
		"morale": {Turns: 0, Value: 20}, "speed": {Turns: 2, Value: 0},
		"poison": {Turns: 3}, "counter": {Turns: 1},
	})
	check(t, len(icons), 5)
	check(t, icons[0].kind, "attack")
	check(t, icons[0].down, false)
	check(t, icons[1].kind, "attack")
	check(t, icons[1].down, true)
	check(t, icons[2].kind, "defense")
	check(t, icons[2].down, true)
	check(t, icons[3].kind, "mind")
	check(t, icons[4].kind, "counter")

}

func TestOriginalStatsExcludeTemporaryEffects(t *testing.T) {
	g := battleController(t)
	before := g.S.Engine.Observe()
	u := g.unitView(before, "hero")
	if u == nil {
		t.Fatal("missing unit")
	}
	base := g.originalStats(u)
	snap := g.S.Engine.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == u.ID {
			snap.Units[i].Status = map[string]core.Status{"attack": {Turns: 2, Value: 25}, "defense-down": {Turns: 2, Value: 20}}
		}
	}
	e, err := core.Restore(g.S.Data, snap)
	if err != nil {
		t.Fatal(err)
	}
	g.S.Engine = e
	u = g.unitView(e.Observe(), "hero")
	check(t, g.originalStats(u), base)
	if u.Stats.Attack <= base.Attack || u.Stats.Defense >= base.Defense {
		t.Fatal("buff/debuff did not change displayed stats")
	}
}

func TestMPDisplayRecovery(t *testing.T) {
	g := battleController(t)
	v := &unitVis{shownMP: 5, maxMP: 20}
	g.vis = map[string]*unitVis{"hero": v}
	g.strike(nil, []core.Event{{Kind: "recover-mp", Actor: "hero", Amount: 10}})
	check(t, v.shownMP, 15)
	g.strike(nil, []core.Event{{Kind: "recover-mp", Actor: "hero", Amount: 10}})
	check(t, v.shownMP, 20)
}

func TestStatusPixelLayout(t *testing.T) {
	status := map[string]core.Status{
		"poison": {Turns: 2}, "root": {Turns: 0}, "seal": {Turns: 1},
		"attack": {Turns: 2, Value: 15}, "defense": {Turns: 1, Value: -10},
	}
	check(t, unitStatusKeys(status), []string{"poison", "seal", "attack-up", "defense-down"})
	check(t, statusBarOffset(&unitVis{status: status}), float64(18))
	check(t, statusBarOffset(&unitVis{status: map[string]core.Status{"poison": {Turns: 0}}}), float64(4))
}

func TestStatusPixelAssets(t *testing.T) {
	data, err := os.ReadFile("../../assets/graphics/status/icons.json")
	if err != nil {
		t.Fatal(err)
	}
	var rects map[string][4]int
	if err := json.Unmarshal(data, &rects); err != nil {
		t.Fatal(err)
	}
	im, err := loadRGBA("../../assets/graphics/status/icons.png")
	if err != nil {
		t.Fatal(err)
	}
	check(t, len(rects), 26)
	for key, r := range rects {
		t.Run(key, func(t *testing.T) {
			check(t, r[2], 9)
			check(t, r[3], 9)
			opaque := 0
			for y := 0; y < 9; y++ {
				for x := 0; x < 9; x++ {
					c := im.RGBAAt(r[0]+x, r[1]+y)
					if c.A != 0 && c.A != 255 {
						t.Fatal("antialiased status pixel")
					}
					if c.A == 255 {
						opaque++
					}
					if strings.HasSuffix(key, "-up") || strings.HasSuffix(key, "-down") {
						if (x < 2 || x > 6 || y < 1 || y > 6) && c.A != 0 {
							t.Fatal("arrow exceeds 5x6 pixels")
						}
					}
				}
			}
			if opaque < 8 || opaque > 65 {
				t.Fatalf("unexpected silhouette area: %d", opaque)
			}
			if im.RGBAAt(r[0], r[1]).A != 0 {
				t.Fatal("opaque background")
			}
		})
	}
}
