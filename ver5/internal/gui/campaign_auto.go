package gui

import (
	"context"
	"path/filepath"

	"srpg/internal/autoplay"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

type campaignPlan struct {
	engine   *core.Engine
	revision uint64
	commands []core.Command
	prepared bool
	err      error
}

// campaignPrep prepares the party and answers learning windows in place of the player (the end-to-end test's
// autoplay.Variety).
type campaignPrep interface {
	PrepareCommands(e *core.Engine, try int) []core.Command
	Learn(e *core.Engine) core.Command
}

type campaignAuto struct {
	prep         campaignPrep // when set, preparations and learning are its
	player       *autoplay.Searcher
	model        *autoplay.BattleModel
	cancel       context.CancelFunc
	pending      chan campaignPlan
	queue        []core.Command
	engine       *core.Engine
	revision     uint64
	checkpoint   *core.State
	node         string
	try          int
	prepared     bool
	presented    bool
	ownedOverlay string
	resultAt     float64
}

func (g *Game) campaignActive() bool { return g.campaign != nil }

func (g *Game) stopCampaignAuto() {
	if a := g.campaign; a != nil {
		a.cancel()
		g.campaign = nil
	}
}

func (g *Game) toggleCampaignAuto() {
	if g.campaignActive() {
		g.stopCampaignAuto()
		return
	}
	if !DebugBuild || !g.Debug || g.S.Engine == nil {
		return
	}
	// The planner owns this bundle, so cancelling it also permits editing the GUI's content safely.
	d, err := content.LoadScenario(filepath.Join(g.root, "assets", "data"), g.S.Data.ScenarioID)
	if err != nil || d.Hash != g.S.Data.Hash {
		g.toast("자동실행용 게임 데이터를 읽을 수 없습니다", bad)
		return
	}
	var model *autoplay.BattleModel
	if !g.campaignRule {
		model, err = autoplay.LoadBattleModel(filepath.Join(g.root, "assets", "autoplay", "battle.json"))
		if err != nil || model.ContentHash != d.Hash {
			g.toast("학습 전투 모델을 읽을 수 없습니다. 모델과 게임 데이터를 확인하세요", bad)
			return
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := autoplay.CampaignPlayer(d)
	p.Context = ctx
	g.campaign = &campaignAuto{player: p, model: model, cancel: cancel, try: 1}
	g.autoplay = false
	g.clearOrder()
}

func (g *Game) campaignPolicySlot() *cornerSlot {
	if !DebugBuild || !g.Debug {
		return nil
	}
	label := "학습(시험)"
	if g.campaignRule {
		label = "전투: 탐색"
	}
	slot := &cornerSlot{label: label}
	if !g.campaignActive() {
		slot.click = func() { g.campaignRule = !g.campaignRule }
	}
	return slot
}

func (g *Game) campaignSlot() *cornerSlot {
	if !DebugBuild || !g.Debug {
		return nil
	}
	label := "자동실행"
	if g.campaignActive() {
		label = "자동 중지"
		if g.campaign.pending != nil {
			label = "계산 중지"
		}
	}
	return &cornerSlot{label: label, toggle: true, on: g.campaignActive(), click: g.toggleCampaignAuto}
}

func (g *Game) planCampaign(kind string, actor string) {
	a := g.campaign
	snap := g.S.Engine.Snapshot()
	engine := g.S.Engine
	ch := make(chan campaignPlan, 1)
	a.pending = ch
	try := a.try
	go func() {
		plan := campaignPlan{engine: engine, revision: snap.Revision, prepared: kind == "prepare"}
		e, err := core.Restore(a.player.D, snap)
		if err == nil {
			switch kind {
			case "prepare":
				if a.prep != nil {
					plan.commands = a.prep.PrepareCommands(e, try)
				} else if a.model != nil && a.model.Preparation != nil {
					p := &autoplay.NeuralPlayer{Searcher: a.player, Model: a.model}
					plan.commands = p.PrepareCommands(e, try)
					err = p.AutoplayError()
				} else {
					plan.commands = a.player.PrepareCommands(e, try)
				}
			case "act":
				if a.model != nil {
					plan.commands = []core.Command{a.model.Select(e)}
				} else {
					plan.commands = []core.Command{a.player.Act(e, actor)}
				}
			case "choice":
				plan.commands = []core.Command{autoplay.ChooseStory(e, a.player)}
			case "explore":
				plan.commands = a.player.Explore(e)
			}
			if err == nil {
				err = a.player.Context.Err()
			}
		}
		plan.err = err
		ch <- plan
	}()
}

func (g *Game) queueCampaign(commands []core.Command) {
	a := g.campaign
	a.queue, a.engine, a.revision = commands, g.S.Engine, g.S.View().Revision
	a.presented = false
}

func (g *Game) closeCampaignOverlay() {
	a := g.campaign
	if a.ownedOverlay != "" && g.overlay() == a.ownedOverlay {
		g.party.learning = false
		g.overlays = g.overlays[:len(g.overlays)-1]
	}
	a.ownedOverlay = ""
}

// presentCampaignCommand shows the real window before the AI spends points or changes equipment.
func (g *Game) presentCampaignCommand(c core.Command) {
	a := g.campaign
	switch c.Kind {
	case "learn", "equip", "unequip", "deputy":
		g.selected = c.Actor
		g.party.items, g.party.table = false, false
		switch c.Kind {
		case "learn":
			g.party.tab, g.party.sortieTab, g.party.trait = 1, 1, c.Trait
		case "equip", "unequip":
			g.party.tab, g.party.sortieTab = 2, 2
		case "deputy":
			g.party.sortieTab = 3
			return // the sortie's deputy tab is already on screen
		}
		if g.overlay() == "" {
			g.overlays = append(g.overlays, "party")
		}
		a.ownedOverlay = "party"
	case "item":
		g.selected = c.Target // the promotion itself has its ordinary modal
	case "buy":
		g.shopFrom = c.Target
		g.shopView = shopView{of: c.Target, pick: c.Item, n: c.Count, list: tableList{order: byPrice}}
		if g.overlay() == "" {
			g.overlays = append(g.overlays, "shop")
		}
		a.ownedOverlay = "shop"
	}
}

func (g *Game) retryCampaign() {
	a := g.campaign
	if a.try >= 5 {
		g.stopCampaignAuto()
		g.toast("5회 시도 후 자동실행을 멈췄습니다", bad)
		return
	}
	r := g.request(session.Request{Op: "retry"})
	if !r.OK {
		g.stopCampaignAuto()
		return
	}
	if a.checkpoint == nil {
		snap := g.S.Engine.Snapshot()
		a.checkpoint, a.node = &snap, g.S.View().Node
	}
	{
		snap := *a.checkpoint
		snap.RNG += uint64(a.try) * 0x9e3779b97f4a7c15
		snap.Revision = g.S.View().Revision + 1
		e, err := core.Restore(g.S.Data, snap)
		if err != nil {
			g.stopCampaignAuto()
			g.toast(err.Error(), bad)
			return
		}
		g.S.Engine, g.S.Dirty = e, true
	}
	a.try++
	a.prepared, a.presented = false, false
	a.queue, a.resultAt = nil, 0
}

func (g *Game) updateCampaignAuto() {
	a := g.campaign
	if a == nil {
		return
	}
	if !DebugBuild || !g.Debug || g.S.Screen != "playing" || g.S.Engine == nil {
		g.stopCampaignAuto()
		return
	}
	if a.pending != nil {
		select {
		case plan := <-a.pending:
			a.pending = nil
			if plan.err != nil {
				g.stopCampaignAuto()
				g.toast("자동실행: "+plan.err.Error(), bad)
				return
			}
			if plan.engine != g.S.Engine || plan.revision != g.S.View().Revision {
				return // the user changed the game while this copy was thinking
			}
			g.queueCampaign(plan.commands)
			a.prepared = a.prepared || plan.prepared
		default:
			return
		}
	}
	if g.busy() || g.held != nil || len(g.modals) > 0 {
		if len(g.modals) > 0 && (g.modals[0].t > 1.2 || g.fastTest) {
			g.closeModal()
		}
		return
	}
	o := g.S.View()
	if len(a.queue) > 0 {
		if a.engine != g.S.Engine || a.revision != o.Revision {
			a.queue, a.prepared = nil, false
			g.closeCampaignOverlay()
			return
		}
		c := a.queue[0]
		if !a.presented {
			g.closeCampaignOverlay()
			g.presentCampaignCommand(c)
			a.presented = true
			return
		}
		r := g.request(session.Request{Op: "command", Command: c})
		if !r.OK {
			g.stopCampaignAuto()
			return
		}
		if c.Kind == "walk" {
			g.explore = nil // lay out the area at the position the real walk command reached
		}
		a.queue, a.revision = a.queue[1:], g.S.View().Revision
		a.presented = false
		return
	}
	if a.ownedOverlay != "" {
		g.closeCampaignOverlay()
	}
	if g.overlay() != "" && !g.party.learning {
		return
	}
	if len(o.Learning) > 0 {
		var p campaignPrep = a.player
		if a.prep != nil {
			p = a.prep
		} else if a.model != nil {
			p = &autoplay.NeuralPlayer{Searcher: a.player, Model: a.model}
		}
		g.queueCampaign([]core.Command{p.Learn(g.S.Engine)})
		return
	}
	if g.party.learning {
		g.party.learning = false
		g.overlays = nil
	}
	switch {
	case o.Phase == "complete":
		g.stopCampaignAuto()
	case o.Phase == "result":
		if a.resultAt == 0 {
			a.resultAt = g.uiSeconds
		}
		if g.uiSeconds-a.resultAt < 4 && !g.fastTest {
			return
		}
		if o.Result == "victory" {
			a.resultAt = 0
			g.queueCampaign([]core.Command{{Kind: "continue"}})
		} else {
			g.retryCampaign()
		}
	case g.S.Engine.Walking():
		g.planCampaign("explore", "")
	case o.Phase == "scenario" && o.Dialogue.Kind == "choice":
		g.planCampaign("choice", "")
	case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
		if a.node != o.Node {
			snap := g.S.Engine.Snapshot()
			a.checkpoint, a.node, a.try, a.prepared = &snap, o.Node, 1, false
		}
		if !a.prepared {
			g.planCampaign("prepare", "")
		} else {
			g.queueCampaign([]core.Command{{Kind: "start"}})
		}
	case o.Phase == "deployment":
		commands, err := g.S.Engine.DeploymentPlan()
		if err != nil {
			g.stopCampaignAuto()
			g.toast(err.Error(), bad)
		} else {
			g.queueCampaign(commands)
		}
	case o.Phase == "battle" && o.Turn != "ally":
		g.resolvePhase()
	case o.Phase == "battle":
		for _, u := range o.UnitViews {
			if u.Faction == "ally" && !u.AI && u.HP > 0 && !u.Done {
				g.planCampaign("act", u.ID)
				return
			}
		}
		g.queueCampaign([]core.Command{{Kind: "end"}})
	}
}
