package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadChecksStartGroups(t *testing.T) {
	// the mini stage's ally cells are (1,3), (1,2) and (2,2)
	areas := objs(obj("ID", 1, "Rect", obj("X", 1, "Y", 2, "W", 2, "H", 1), "Name", "north"),
		obj("ID", 2, "Rect", obj("X", 1, "Y", 3, "W", 1, "H", 1), "Name", "south"))
	split := func(edit func(st map[string]any)) error {
		return loadWithStage(t, func(st map[string]any) {
			st["Areas"] = areas
			st["Required"] = objs("archer1")
			st["Groups"] = objs(obj("Name", "본군", "Area", 1), obj("Name", "별동대", "Area", 2, "Required", objs("archer1")))
			if edit != nil {
				edit(st)
			}
		})
	}
	assert.NoError(t, split(nil))
	for name, edit := range map[string]func(map[string]any){
		"a required officer the stage does not require": func(st map[string]any) { st["Required"] = objs() },
		"an area without ally cells": func(st map[string]any) {
			st["Groups"].([]any)[0].(map[string]any)["Area"] = 3
		},
		"two groups on one cell": func(st map[string]any) {
			st["Groups"].([]any)[0].(map[string]any)["Area"] = 2
			st["Groups"].([]any)[0].(map[string]any)["Name"] = "둘"
		},
		"more leaders than cells": func(st map[string]any) {
			st["Required"] = objs("archer1", "hero")
			st["Groups"].([]any)[1].(map[string]any)["Required"] = objs("archer1", "hero")
		},
		"a leader recommended off its cells": func(st map[string]any) { st["Recommend"] = obj("archer1", obj("X", 1, "Y", 2)) },
		"a name twice": func(st map[string]any) {
			st["Groups"].([]any)[0].(map[string]any)["Name"] = "별동대"
		},
	} {
		assert.Error(t, split(edit), name)
	}
}

func TestLoadChecksWavesAilmentsAndShares(t *testing.T) {
	wave := obj("Kind", "spawn", "Spawns", objs(obj("Character", "troop", "X", 6, "Y", 6, "Side", "enemy", "Policy", "assault")))
	assert.NoError(t, loadWithStage(t, func(st map[string]any) {
		st["Events"] = objs(
			obj("ID", "wave", "Every", 3, "If", objs("turnstart:enemy"), "Do", objs(wave)),
			obj("ID", "panic", "Do", objs(obj("Kind", "buff", "Targets", objs("boss"), "Stat", "confusion", "Turns", 1))))
		st["Enemies"].([]any)[0].(map[string]any)["HPPct"] = 50
	}))
	for name, edit := range map[string]func(map[string]any){
		"Every with Repeat": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "w", "Every", 2, "Repeat", true, "Do", objs(wave)))
		},
		"a repeating spawn": func(st map[string]any) { st["Events"] = objs(obj("ID", "w", "Repeat", true, "Do", objs(wave))) },
		"an ailment with a value": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "p", "Do", objs(obj("Kind", "buff", "Targets", objs("boss"), "Stat", "confusion", "Value", 10))))
		},
		"a stat without a value": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "p", "Do", objs(obj("Kind", "buff", "Targets", objs("boss"), "Stat", "attack"))))
		},
		"a full share":   func(st map[string]any) { st["Enemies"].([]any)[0].(map[string]any)["HPPct"] = 100 },
		"share and HP":   func(st map[string]any) { e := st["Enemies"].([]any)[0].(map[string]any); e["HPPct"], e["HP"] = 50, 10 },
		"a share below0": func(st map[string]any) { st["Enemies"].([]any)[0].(map[string]any)["HPPct"] = -1 },
	} {
		assert.Error(t, loadWithStage(t, edit), name)
	}
}

func TestLoadChecksSince(t *testing.T) {
	c, err := ParseCond("since:ambush>=6")
	assert.NoError(t, err)
	assert.Equal(t, Cond{Op: "since", Arg: "ambush", N: 6}, c)
	assert.NoError(t, loadWithStage(t, func(st map[string]any) {
		st["Events"] = objs(obj("ID", "ambush", "Do", objs(obj("Kind", "flag", "Flags", obj("x", 1)))))
		st["Victory"] = objs(obj("If", objs("since:ambush>=6"), "Text", "버틴다"))
	}))
	assert.Error(t, loadWithStage(t, func(st map[string]any) {
		st["Victory"] = objs(obj("If", objs("since:nothing>=6"), "Text", "버틴다"))
	}), "an event the stage does not have")
}
