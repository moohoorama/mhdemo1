package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuigongDemo(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "suigong", Stage: "B25", Frames: 10})
	require.NoError(t, err)
	f := d.g.field
	require.NotNil(t, f.Surge)

	t.Run("the army fills the ford shot's left and bottom, the cliffs flank the gorge", func(t *testing.T) {
		assert.Greater(t, len(suigongArmy()), 60)
		assert.EqualValues(t, 'h', suigongGround(suigongRiver-1, suigongDam))
		assert.EqualValues(t, 'h', suigongGround(suigongRiver+suigongWidth, suigongDam))
		assert.EqualValues(t, 'w', suigongGround(suigongRiver+suigongWidth-1, suigongDam), "the river is six cells wide")
		assert.EqualValues(t, 'g', suigongGround(suigongRiver-1, suigongFord), "the cliffs end below the dam")
	})
	t.Run("the dam holds until it bursts", func(t *testing.T) {
		f.SurgeT = 0
		assert.Zero(t, f.surgeDepth(suigongRiver+1, suigongFord+1), "the ford is only wet before the flood")
		assert.NotEmpty(t, f.surgeItems(), "the gorge's rock and the water behind the dam")
	})
	t.Run("관우 and 미축 stand on the cliff over the dam, looking down the river", func(t *testing.T) {
		_, group := suigongGroupOf(d.g.S.Data.Stages[d.g.S.Data.StageIndex("B25")])
		var stands []string
		for _, st := range group {
			if st.Kind == "face" {
				stands = append(stands, st.Unit)
			}
		}
		require.Len(t, stands, 2)
		for i, id := range stands {
			p := suigongStands[i]
			u := d.g.vis[id]
			require.NotNil(t, u, id)
			assert.Equal(t, [2]float64{float64(p.X), float64(p.Y)}, [2]float64{u.u, u.v}, id)
			assert.Equal(t, float64(d.g.field.A.CliffRise), u.lift, id)
			assert.Equal(t, "SW", u.dir, id)
		}
	})
	t.Run("the flood stands over the ford at the end", func(t *testing.T) {
		f.SurgeT = f.Surge.Length()
		assert.Greater(t, f.surgeDepth(suigongRiver+1, suigongFord+1), 5.0)
		assert.Zero(t, f.surgeDepth(suigongRiver+suigongWidth+4, suigongFord+1), "the bank far from the river stays dry")
	})
}

func TestSuigongMarch(t *testing.T) {
	for _, r := range suigongArmy() {
		assert.True(t, r.u >= 0 && r.u < suigongW && r.v >= suigongFord && r.v < suigongH, "start cell (%d,%d) is on the map", r.u, r.v)
	}
	d, err := NewDemo("../..", DemoOptions{Name: "suigong", Stage: "B25", Frames: 10})
	require.NoError(t, err)
	var soldier *unitVis
	for _, u := range d.g.vis {
		if !u.ally && u.v >= suigongFord && !u.mounted && u.u == float64(int(u.u)) && (soldier == nil || u.id < soldier.id) {
			soldier = u
		}
	}
	require.NotNil(t, soldier)
	from, fed := soldier.u, d.g.sfxCalls
	const secs = 3
	for f := 0; f < 60*secs; f++ {
		walks := map[string]*walkState{}
		for id, u := range d.g.vis {
			walks[id] = u.walk
		}
		before := d.g.sfxCalls
		d.g.advance(1.0 / 60)
		began := 0
		for id, u := range d.g.vis {
			x, y := d.g.field.Center(u.u, u.v)
			heard := distanceGain((x-d.g.camX)*d.g.zoom/(float64(worldW)/2), (y-d.g.camY)*d.g.zoom/(float64(worldH)/2)) > 0
			if u.walk != nil && u.walk != walks[id] && heard {
				began++
			}
		}
		require.Equal(t, began, d.g.sfxCalls-before, "frame %d: every cell a soldier begins within earshot sounds one footstep or hoof beat, and only those do", f)
		assert.Equal(t, "walk", soldier.anim)
		assert.Equal(t, "NW", soldier.dir)
	}
	moved := from - soldier.u
	assert.InDelta(t, secs*marchSpeed, moved, 1, "it walks a cell in stepTime, not at some other pace")
	assert.Greater(t, d.g.sfxCalls-fed, 600, "a crowd of footsteps")
	assert.Greater(t, soldier.animT, 0.0)
}
