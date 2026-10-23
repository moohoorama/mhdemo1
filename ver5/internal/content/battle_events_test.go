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
	st := Stage{Width: 12, Height: 8, Areas: []Area{{ID: 1, Rect: MapRect{X: 11, Y: 0, W: 1, H: 8}}, {ID: 2, Rect: MapRect{X: 20, Y: 0, W: 1, H: 8}}}}
	for name, c := range map[string]struct {
		policy string
		goal   *Point
		area   int
		want   bool
	}{
		"default":               {"", nil, 0, true},
		"hold":                  {"hold", nil, 0, true},
		"assault":               {"assault", nil, 0, true},
		"march to a cell":       {"march", inside, 0, true},
		"flee to a cell":        {"flee", inside, 0, true},
		"flee to an area":       {"flee", nil, 1, true},
		"march needs a goal":    {"march", nil, 0, false},
		"goal off the map":      {"flee", outside, 0, false},
		"area off the map":      {"flee", nil, 2, false},
		"no such area":          {"march", nil, 3, false},
		"a cell or an area":     {"flee", inside, 1, false},
		"hold has no goal":      {"hold", inside, 0, false},
		"hold has no goal area": {"hold", nil, 1, false},
		"goal needs a policy":   {"", inside, 0, false},
		"unknown policy":        {"charge", nil, 0, false},
	} {
		assert.Equal(t, c.want, st.ValidPolicy(c.policy, c.goal, c.area), name)
	}
}

func TestLoadChecksTerrainActions(t *testing.T) {
	area := obj("ID", 1, "Rect", obj("X", 5, "Y", 6, "W", 2, "H", 1), "Name", "grass")
	terrain := func(extra ...any) map[string]any {
		return obj(append([]any{"ID", "e", "Do", objs(obj("Kind", "terrain", "Area", 1, "State", "e"))}, extra...)...)
	}
	setup := func(do map[string]any) func(map[string]any) {
		return func(st map[string]any) { st["Areas"] = objs(area); st["Events"] = objs(do) }
	}
	action := func(kv ...any) map[string]any {
		return obj("ID", "e", "Do", objs(obj(append([]any{"Kind", "terrain"}, kv...)...)))
	}
	assert.NoError(t, loadWithStage(t, setup(terrain())))
	assert.NoError(t, loadWithStage(t, setup(action("Area", 1, "State", "F", "Turns", 3))))
	assert.NoError(t, loadWithStage(t, setup(action("Area", 1, "State", "-"))), "a state is cleared with the none glyph")
	for name, do := range map[string]map[string]any{
		"unknown state": action("Area", 1, "State", "?"),
		"no state":      action("Area", 1),
		"unknown area":  action("Area", 7, "State", "F"),
		"no area":       action("State", "F"),
		"negative turn": action("Area", 1, "State", "F", "Turns", -1),
	} {
		t.Run(name, func(t *testing.T) { assert.Error(t, loadWithStage(t, setup(do))) })
	}
}

