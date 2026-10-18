package content

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBattleConditions(t *testing.T) {
	for in, want := range map[string]Cond{
		"round>=3":                      {Op: "round", N: 3},
		"turnstart:enemy":               {Op: "turnstart", Arg: "enemy", N: 1},
		"defeated:guards":               {Op: "defeated", Arg: "guards", N: 1},
		"alldefeated:guards":            {Op: "alldefeated", Arg: "guards", N: 1},
		"at:arsonists:3":                {Op: "at", Arg: "arsonists", Arg2: "3", N: 3},
		"!allat:troops:12":              {Not: true, Op: "allat", Arg: "troops", Arg2: "12", N: 12},
		"adjacent:a:b":                  {Op: "adjacent", Arg: "a", Arg2: "b", N: 1},
		"hp:lubu<30":                    {Op: "hp", Arg: "lubu", N: 30},
		"duel:hulao-mushun":             {Op: "duel", Arg: "hulao-mushun", N: 1},
		"any:[round>=4, defeated:camp]": {Op: "any", Arg: "[round>=4, defeated:camp]", N: 1, Any: []Cond{{Op: "round", N: 4}, {Op: "defeated", Arg: "camp", N: 1}}},
	} {
		got, err := ParseCond(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"round>=0", "round>=x", "turnstart:nobody", "at:x", "at:x:0", "at::2", "adjacent:a", "hp:x<0", "hp:x<101", "hp:x", "any:round>=2", "any:[any:[round>=2]]", "any:[nonsense]", "duel:"} {
		_, err := ParseCond(bad)
		assert.Error(t, err, bad)
	}
}

// loadWithStage loads the mini bundle with its first stage file changed by edit.
func loadWithStage(t *testing.T, edit func(map[string]any)) error {
	dir := filepath.Join(t.TempDir(), "mini")
	copyTree(t, "../testdata/mini", dir)
	p := filepath.Join(dir, "scenario", "mini", "stage01.json")
	b, err := os.ReadFile(p)
	require.NoError(t, err)
	var st map[string]any
	require.NoError(t, json.Unmarshal(b, &st))
	edit(st)
	b, err = json.Marshal(st)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, b, 0644))
	_, err = Load(dir)
	return err
}

func objs(v ...any) []any { return v }
func obj(kv ...any) map[string]any {
	m := map[string]any{}
	for i := 0; i < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

var camp = obj("ID", 1, "Rect", obj("X", 8, "Y", 1, "W", 4, "H", 4), "Name", "camp", "Description", "the enemy camp")

func TestLoadAcceptsEventsAndAreas(t *testing.T) {
	require.NoError(t, loadWithStage(t, func(st map[string]any) {
		st["Areas"] = objs(camp)
		st["Events"] = objs(
			obj("ID", "a", "If", objs("round>=2", "at:boss:1", "any:[defeated:boss, alive:hero]", "!flag:x", "turnstart:ally"),
				"Do", objs(obj("Kind", "say", "Lines", objs(obj("Actor", "boss", "Text", "hi"))), obj("Kind", "flag", "Flags", obj("x", 1)))),
			obj("ID", "b", "Repeat", true, "If", objs("hp:boss<50"), "Do", objs(obj("Kind", "remove", "Targets", objs("boss")))),
		)
	}))
}

func TestLoadRejectsBadEventsAndAreas(t *testing.T) {
	say := obj("Kind", "say", "Lines", objs(obj("Actor", "boss", "Text", "hi")))
	event := func(extra ...any) map[string]any {
		return obj(append([]any{"ID", "e", "Do", objs(say)}, extra...)...)
	}
	cases := map[string]func(map[string]any){
		"duplicate area id": func(st map[string]any) { st["Areas"] = objs(camp, camp) },
		"area without name": func(st map[string]any) { st["Areas"] = objs(obj("ID", 1, "Rect", obj("X", 0, "Y", 0, "W", 1, "H", 1))) },
		"area id zero": func(st map[string]any) {
			st["Areas"] = objs(obj("ID", 0, "Rect", obj("X", 0, "Y", 0, "W", 1, "H", 1), "Name", "n"))
		},
		"area empty": func(st map[string]any) {
			st["Areas"] = objs(obj("ID", 1, "Rect", obj("X", 0, "Y", 0, "W", 0, "H", 1), "Name", "n"))
		},
		"area off the map": func(st map[string]any) {
			st["Areas"] = objs(obj("ID", 1, "Rect", obj("X", 40, "Y", 0, "W", 2, "H", 2), "Name", "n"))
		},
		"event without id": func(st map[string]any) { st["Events"] = objs(obj("Do", objs(say))) },
		"event without do": func(st map[string]any) { st["Events"] = objs(obj("ID", "e")) },
		"duplicate event":  func(st map[string]any) { st["Events"] = objs(event(), event()) },
		"unknown action":   func(st map[string]any) { st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "explode")))) },
		"say unknown actor": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "say", "Lines", objs(obj("Actor", "ghost", "Text", "x"))))))
		},
		"say without lines": func(st map[string]any) { st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "say")))) },
		"remove unknown": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "remove", "Targets", objs("ghost")))))
		},
		"remove nothing": func(st map[string]any) { st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "remove")))) },
		"flag nothing":   func(st map[string]any) { st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "flag")))) },
		"spawn nothing":  func(st map[string]any) { st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "spawn")))) },
		"repeat spawns": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "e", "Repeat", true, "Do", objs(obj("Kind", "spawn", "Spawns", objs(obj("Character", "troop", "X", 1, "Y", 1, "AI", true))))))
		},
		"unknown target":       func(st map[string]any) { st["Events"] = objs(event("If", objs("defeated:ghost"))) },
		"unknown area":         func(st map[string]any) { st["Events"] = objs(event("If", objs("at:boss:9"))) },
		"unknown duel":         func(st map[string]any) { st["Events"] = objs(event("If", objs("duel:nope"))) },
		"story-only condition": func(st map[string]any) { st["Events"] = objs(event("If", objs("dead:boss"))) },
		"bad condition":        func(st map[string]any) { st["Events"] = objs(event("If", objs("round>=0"))) },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) { assert.Error(t, loadWithStage(t, edit)) })
	}
}

