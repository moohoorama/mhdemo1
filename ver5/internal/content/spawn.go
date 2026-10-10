package content

import "fmt"

func (d *Data) ClassFamily(id string) string {
	c := d.Classes[id]
	if c.Family != "" {
		return c.Family
	}
	return c.Terrain
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
	if s.Facing != "" && !in(s.Facing, "NE", "SE", "SW", "NW") {
		return fmt.Errorf("방향 오류")
	}
	if s.HP != nil && *s.HP < 1 || s.MP != nil && *s.MP < 0 {
		return fmt.Errorf("HP/MP 범위 오류")
	}
	if s.Traits != nil {
		for _, id := range *s.Traits {
			if _, ok := d.Traits[id]; !ok {
				return fmt.Errorf("알 수 없는 특성 %s", id)
			}
		}
	}
	if s.Equipment != nil {
		slots := map[string]bool{}
		cl := d.Classes[class]
		for _, id := range *s.Equipment {
			eq, ok := d.Equipment[id]
			if !ok || slots[eq.Slot] || eq.Slot == "weapon" && eq.Type != cl.Weapon || eq.Slot == "armor" && eq.Type != cl.Armor {
				return fmt.Errorf("장비 호환/슬롯 오류 %s", id)
			}
			slots[eq.Slot] = true
		}
	}
	effective := map[string]bool{}
	base := c.Traits
	if s.Traits != nil {
		base = *s.Traits
	}
	for _, id := range append(append([]string{}, base...), d.Classes[class].Grants...) {
		effective[id] = true
	}
	equipment := c.Equipment
	if s.Equipment != nil {
		equipment = *s.Equipment
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
