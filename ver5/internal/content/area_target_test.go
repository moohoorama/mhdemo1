package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadChecksAreaTargets(t *testing.T) {
	ok := func(st map[string]any) {
		st["Areas"] = objs(camp)
		st["Events"] = objs(obj("ID", "e", "Do", objs(
			obj("Kind", "damage", "Targets", objs("enemy@area:1", "area:1"), "Value", 50),
			obj("Kind", "buff", "Targets", objs("boss@area:1"), "Stat", "confusion"))))
	}
	require.NoError(t, loadWithStage(t, ok))
	bad := map[string]func(map[string]any){
		"unknown area in a target": func(st map[string]any) {
			st["Areas"] = objs(camp)
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "damage", "Targets", objs("enemy@area:9"), "Value", 50))))
		},
		"area target without areas": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "remove", "Targets", objs("area:1")))))
		},
		"unknown side in an area target": func(st map[string]any) {
			st["Areas"] = objs(camp)
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "remove", "Targets", objs("ghost@area:1")))))
		},
		"area number is not a number": func(st map[string]any) {
			st["Areas"] = objs(camp)
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "remove", "Targets", objs("enemy@area:x")))))
		},
	}
	for name, edit := range bad {
		t.Run("rejects "+name, func(t *testing.T) { assert.Error(t, loadWithStage(t, edit)) })
	}
}