func TestStoryNodesRefuseBattleConditions(t *testing.T) {
	for _, cond := range []string{"round>=2", "turnstart:ally", "defeated:hero", "at:hero:1", "duel:x"} {
		d := &Data{Characters: map[string]Character{"hero": {}}}
		assert.Error(t, d.validateCond(cond, nil), cond)
	}
}

func TestLoadRejectsBadObjectives(t *testing.T) {
	win := obj("If", objs("defeated:boss"), "Text", "boss")
	cases := map[string]func(map[string]any){
		"no victory":            func(st map[string]any) { st["Victory"] = objs() },
		"victory without if":    func(st map[string]any) { st["Victory"] = objs(obj("Text", "x")) },
		"victory without text":  func(st map[string]any) { st["Victory"] = objs(obj("If", objs("defeated:boss"))) },
		"victory unknown unit":  func(st map[string]any) { st["Victory"] = objs(obj("If", objs("defeated:ghost"), "Text", "x")) },
		"victory story cond":    func(st map[string]any) { st["Victory"] = objs(obj("If", objs("dead:boss"), "Text", "x")) },
		"defeat without if":     func(st map[string]any) { st["Defeat"] = objs(obj("Text", "x")) },
		"defeat unknown area":   func(st map[string]any) { st["Defeat"] = objs(obj("If", objs("at:enemy:5"), "Text", "x")) },
		"required unknown":      func(st map[string]any) { st["Required"] = objs("ghost") },
		"required duplicate":    func(st map[string]any) { st["Required"] = objs("hero", "hero") },
		"required not playable": func(st map[string]any) { st["Required"] = objs("boss") },
		"too many required":     func(st map[string]any) { st["Required"] = objs("hero", "archer1", "sage") },
		"no sortie places":      func(st map[string]any) { st["Max"] = 0 },
		"objective duplicate": func(st map[string]any) {
			o := obj("ID", "a", "Text", "t", "If", objs("alive:hero"))
			st["Sub"] = objs(o, o)
		},
		"objective without text": func(st map[string]any) { st["Sub"] = objs(obj("ID", "a", "If", objs("alive:hero"))) },
		"objective without if":   func(st map[string]any) { st["Sub"] = objs(obj("ID", "a", "Text", "t")) },
		"objective unknown reward": func(st map[string]any) {
			st["Sub"] = objs(obj("ID", "a", "Text", "t", "If", objs("alive:hero"), "Reward", obj("gold", 1)))
		},
		"reserved group name": func(st map[string]any) { st["Enemies"].([]any)[1].(map[string]any)["Group"] = "enemy" },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) { assert.Error(t, loadWithStage(t, edit)) })
	}
	assert.NoError(t, loadWithStage(t, func(st map[string]any) {
		st["Victory"] = objs(win, obj("If", objs("alldefeated:enemy", "round>=2"), "Text", "rout"))
		st["Defeat"] = objs(obj("If", objs("alldefeated:ally")))
		st["Required"] = objs("hero")
		st["Sub"] = objs(obj("ID", "a", "Text", "t", "If", objs("alive:hero"), "Flag", "flag-x", "Reward", obj("tonic", 1)))
	}))
}

