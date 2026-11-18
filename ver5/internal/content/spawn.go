package content

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
)

// Promotes is where a character serving in class can be promoted: the class's branches, or for a character drawn
// per class (Look.Sprites) only the branches it has a sprite for, so its own look follows every promotion.
func (d *Data) Promotes(character, class string) []string {
	next, sprites := d.Classes[class].Promotes, d.Characters[character].Look.Sprites
	if len(sprites) == 0 {
		return next
	}
	var out []string
	for _, c := range next {
		for key := range sprites {
			if k, _, _ := strings.Cut(key, ":"); k == c {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// ClassLine is the promotion line that ends in class, from its first-tier class to class itself.
func (d *Data) ClassLine(class string) []string {
	line := []string{class}
	for c := d.PromotedFrom(class); c != ""; c = d.PromotedFrom(c) {
		line = append([]string{c}, line...)
	}
	return line
}

// PromotedAt is the class an officer of def in class serves in at the level. A character's data names its final
// class: on that line the officer takes the furthest class the level reaches, stepping back or on. Off it (a branch
// an 인수 chose, a stage's own class) a class the level has not reached steps back to the class it was promoted
// from, and one the level has passed goes on through every promotion the level allows, each time to its first
// branch.
func (d *Data) PromotedAt(def, class string, level int) string {
	if line := d.ClassLine(d.Characters[def].Class); slices.Contains(line, class) {
		for i := len(line) - 1; i > 0; i-- {
			if t := d.Classes[line[i]].Tier; t <= len(d.Rules.PromoteLevels) && level >= d.Rules.PromoteLevels[t-1] {
				return line[i]
			}
		}
		return line[0]
	}
	for t := d.Classes[class].Tier; t > 0 && level < d.Rules.PromoteLevels[t-1]; t = d.Classes[class].Tier {
		prev := d.PromotedFrom(class)
		if prev == "" {
			break
		}
		class = prev
	}
	for {
		cl, next := d.Classes[class], d.Promotes(def, class)
		if cl.Tier >= len(d.Rules.PromoteLevels) || level < d.Rules.PromoteLevels[cl.Tier] || len(next) == 0 {
			return class
		}
		class = next[0]
	}
}

// PromotedFrom is the class whose promotions lead to class, or "" for a first-tier class.
func (d *Data) PromotedFrom(class string) string {
	for _, id := range slices.Sorted(maps.Keys(d.Classes)) {
		if slices.Contains(d.Classes[id].Promotes, class) {
			return id
		}
	}
	return ""
}

func (d *Data) ClassFamily(id string) string {
	c := d.Classes[id]
	if c.Family != "" {
		return c.Family
	}
	return c.Terrain
}

// Carries reports whether a class holding the traits can wield the equipment. Weapons and armor must match the
// class's types; a weapon type may also come from a trait's FxWield.
func (d *Data) Carries(cl Class, traits []string, eq Equipment) bool {
	switch eq.Slot {
	case "weapon":
		if eq.Type == cl.Weapon {
			return true
		}
		for _, t := range traits {
			if d.Traits[t].Effects[FxWield+eq.Type] != 0 {
				return true
			}
		}
		return false
	case "armor":
		return eq.Type == cl.Armor
	}
	return true
}

// SpawnTraits are the traits a spawn may start with: its own list, else its character's whole plan.
func (d *Data) SpawnTraits(s Spawn) []string {
	if s.Traits != nil {
		return *s.Traits
	}
	return d.Characters[s.Character].Traits
}

// PlannedTraits are the traits an NPC officer of the character holds in the class at the level: the first of its
// own plan, one more every Rules.TraitPlanStep levels, and those of its TraitAt it has the level for (or own in full
// when the stage sets it), then the class plan's.
func (d *Data) PlannedTraits(def, class string, level int, own *[]string) []string {
	var out []string
	if own != nil {
		out = slices.Clone(*own)
	} else if plan := d.Characters[def].Traits; len(plan) > 0 && d.Rules.TraitPlanStep > 0 {
		out = slices.Clone(plan[:min(len(plan), (max(level, 1)-1)/d.Rules.TraitPlanStep+1)])
	}
	if own == nil {
		for _, t := range slices.Sorted(maps.Keys(d.Characters[def].TraitAt)) {
			if level >= d.Characters[def].TraitAt[t] && !slices.Contains(out, t) {
				out = append(out, t)
			}
		}
	}
	for _, lt := range d.Classes[class].Plan {
		if level >= lt.Level && !slices.Contains(out, lt.Trait) {
			out = append(out, lt.Trait)
		}
	}
	return out
}

// StageLevel is the level the order-th battle of the story is fought at: the player's officers are
// expected there, and the stage's spawns are ranked from it. Zero when the rules set no stage level.
func (r Rules) StageLevel(order int) int {
	if r.StageLevelBase == 0 {
		return 0
	}
	return r.StageLevelBase + int(math.Round(r.StageLevelStep*float64(max(order, 1)-1)))
}

// GearTier is one step of Rules.GearTiers: the tier of plain gear worn from the chapter numbered Chapter on.
type GearTier struct{ Chapter, Tier int }

// GearTierOf is the tier of plain gear soldiers wear in the chapter with the given number; 0 before the first step.
func (r Rules) GearTierOf(chapter int) int {
	tier := 0
	for _, g := range r.GearTiers {
		if chapter >= g.Chapter {
			tier = g.Tier
		}
	}
	return tier
}

func (r Rules) maxGearTier() int {
	top := 0
	for _, g := range r.GearTiers {
		top = max(top, g.Tier)
	}
	return top
}

// GearAt is the piece of the same Line as gear id at the tier (a 구리검 in the third chapter is a 강검). Gear
// that comes in no tiers, or has no piece at the tier, stays as it is.
func (d *Data) GearAt(id string, tier int) string {
	eq := d.Equipment[id]
	if eq.Line == "" || eq.Tier == tier || tier == 0 {
		return id
	}
	for other, v := range d.Equipment {
		if v.Line == eq.Line && v.Tier == tier {
			return other
		}
	}
	return id
}

// GearTierAt is the gear tier of the order-th battle of the story; 0 when there is none.
func (d *Data) GearTierAt(order int) int {
	for _, s := range d.Stages {
		if s.Order == order && order > 0 {
			return s.GearTier
		}
	}
	return 0
}

// JoinLevel is the level an officer who joins before the order-th battle is raised to: the stage level with a
// troop's lead, shifted by the difficulty (EnemyShift), so the newcomer stands to that difficulty's enemy as on normal.
func (r Rules) JoinLevel(order int, shift float64) int {
	base := r.StageLevel(order)
	if base == 0 {
		return 0
	}
	lead := (r.RankLead["troop"] + shift) * float64(base+r.StatLevelBase)
	return min(max(base+int(math.Round(lead)), 1), r.MaxLevel)
}

// SpawnLevel is the level a spawn enters the order-th battle at: the stage level plus its rank's
// lead, or its own Level or its character's where the rules set no stage level.
func (d *Data) SpawnLevel(order int, s Spawn) int {
	base := d.Rules.StageLevel(order)
	if base == 0 {
		if s.Level > 0 {
			return s.Level
		}
		return d.Characters[s.Character].Level
	}
	rank := s.Rank
	if rank == "" {
		rank = "officer"
		if d.Characters[s.Character].Template {
			rank = "troop"
		}
	}
	lead := (d.Rules.RankLead[rank] + s.Shift) * float64(base+d.Rules.StatLevelBase)
	return min(max(base+int(math.Round(lead)), 1), d.Rules.MaxLevel)
}

func (d *Data) ValidateSpawn(s Spawn) error {
	c, ok := d.Characters[s.Character]
	if !ok {
		return fmt.Errorf("알 수 없는 병력 %s", s.Character)
	}
	class := c.Class
	if s.Class != "" {
		if _, ok := d.Classes[s.Class]; !ok || d.ClassFamily(s.Class) != d.ClassFamily(c.Class) {
			return fmt.Errorf("계열이 다른 병종 %s", s.Class)
		}
		class = s.Class
	}
	if s.Level < 0 || s.Level > d.Rules.MaxLevel {
		return fmt.Errorf("레벨 범위 오류")
	}
	if d.Rules.StageLevelBase > 0 && s.Level > 0 {
		return fmt.Errorf("%s: 레벨은 스테이지 순번과 직급(Rank)으로 정한다", s.Character)
	}
	if _, ok := d.Rules.RankLead[s.Rank]; s.Rank != "" && !ok {
		return fmt.Errorf("%s: 알 수 없는 직급 %s", s.Character, s.Rank)
	}
	if s.Deputy != "" {
		if q, ok := d.Characters[s.Deputy]; !ok || q.Template || q.Playable || c.Template || c.Playable || s.Deputy == s.Character {
			return fmt.Errorf("%s: 부관 %s는 플레이어 장수가 아닌 이름 있는 장수끼리만 둔다", s.Character, s.Deputy)
		}
	}
	if s.Facing != "" && !in(s.Facing, "NE", "SE", "SW", "NW") {
		return fmt.Errorf("방향 오류")
	}
	if s.HP != nil && *s.HP < 1 || s.MP != nil && *s.MP < 0 || s.HPPct < 0 || s.HPPct > 99 || s.HPPct > 0 && s.HP != nil {
		return fmt.Errorf("HP/MP 범위 오류")
	}
	if s.Traits != nil {
		for _, id := range *s.Traits {
			if _, ok := d.Traits[id]; !ok {
				return fmt.Errorf("알 수 없는 특성 %s", id)
			}
		}
	}
	effective := map[string]bool{}
	base := c.Traits
	if s.Traits != nil {
		base = *s.Traits
	}
	for _, id := range base {
		effective[id] = true
	}
	equipment := c.Equipment
	if s.Equipment != nil {
		equipment = *s.Equipment
	}
	slots := map[string]bool{}
	for _, id := range equipment {
		eq, ok := d.Equipment[id]
		if !ok || slots[eq.Slot] || !d.Carries(d.Classes[class], d.SpawnTraits(s), eq) {
			return fmt.Errorf("장비 호환/슬롯 오류 %s", id)
		}
		slots[eq.Slot] = true
	}
	for _, id := range equipment {
		for _, t := range d.Equipment[id].Traits {
			effective[t] = true
		}
	}
	if s.Traits != nil {
		for _, id := range *s.Traits {
			for _, required := range d.Traits[id].Requires {
				if !effective[required] {
					return fmt.Errorf("특성 %s: 선행 특성 %s 필요", id, required)
				}
			}
		}
	}
	return nil
}
