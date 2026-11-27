package core

import (
	"slices"

	"srpg/internal/content"
)

// EquipmentUpgrade describes a warehouse piece that improves a character's current equipment.
type EquipmentUpgrade struct {
	Actor, Slot, Item     string
	Before, After         Stats
	MovementGain          int
	Traits, Skills, Learn []string
}

// EquipmentUpgrades compares legal changes in preparation, including effects that the stat sheet
// omits. A shared warehouse piece is evaluated independently for each officer.
func (e *Engine) EquipmentUpgrades() []EquipmentUpgrade {
	return e.upgrades("")
}

// EquipmentUpgradesFor tests one piece as if one copy were in the warehouse, as when it is for sale.
func (e *Engine) EquipmentUpgradesFor(item string) []EquipmentUpgrade {
	if _, known := e.data.Equipment[item]; !known {
		return nil
	}
	return e.upgrades(item)
}

func (e *Engine) upgrades(extra string) []EquipmentUpgrade {
	kind := e.data.Node(e.state.Node).Kind
	if e.state.Phase != "scenario" || kind != "battle" && kind != "explore" || len(e.state.Learning) > 0 {
		return nil
	}
	// Try changes in a private world. Preparation's equipment rules do not depend on placement.
	trial := fromState(e.data, e.Snapshot())
	if extra != "" {
		if trial.state.Warehouse == nil {
			trial.state.Warehouse = map[string]int{}
		}
		trial.state.Warehouse[extra]++
	}
	items := keys(trial.state.Warehouse)
	if extra != "" {
		items = []string{extra}
	}
	if len(items) == 0 {
		return nil
	}
	var out []EquipmentUpgrade
	for _, id := range keys(trial.charIDs) {
		c := trial.char(id)
		if !trial.playable(c) || c.Dead || trial.isDeputy(id) {
			continue
		}
		u := &Unit{ID: id, Character: id, Faction: "ally"}
		before := trial.equipmentProfile(c, u)
		for _, item := range items {
			eq, known := trial.data.Equipment[item]
			old := c.Equipment[eq.Slot]
			if !known || old == item || trial.checkWear(c, item, eq) != nil {
				continue
			}
			trial.wear(c, item)
			after := trial.equipmentProfile(c, u)
			if upgrade, ok := trial.equipmentUpgrade(before, after); ok {
				upgrade.Actor, upgrade.Slot, upgrade.Item = id, eq.Slot, item
				out = append(out, upgrade)
			}
			trial.state.Warehouse[item]++
			if old == "" {
				delete(c.Equipment, eq.Slot)
			} else {
				c.Equipment[eq.Slot] = old
				trial.state.Warehouse[old]--
			}
		}
	}
	return out
}

type equipmentProfile struct {
	stats                 Stats
	move                  int
	traits, skills, learn []string
}

func (e *Engine) equipmentProfile(c *Character, u *Unit) equipmentProfile {
	weapon := e.weaponType(u)
	melee, ranged := e.canMelee(u), e.canRange(u)
	skills := slices.DeleteFunc(e.skills(c), func(id string) bool {
		s := e.data.Skills[id]
		return s.Weapon != "" && s.Weapon != weapon || s.Mode == content.ModeMelee && !melee || s.Mode == content.ModeRanged && !ranged
	})
	return equipmentProfile{e.stats(u), e.movement(u), e.traits(c), skills, e.learnable(c)}
}

func (e *Engine) equipmentUpgrade(before, after equipmentProfile) (EquipmentUpgrade, bool) {
	result := EquipmentUpgrade{Before: before.stats, After: after.stats, MovementGain: after.move - before.move}
	if result.MovementGain < 0 {
		return result, false
	}
	values := func(s Stats) [8]int {
		return [8]int{s.Attack, s.Defense, s.Mind, s.Agility, s.Morale, s.MaxHP, s.MaxMP, s.Recovery}
	}
	improves := result.MovementGain > 0
	newValues := values(after.stats)
	for i, v := range values(before.stats) {
		if newValues[i] < v {
			return result, false
		}
		improves = improves || newValues[i] > v
	}
	for _, id := range before.traits {
		if slices.Contains(after.traits, id) {
			continue
		}
		// A trait granting only movement can be replaced by an equal movement bonus.
		effects := e.data.Traits[id].Effects
		if len(effects) != 1 || effects[content.FxMove] == 0 {
			return result, false
		}
	}
	for _, id := range before.skills {
		if !slices.Contains(after.skills, id) {
			return result, false
		}
	}
	for _, id := range before.learn {
		if !slices.Contains(after.learn, id) && !slices.Contains(after.traits, id) {
			return result, false
		}
	}
	for _, id := range after.traits {
		if slices.Contains(before.traits, id) {
			continue
		}
		improvesTrait := false
		for effect, value := range e.data.Traits[id].Effects {
			// Automatic item use spends resources, and negative modifiers may introduce trade-offs.
			if value < 0 || effect == content.FxPouchHeal {
				return result, false
			}
			// Movement has already been compared as a stat.
			improvesTrait = improvesTrait || effect != content.FxMove && value > 0
		}
		if improvesTrait {
			result.Traits = append(result.Traits, id)
		}
	}
	for _, id := range after.skills {
		if !slices.Contains(before.skills, id) {
			result.Skills = append(result.Skills, id)
		}
	}
	for _, id := range after.learn {
		if !slices.Contains(before.learn, id) {
			result.Learn = append(result.Learn, id)
		}
	}
	return result, improves || len(result.Traits)+len(result.Skills)+len(result.Learn) > 0
}