func TestLoadChecksPoliciesAndAIActions(t *testing.T) {
	spawn := func(extra ...any) map[string]any {
		return obj(append([]any{"Character", "troop", "X", 1, "Y", 1, "AI", true}, extra...)...)
	}
	ai := func(extra ...any) map[string]any {
		return obj("ID", "e", "Do", objs(obj(append([]any{"Kind", "ai", "Targets", objs("boss")}, extra...)...)))
	}
	bad := map[string]func(map[string]any){
		"unknown policy":     func(st map[string]any) { st["Enemies"].([]any)[1].(map[string]any)["Policy"] = "dance" },
		"march without goal": func(st map[string]any) { st["Enemies"].([]any)[1].(map[string]any)["Policy"] = "march" },
		"goal off the map": func(st map[string]any) {
			p := st["Enemies"].([]any)[1].(map[string]any)
			p["Policy"], p["Goal"] = "flee", obj("X", 99, "Y", 0)
		},
		"hold with a goal": func(st map[string]any) {
			p := st["Enemies"].([]any)[1].(map[string]any)
			p["Policy"], p["Goal"] = "hold", obj("X", 1, "Y", 1)
		},
		"goal without policy": func(st map[string]any) { st["Enemies"].([]any)[1].(map[string]any)["Goal"] = obj("X", 1, "Y", 1) },
		"event spawn bad policy": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "spawn", "Spawns", objs(spawn("Policy", "march"))))))
		},
		"ai without policy": func(st map[string]any) { st["Events"] = objs(ai()) },
		"ai unknown target": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "ai", "Targets", objs("ghost"), "Policy", "hold"))))
		},
		"ai without targets": func(st map[string]any) {
			st["Events"] = objs(obj("ID", "e", "Do", objs(obj("Kind", "ai", "Policy", "hold"))))
		},
		"ai march without goal": func(st map[string]any) { st["Events"] = objs(ai("Policy", "march")) },
		"ai goal off the map":   func(st map[string]any) { st["Events"] = objs(ai("Policy", "flee", "Goal", obj("X", -1, "Y", 0))) },
	}
	for name, edit := range bad {
		t.Run(name, func(t *testing.T) { assert.Error(t, loadWithStage(t, edit)) })
	}
	assert.NoError(t, loadWithStage(t, func(st map[string]any) {
		p := st["Enemies"].([]any)[1].(map[string]any)
		p["Policy"], p["Goal"] = "march", obj("X", 2, "Y", 2)
		st["Enemies"].([]any)[2].(map[string]any)["Policy"] = "assault"
		st["Events"] = objs(
			obj("ID", "a", "Do", objs(obj("Kind", "ai", "Targets", objs("boss"), "Policy", "flee", "Goal", obj("X", 0, "Y", 0)))),
			obj("ID", "b", "Do", objs(obj("Kind", "ai", "Targets", objs("boss"), "Policy", "hold"))),
			obj("ID", "c", "Do", objs(obj("Kind", "spawn", "Spawns", objs(spawn("Policy", "assault", "Side", "enemy"))))))
	}))
}

func TestValidPolicyChecksPolicyAgainstGoal(t *testing.T) {
	inside, outside := &Point{X: 3, Y: 3}, &Point{X: 12, Y: 3}
	for name, c := range map[string]struct {
		policy string
		goal   *Point
		want   bool
	}{
		"default":             {"", nil, true},
		"hold":                {"hold", nil, true},
		"assault":             {"assault", nil, true},
		"march to a cell":     {"march", inside, true},
		"flee to a cell":      {"flee", inside, true},
		"march needs a goal":  {"march", nil, false},
		"goal off the map":    {"flee", outside, false},
		"hold has no goal":    {"hold", inside, false},
		"goal needs a policy": {"", inside, false},
		"unknown policy":      {"charge", nil, false},
	} {
		assert.Equal(t, c.want, ValidPolicy(c.policy, c.goal, 12, 8), name)
	}
}
