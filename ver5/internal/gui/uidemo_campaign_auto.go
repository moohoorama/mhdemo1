package gui

import (
	"fmt"
	"os"

	"srpg/internal/session"
)

func init() { demos["campaign-auto"] = demoCampaignAuto }

// demoCampaignAuto starts a normal hard campaign and drives the same controller as srpg-gui.
func demoCampaignAuto(d *Demo) error {
	g := d.g
	g.Debug = DebugBuild
	if r := g.request(session.Request{Op: "new", Seed: 12, Difficulty: "hard"}); !r.OK {
		return fmt.Errorf("new campaign: %s", r.Error)
	}
	g.toggleCampaignAuto()
	if !g.campaignActive() {
		return fmt.Errorf("campaign auto did not start (check battle model)")
	}
	fmt.Fprintf(os.Stderr, "campaign-auto: neural=%v preparation=%v\n", g.campaign.model != nil, g.campaign.model != nil && g.campaign.model.Preparation != nil)
	g.speed = 8
	d.playGame = true
	last := ""
	loggedSpecialist := false
	d.step = func(int) {
		o := g.S.View()
		if !loggedSpecialist && o.Phase == "battle" && g.campaign != nil && g.campaign.model != nil {
			id := g.S.Data.Stages[o.Stage].ID
			if model := g.campaign.model.StageModels[id]; model != nil {
				fmt.Fprintf(os.Stderr, "campaign-auto: %s specialist episodes=%d objective=%s\n", id, model.Episodes, model.Objective)
			}
			loggedSpecialist = true
		}
		key := fmt.Sprintf("%s %s %s round %d", o.Phase, o.Node, o.Turn, o.Round)
		if key != last {
			fmt.Fprintln(os.Stderr, "campaign-auto:", key)
			last = key
		}
	}
	return nil
}
