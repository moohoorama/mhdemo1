package gui

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/duel"
	"srpg/internal/session"
	"srpg/internal/sprite"
)

func TestDuelCues(t *testing.T) {
	story := duel.Story{Beats: []duel.Beat{
		{Kind: "talk", Duration: 1},
		{Kind: "block", Side: 1, Duration: 1},
		{Kind: "rush-critical", Side: 0, Duration: 1.2},
		{Kind: "dodge", Side: 1, Duration: 1},
		{Kind: "shoot", Side: 1, Duration: 1.2},
		{Kind: "win", Side: 0, Duration: 3},
	}}
	t.Run("every exchange in order", func(t *testing.T) {
		assert.Equal(t, []string{"swing", "block", "swing-crit", "crit", "swing", "shoot", "crit", "duel-fall"}, duelCues(story, -1, story.Duration()))
	})
	t.Run("a step hears only what it passed", func(t *testing.T) {
		assert.Equal(t, []string{"swing"}, duelCues(story, 1.1, 1.3))
		assert.Equal(t, []string{"block"}, duelCues(story, 1.3, 1.6))
		assert.Empty(t, duelCues(story, 1.6, 1.6))
	})
}

func TestStrikeCues(t *testing.T) {
	t.Run("a critical hit, a hit and a block, each once", func(t *testing.T) {
		events := []core.Event{
			{Kind: "critical", Target: "a"}, {Kind: "damage", Target: "a"},
			{Kind: "damage", Target: "b"}, {Kind: "damage", Target: "c"}, {Kind: "miss", Target: "d"},
		}
		assert.Equal(t, []string{"crit", "hit", "block"}, strikeCues(events))
	})
	t.Run("no blow, no sound", func(t *testing.T) {
		assert.Empty(t, strikeCues([]core.Event{{Kind: "heal", Target: "a"}}))
	})
}

func TestLoudestAt(t *testing.T) {
	pcm := make([]byte, 4*sampleRate/2) // half a second of silence with a click at 0.3 s
	for j := sampleRate * 3 / 10; j < sampleRate*3/10+100; j++ {
		pcm[4*j+1], pcm[4*j+3] = 0x40, 0x40
	}
	assert.InDelta(t, .3, loudestAt(pcm), .02)
}

func TestSfxAt(t *testing.T) {
	g := &Game{S: &session.Session{}, sound: &soundBank{peak: map[string]float64{"hit": .1, "fire": .5}}}
	g.S.Settings.Sound = "off"
	p := &Player{g: g}
	p.sfxAt("hit", .4)
	p.sfxAt("fire", .3) // needs longer than it has: at once
	require.Len(t, p.sounds, 1)
	assert.InDelta(t, .3, p.sounds[0].left, 1e-9)
	p.updateSounds(.29)
	assert.Len(t, p.sounds, 1)
	p.updateSounds(.02)
	assert.Empty(t, p.sounds)
}

func TestRiderArt(t *testing.T) {
	g := &Game{S: &session.Session{Data: &content.Data{
		Classes: map[string]content.Class{
			"light_cavalry":  {Sprite: "light_cavalry", Tags: []string{content.TagMounted}},
			"light_infantry": {Sprite: "light_infantry"},
		},
		Characters: map[string]content.Character{
			"zhangfei": {Look: content.Look{Sprites: map[string]string{"light_cavalry": "zhangfei"}}},
			"zhaoyun":  {Look: content.Look{Sprites: map[string]string{"light_cavalry:sword": "zhaoyun_sword"}}},
			"liubei":   {Look: content.Look{Sprites: map[string]string{"light_infantry": "liubei"}}},
		},
	}}}
	for id, want := range map[string]bool{
		"light_cavalry": true, "light_cavalry_white": true, "liubei_mounted": true, "horse_archer": true,
		"zhangfei": true, "zhaoyun_sword": true,
		"light_infantry": false, "liubei": false, "liubei_noncombat": false,
	} {
		assert.Equal(t, want, g.riderArt(id), id)
	}
}

func TestMeritSounds(t *testing.T) {
	newGame := func() *Game {
		g := &Game{S: &session.Session{}}
		g.S.Settings.Sound = "off"
		return g
	}
	t.Run("each card once as it lands, then the spoils", func(t *testing.T) {
		g := newGame()
		for f := 0; f <= 200; f++ {
			g.meritSounds(2, float64(f)/60)
		}
		assert.Equal(t, 3, g.sfxCalls) // gold, silver, coins
	})
	t.Run("the other officers' lines when there are more than three", func(t *testing.T) {
		g := newGame()
		g.meritSounds(5, 3)
		assert.Equal(t, 5, g.sfxCalls)
	})
	t.Run("a skipped show is silent", func(t *testing.T) {
		g := newGame()
		g.meritSounds(5, .1)
		g.resultSkip = true
		g.meritSounds(5, 99)
		assert.Zero(t, g.sfxCalls)
	})
}

