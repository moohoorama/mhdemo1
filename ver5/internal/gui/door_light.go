package gui

import (
	"strings"

	"srpg/internal/content"
)

// doorLight lays the sun coming in at a room's doors (docs/plan-scene-exits.md): at each exit of the area that leads
// out to a street (its Face set, toward the room's front walls SW or SE, on a map without named exits of its own) a
// pool of light on the floor and a faint shaft over whoever stands in it, warm white by day, orange at dusk, none at
// night. The room maps are shared by many towns, so the light follows the area's exit, not the map.
func (g *Game) doorLight(a *content.ExploreArea) {
	s := g.scene
	if s == nil || a == nil {
		return
	}
	if art := g.mapArt(a.Map); art == nil || art.SceneMap == nil || len(art.SceneMap.Exits) > 0 {
		return
	}
	tone := map[string]string{"": "day", "dawn": "day", "dusk": "dusk"}[s.tod]
	if tone == "" {
		return
	}
	for _, e := range a.Exits {
		if e.Face != "SW" && e.Face != "SE" {
			continue
		}
		u, v := smallTileCenter(e.At)
		for _, kind := range []string{"door_light", "door_beam"} {
			name := kind + "_" + strings.ToLower(e.Face) + "_" + tone
			if _, ok := g.assets.Props[name]; ok {
				s.props = append(s.props, sceneProp{name: name + "#" + e.Name, u: u, v: v})
			}
		}
	}
}

// smallTileCenter is the scene position whose map point is the centre of small tile t: a prop drawn there has its
// pivot on that tile's centre (a cell's point is the centre of its 2x2 small tiles, so a tile is a quarter cell off it).
func smallTileCenter(t [2]int) (float64, float64) {
	return float64(t[0])/2 - .25, float64(t[1])/2 - .25
}
