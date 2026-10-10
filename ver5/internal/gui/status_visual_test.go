package gui

import (
	"srpg/internal/core"
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
	if icons[0].tint == icons[2].tint || icons[2].tint == icons[3].tint {
		t.Fatal("attribute colors are not distinct")
	}
	check(t, statusIconColors["agility"], statusIconColors["morale"])
}
