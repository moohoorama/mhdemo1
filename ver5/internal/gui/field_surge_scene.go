package gui

import (
	"fmt"

	"srpg/internal/scenes"
)

// beginSceneSuigong plays the flood of the step's surge in the story scene, on the scene's map and with its cast:
// the step's actor leads the army across the ford, every other named actor of the scene stands on the cliff over the dam
// (the map lifts them there). The army is made here, wearing the step's Army pattern, as extras of the scene who are
// gone when the show ends. The
// lines the people speak are the beats' own Say steps.
func (g *Game) beginSceneSuigong(st scenes.Step) {
	g.endSuigong()
	s := g.scene
	surge, err := g.assets.Surge(st.Surge)
	leader := s.actors[st.Actor]
	if err != nil || leader == nil || g.field == nil {
		g.toast(fmt.Sprintf("수공: %v", err), bad)
		return
	}
	var stands, cast []string
	for key := range s.actors {
		if s.extra[key] {
			cast = append(cast, key) // the scene's own soldiers give way to the army
		} else if key != st.Actor {
			stands = append(stands, key)
		}
	}
	if len(st.Army) != 2 {
		g.toast("수공: 병사 둘(보병, 기병)의 배역이 필요하다", bad)
		return
	}
	var extras []string
	for i, r := range suigongArmy() {
		pattern := s.def.Cast[st.Army[0]]
		if r.rides {
			pattern = s.def.Cast[st.Army[1]]
		}
		pattern.At, pattern.Face = []float64{float64(r.u), float64(r.v)}, "NW"
		key := fmt.Sprintf("수공#%d", i)
		v := g.castActor(key, pattern)
		if v == nil {
			continue
		}
		v.ally, v.still, v.fade = false, false, 1
		s.actors[key], s.extra[key], s.free[key] = v, true, true
		extras = append(extras, key)
	}
	s.free[st.Actor] = true
	show, err := g.newSuigongShow(surge, stands, suigongLine{}, nil, s)
	if err != nil {
		g.toast(err.Error(), bad)
		return
	}
	show.extras, show.damPlace = extras, st.DamPlace
	show.change = func() {
		for _, key := range append(cast, st.Actor) { // whoever was still wading across stops where the story left them
			s.paths[key] = nil
			s.actors[key].walk = nil
		}
		for _, key := range cast {
			s.actors[key].gone = true
		}
		for _, key := range extras {
			s.actors[key].fade = 0
		}
		leader.u, leader.v, leader.dir, leader.walk, leader.ally, leader.still = suigongRiver+2+marchLead, suigongFord+1, "NW", nil, false, false
	}
	show.node = g.S.View().Node
	g.show = show
	s.wait = show.end
	show.at(0)
}
