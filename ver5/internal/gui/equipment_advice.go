package gui

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// Cache by engine as well as revision, since a load or a new game can reuse a revision.
type equipmentAdviceCache struct {
	engine   *core.Engine
	revision uint64
	byActor  map[string]map[string]map[string]string // actor -> slot -> item -> reason
}

func (g *Game) equipmentAdvice(o core.Observation, actor string) map[string]map[string]string {
	if g.S.Settings.NoGuide || g.S.Engine == nil {
		return nil
	}
	cache := &g.gearAdvice
	if cache.engine != g.S.Engine || cache.revision != o.Revision {
		*cache = equipmentAdviceCache{engine: g.S.Engine, revision: o.Revision, byActor: make(map[string]map[string]map[string]string)}
		for _, up := range cache.engine.EquipmentUpgrades() {
			slots := cache.byActor[up.Actor]
			if slots == nil {
				slots = make(map[string]map[string]string)
				cache.byActor[up.Actor] = slots
			}
			if slots[up.Slot] == nil {
				slots[up.Slot] = make(map[string]string)
			}
			slots[up.Slot][up.Item] = g.equipmentReason(up)
		}
	}
	return cache.byActor[actor]
}

type shopAdviceCache struct {
	engine   *core.Engine
	revision uint64
	item     string
	ups      []core.EquipmentUpgrade
}

// shopAdvice lists who would gain from buying one of an item, cached like equipmentAdvice.
func (g *Game) shopAdvice(o core.Observation, item string) []core.EquipmentUpgrade {
	if g.S.Settings.NoGuide || g.S.Engine == nil {
		return nil
	}
	c := &g.shopGain
	if c.engine != g.S.Engine || c.revision != o.Revision || c.item != item {
		*c = shopAdviceCache{engine: g.S.Engine, revision: o.Revision, item: item, ups: g.S.Engine.EquipmentUpgradesFor(item)}
	}
	return c.ups
}

func (g *Game) equipmentReason(up core.EquipmentUpgrade) string {
	switch {
	case len(up.Skills) > 0:
		return g.skillName(up.Skills[0]) + " 사용 가능"
	case up.MovementGain > 0:
		return fmt.Sprintf("이동력 +%d", up.MovementGain)
	case len(up.Traits) > 0:
		return g.traitName(up.Traits[0]) + " 효과"
	}
	if changes := statDeltas(up.Before, up.After); len(changes) > 0 {
		return fmt.Sprintf("%s +%d", changes[0].name, changes[0].d)
	}
	if gain := up.After.Recovery - up.Before.Recovery; gain > 0 {
		return fmt.Sprintf("병법치 회복 +%d", gain)
	}
	return g.traitName(up.Learn[0]) + " 학습 목록 추가"
}

func recommendationRim(dst *ebiten.Image, r image.Rectangle) {
	x, y, w, h := float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
	const edge = 2
	c := guideGreen[1]
	c.A = 210
	rect(dst, x, y, w, edge, c)
	rect(dst, x, y+h-edge, w, edge, c)
	rect(dst, x, y+edge, edge, h-2*edge, c)
	rect(dst, x+w-edge, y+edge, edge, h-2*edge, c)
}
