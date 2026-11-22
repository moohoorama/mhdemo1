package gui

import (
	"fmt"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["story-farewell"] = demoStoryFarewell }

// demoStoryFarewell shows 장비's notice, the walk, deputy appointment and 조운's equipment before the farewell.
func demoStoryFarewell(d *Demo) error {
	level := d.g.S.Data.Rules.PromoteLevels[0]
	if err := d.demoJumpAtLevel("c13-start", level); err != nil {
		return err
	}
	g := d.g
	for g.S.View().Node != "c13-x-notice" {
		if _, err := g.S.Engine.Apply(core.Command{Kind: "next"}); err != nil {
			return fmt.Errorf("reach the farewell notice from %s: %w", g.S.View().Node, err)
		}
	}
	g.intro = nil
	g.S.Settings.InstantText = true
	d.playGame = true
	d.step = func(f int) {
		switch f {
		case 115:
			g.advanceDialogue()
		case 195:
			r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: "liubei", Trait: "deputy_appoint"}})
			if !r.OK {
				d.err = fmt.Errorf("teach 부관임명: %s", r.Error)
				return
			}
			g.selected = "liubei"
			g.pressLabel("관리")
		case 215:
			g.pressLabel("부관")
		case 230:
			g.deputyPick = "jianyong"
		case 270:
			g.pressLabel("부관 임명")
		case 310:
			g.selected = "zhaoyun"
			g.pressLabel("장비")
		case 350:
			g.pressLabel("닫기")
		case 365:
			if g.exploring() {
				d.err = g.demoExploreAct("talk:조운")
			}
		}
	}
	return nil
}
