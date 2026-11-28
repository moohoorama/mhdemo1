package gui

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// TestLeavingKeepsTheWalkingPace walks an actor out by a map exit at a run (Speed 2) and checks that the walk off
// the map, fading, keeps the pace of the walk before it instead of slowing down at the edge.
func TestLeavingKeepsTheWalkingPace(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	const mapID = "S_zhuo"
	art := g.assets.Maps[mapID]
	require.NotNil(t, art)
	require.NotEmpty(t, art.SceneMap.Exits)
	gr := newExitGround(art, g.assets.SceneRules)
	e := art.SceneMap.Exits[0]
	at := exitCell(e)
	st := scenemap.ExitStep[e.Dir]
	var from [2]float64
	found := false
	for k := 4.0; k < 12 && !found; k++ { // a start a few cells inside the exit with a way out through it
		from = [2]float64{at[0] - float64(st[0])*k, at[1] - float64(st[1])*k}
		path, _, ok := gr.throughExit(from, e.ID, nil)
		found = ok && len(path) >= 4
	}
	require.True(t, found, "a start inside exit %s", e.ID)

	g.openScene(&scenes.Scene{ID: "pace", Map: mapID, Cast: map[string]scenes.Actor{"유비": {At: from[:], Face: e.Dir}}})
	g.startStep(scenes.Step{Actor: "유비", Exit: true, Out: e.ID, Speed: 2})
	v := g.scene.actors["유비"]
	require.NotNil(t, v)
	const dt = 1.0 / 60
	var walkDist, walkTime, leaveDist, leaveTime float64
	last := [2]float64{v.u, v.v}
	for i := 0; i < 60*20 && !v.gone; i++ {
		leaving := v.leave != nil
		g.updateScene(dt)
		step := math.Hypot(v.u-last[0], v.v-last[1])
		last = [2]float64{v.u, v.v}
		if leaving {
			leaveDist, leaveTime = leaveDist+step, leaveTime+dt
		} else if step > 0 {
			walkDist, walkTime = walkDist+step, walkTime+dt
		}
	}
	require.True(t, v.gone, "the actor leaves")
	require.Positive(t, walkTime)
	require.Positive(t, leaveTime)
	walk, off := walkDist/walkTime, leaveDist/leaveTime
	t.Logf("walk %.2f cells/s, off the map %.2f cells/s", walk, off)
	assert.InEpsilon(t, walk, off, .15, "the walk off the map keeps the walking pace")
}
