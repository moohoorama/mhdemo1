package gui

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type seatSpot struct {
	name string
	x, y float64 // small-tile centre
}

// seatSpots returns where a map's seats (dais, throne) and benches stand, as small-tile centres.
func seatSpots(t *testing.T, d *Demo, mapID string) (seats, benches []seatSpot) {
	art := d.g.assets.Maps[mapID]
	require.NotNil(t, art, mapID)
	if art.SceneMap == nil {
		return nil, nil
	}
	for _, o := range art.SceneMap.Objects {
		s := seatSpot{o.Name, float64(o.U) + .5 + float64(o.DX)/32 + float64(o.DY)/16, float64(o.V) + .5 - float64(o.DX)/32 + float64(o.DY)/16}
		switch {
		case strings.HasPrefix(o.Name, "dais"), o.Name == "throne_u":
			seats = append(seats, s)
		case o.Name == "bench":
			benches = append(benches, s)
		}
	}
	return seats, benches
}

func seated(pose string) bool { return pose == "sit" || pose == "throne" }

// checkSeats applies the seat rules to one person at small-tile centre (x, y); a sitter near a seat sits within sitMax of it
// (a walk's people stand on whole small tiles, so it gets a wider margin).
func checkSeats(who string, pose string, x, y, sitMax float64, seats, benches []seatSpot) []string {
	var bad []string
	for _, s := range seats {
		d := math.Hypot(x-s.x, y-s.y)
		switch {
		case seated(pose) && d < 2 && d > sitMax:
			bad = append(bad, who+" sits off "+s.name)
		case pose == "" && d < 1.2:
			bad = append(bad, who+" stands on "+s.name)
		}
	}
	if pose == "" {
		for _, b := range benches {
			if math.Hypot(x-b.x, y-b.y) < .5 {
				bad = append(bad, who+" stands in a bench")
			}
		}
	}
	return bad
}

// Whoever sits on a scene map or an explore stage sits on its seat; nobody stands on one.
func TestSceneSeats(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	var bad []string
	ids := make([]string, 0)
	for id := range d.g.scenes.Scenes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		sc := d.g.scenes.Scenes[id]
		seats, benches := seatSpots(t, d, sc.Map)
		for key, a := range sc.Cast {
			if a.Off || len(a.At) < 2 || (len(seats) == 0 && len(benches) == 0) {
				continue
			}
			bad = append(bad, checkSeats(id+"/"+key, a.Pose, 2*a.At[0]+1, 2*a.At[1]+1, .6, seats, benches)...)
		}
		if len(seats) > 0 && len(sc.Cast) > 0 {
			sat := false
			for _, a := range sc.Cast {
				sat = sat || seated(a.Pose)
			}
			if !sat {
				t.Logf("empty seat: %s on %s", id, sc.Map)
			}
		}
	}
	for _, ex := range d.g.S.Data.Explores {
		var areas []string
		for name := range ex.Areas {
			areas = append(areas, name)
		}
		sort.Strings(areas)
		for _, name := range areas {
			area := ex.Areas[name]
			if d.g.assets.Maps[area.Map] == nil {
				continue
			}
			seats, benches := seatSpots(t, d, area.Map)
			for key, p := range area.People {
				bad = append(bad, checkSeats(name+"/"+key, p.Pose, float64(p.At[0])+.5, float64(p.At[1])+.5, 1, seats, benches)...)
			}
		}
	}
	sort.Strings(bad)
	for _, b := range bad {
		t.Log(b)
	}
	assert.Empty(t, bad)
}