func TestLoadChecksSceneActions(t *testing.T) {
	spawn := func(extra ...any) map[string]any {
		return obj(append([]any{"Character", "sage", "X", 2, "Y", 2}, extra...)...)
	}
	event := func(do ...any) func(map[string]any) {
		return func(st map[string]any) { st["Events"] = objs(obj("ID", "e", "Do", do)) }
	}
	act := func(kv ...any) map[string]any { return obj(kv...) }
	step := func(kv ...any) map[string]any { return obj(kv...) }
	at := obj("X", 3, "Y", 3)
	good := event(
		act("Kind", "buff", "Targets", objs("boss"), "Stat", "speed", "Value", -2),
		act("Kind", "buff", "Targets", objs("ally"), "Stat", "attack", "Value", 20, "Turns", 4),
		act("Kind", "join", "Spawns", objs(spawn())),
		act("Kind", "move", "Moves", objs(obj("Unit", "boss", "To", at), obj("Unit", "hero", "To", at))),
		act("Kind", "duel", "Duel", "none-needed"),
	)
	assert.Error(t, loadWithStage(t, good), "the stage has no such duel")
	assert.NoError(t, loadWithStage(t, event(
		act("Kind", "buff", "Targets", objs("boss"), "Stat", "speed", "Value", -2),
		act("Kind", "join", "Spawns", objs(spawn())),
		act("Kind", "move", "Moves", objs(obj("Unit", "boss", "To", at))),
		act("Kind", "cutscene", "Steps", objs(
			objs(step("Kind", "move", "Unit", "boss", "To", at), step("Kind", "play", "Unit", "hero", "Motion", "attack")),
			objs(step("Kind", "say", "Unit", "hero", "Text", "간다"), step("Kind", "face", "Unit", "boss", "Facing", "NE")),
			objs(step("Kind", "wait", "Seconds", 1.5)))))))
	bad := map[string]func(map[string]any){
		"buff unknown stat":    event(act("Kind", "buff", "Targets", objs("boss"), "Stat", "luck", "Value", 5)),
		"buff no value":        event(act("Kind", "buff", "Targets", objs("boss"), "Stat", "attack")),
		"buff huge value":      event(act("Kind", "buff", "Targets", objs("boss"), "Stat", "attack", "Value", 500)),
		"buff negative turns":  event(act("Kind", "buff", "Targets", objs("boss"), "Stat", "attack", "Value", 5, "Turns", -1)),
		"buff unknown target":  event(act("Kind", "buff", "Targets", objs("ghost"), "Stat", "attack", "Value", 5)),
		"buff no target":       event(act("Kind", "buff", "Stat", "attack", "Value", 5)),
		"join nobody":          event(act("Kind", "join")),
		"join an enemy":        event(act("Kind", "join", "Spawns", objs(spawn("Character", "boss")))),
		"join the lord":        event(act("Kind", "join", "Spawns", objs(spawn("Character", "hero")))),
		"join as AI":           event(act("Kind", "join", "Spawns", objs(spawn("AI", true)))),
		"join with a policy":   event(act("Kind", "join", "Spawns", objs(spawn("Policy", "hold")))),
		"join off the map":     event(act("Kind", "join", "Spawns", objs(spawn("X", 99)))),
		"join twice":           event(act("Kind", "join", "Spawns", objs(spawn(), spawn("X", 3)))),
		"move nobody":          event(act("Kind", "move")),
		"move unknown unit":    event(act("Kind", "move", "Moves", objs(obj("Unit", "ghost", "To", at)))),
		"move off the map":     event(act("Kind", "move", "Moves", objs(obj("Unit", "boss", "To", obj("X", 99, "Y", 0))))),
		"empty cutscene":       event(act("Kind", "cutscene")),
		"empty group":          event(act("Kind", "cutscene", "Steps", objs(objs()))),
		"step unknown kind":    event(act("Kind", "cutscene", "Steps", objs(objs(step("Kind", "dance", "Unit", "boss"))))),
		"step unknown unit":    event(act("Kind", "cutscene", "Steps", objs(objs(step("Kind", "face", "Unit", "ghost", "Facing", "NE"))))),
		"step move no cell":    event(act("Kind", "cutscene", "Steps", objs(objs(step("Kind", "move", "Unit", "boss"))))),
		"step play no motion":  event(act("Kind", "cutscene", "Steps", objs(objs(step("Kind", "play", "Unit", "boss", "Motion", "dance"))))),
		"step face bad dir":    event(act("Kind", "cutscene", "Steps", objs(objs(step("Kind", "face", "Unit", "boss", "Facing", "N"))))),
		"step say no text":     event(act("Kind", "cutscene", "Steps", objs(objs(step("Kind", "say", "Unit", "boss"))))),
		"step wait no seconds": event(act("Kind", "cutscene", "Steps", objs(objs(step("Kind", "wait"))))),
		"duel unknown":         event(act("Kind", "duel", "Duel", "nope")),
		"repeat join": func(st map[string]any) {
			event(act("Kind", "join", "Spawns", objs(spawn())))(st)
			st["Events"].([]any)[0].(map[string]any)["Repeat"] = true
		},
	}
	for name, edit := range bad {
		t.Run(name, func(t *testing.T) { assert.Error(t, loadWithStage(t, edit)) })
	}
}