func TestFallCue(t *testing.T) {
	g := &Game{S: &session.Session{Data: &content.Data{Characters: map[string]content.Character{
		"guanyu": {}, "lubu": {}, "troop_footman": {Template: true},
	}}}}
	unit := func(id, character, faction, rank string) core.UnitView {
		return core.UnitView{Unit: core.Unit{ID: id, Character: character, Faction: faction, Rank: rank}}
	}
	g.player.final.UnitViews = []core.UnitView{
		unit("guanyu", "guanyu", "ally", ""), unit("a1", "troop_footman", "ally", ""),
		unit("lubu", "lubu", "enemy", ""), unit("e1", "troop_footman", "enemy", "commander"),
		unit("e2", "troop_footman", "enemy", ""), unit("cart", "loot_cart", "enemy", ""),
	}
	for id, want := range map[string]string{
		"guanyu": "ally-fall", "a1": "fall", "lubu": "leader-fall", "e1": "leader-fall", "e2": "fall", "cart": "fall",
		"nobody": "fall",
	} {
		assert.Equal(t, want, g.fallCue(id), id)
	}
}

func TestSceneMotionCue(t *testing.T) {
	g := &Game{S: &session.Session{Data: &content.Data{
		Classes: map[string]content.Class{
			"light_cavalry": {Sprite: "light_cavalry", Attack: content.AttackMelee},
			"horse_archer":  {Sprite: "horse_archer", Attack: content.AttackRanged},
			"archer":        {Sprite: "archer", Attack: content.AttackRanged},
		},
		Characters: map[string]content.Character{
			"lubu":     {Look: content.Look{Sprites: map[string]string{"horse_archer": "lubu_bow"}}},
			"zhangfei": {Look: content.Look{Sprites: map[string]string{"light_cavalry": "zhangfei"}}},
		},
	}}}
	for _, c := range []struct{ art, anim, want string }{
		{"guanyu", "attack", "swing"}, {"lubu_bow", "attack", "shoot"}, {"horse_archer", "attack", "shoot"},
		{"archer", "attack", "shoot"}, {"archer_white", "attack", "shoot"}, {"light_cavalry_white", "attack", "swing"},
		{"zhangfei", "attack", "swing"}, {"zhangfei", "hit", "hit"}, {"zhangfei", "block", "block"}, {"zhangfei", "action", ""},
	} {
		v := &unitVis{art: &sprite.Art{ID: c.art}}
		assert.Equal(t, c.want, g.sceneMotionCue(v, c.anim), c.art+" "+c.anim)
	}
}

func TestNoteSubs(t *testing.T) {
	g := &Game{S: &session.Session{}}
	g.S.Settings.Sound = "off"
	o := core.Observation{State: core.State{Phase: "battle", Round: 2}, Map: &content.Stage{ID: "s07", Sub: []content.Objective{
		{Text: "여포를 퇴각시킨다", If: []string{"defeated:lubu"}}, {Text: "손견을 지킨다", If: []string{"alive:sunjian"}},
	}}}
	g.noteSubs(o, []bool{true, true})
	require.Zero(t, g.sfxCalls, "goals already met when the battle is first seen are silent")

	g.subsMet = subWatch{}
	g.noteSubs(o, []bool{false, true})
	g.noteSubs(o, []bool{false, true})
	assert.Zero(t, g.sfxCalls, "a goal that keeps someone alive is silent")
	g.noteSubs(o, []bool{true, true})
	assert.Equal(t, 1, g.sfxCalls)
	assert.Len(t, g.toasts, 1)
	g.noteSubs(o, []bool{true, true})
	assert.Equal(t, 1, g.sfxCalls, "once")

	o.Round = 1 // a retry
	g.noteSubs(o, []bool{false, true})
	o.Round = 3
	g.noteSubs(o, []bool{true, true})
	assert.Equal(t, 2, g.sfxCalls, "sounds again in the retried battle")
}

func TestDistanceGain(t *testing.T) {
	assert.Equal(t, 1.0, distanceGain(0, 0))
	assert.Equal(t, 1.0, distanceGain(.2, .2))
	assert.InDelta(t, .2, distanceGain(1, 0), 1e-9, "a fifth at the screen's edge")
	assert.InDelta(t, .2, distanceGain(0, -1), 1e-9)
	assert.Less(t, distanceGain(.8, 0), distanceGain(.5, 0))
	assert.Zero(t, distanceGain(2, 0), "silent well off the screen")
}

func TestEverySkillSoundIsAFile(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for id, s := range d.Skills {
		for _, cue := range []string{s.Sound, s.Lead} {
			if cue == "" {
				continue
			}
			_, plain := os.Stat("../../assets/sounds/" + cue + ".ogg")
			_, variant := os.Stat("../../assets/sounds/" + cue + "-1.ogg")
			assert.True(t, plain == nil || variant == nil, "%s: no sound %s", id, cue)
		}
	}
}
