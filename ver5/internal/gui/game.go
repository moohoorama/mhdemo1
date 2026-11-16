// Package gui renders the same Session used by the CLI. All mutations cross
// Session.Handle with a revision; the renderer only consumes copied DTOs.
// The battlefield is the ver3 isometric map; every outcome is shown by claim replay
// (design.md 0.9): AI phases are resolved first and replayed with independent actions
// overlapping, a player's command is a one-action replay. Input follows 조조전:
// a unit moves tentatively, then an action menu opens beside it (command.go).
package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"srpg/internal/ai"
	"srpg/internal/cli"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/replay"
	"srpg/internal/scenes"
	"srpg/internal/session"
	"srpg/internal/sprite"
)

const Width, Height = 1280, 850

type Game struct {
	layer           int
	sound           *soundBank // nil: silent (EnableSound)
	tracedTheme     string     // the theme last written to sfxTrace
	sfxCalls        int        // cues asked for so far: a plain click sounds only when what it did made none
	musicTab        *musicTable
	duelPreview     func() error
	previewIdle     int
	previewRate     float64
	previewPaused   bool
	duelMap, duelUI *ebiten.Image
	// fixedCanvas holds a fixed-layout screen (map editor, duel gallery) before it is centred
	fixedCanvas               *ebiten.Image
	frames                    map[image.Point]*ebiten.Image // inFrame's images, by size
	windows                   map[string]*windowAnim        // popup windows opening, open or closing
	shown                     []*windowAnim                 // windows laid out this frame, in order
	duelCam                   [2]float64
	duelCamHeld               bool
	cutIn                     duelCutIn
	cutPanes                  [2]*ebiten.Image
	gallery                   *duelGallery
	editor                    *mapEditor
	editorReturn              *mapEditor
	editorSession             *session.Session
	deploymentSelected        string
	sortieTried               string // the battle node startSortie last tried to leave for its deployment
	edgeDir                   int    // the edge_arrow frame under the cursor while edgeOn (edge_scroll.go)
	edgeOn                    bool
	deputyPick                string    // the candidate the deputy window previews
	deputyInfo                equipChip // the deputy window's trait badge last pressed
	edgeHeld                  float64   // seconds the left button has been held on an edge arrow
	textSeconds               float64   // the clock lines come out on: uiSeconds run at the replay speed
	cursorHidden              bool
	uiSeconds                 float64
	moneyShown                int  // the money as the screen counts it toward the real sum
	moneyReady                bool // moneyShown has been set from a game
	textCache                 map[string]*ebiten.Image // rendered text by key (ui_anim.go)
	root                      string
	assets                    *Assets
	field                     *Field
	vis                       map[string]*unitVis
	popups                    []*popup
	effects                   []*effect
	lobs                      []*Lob
	releases                  map[string]map[string][2]float64 // sprite id -> facing -> where its thrown stone leaves
	player                    Player
	banner                    string
	turnBanner                *turnBanner
	openingTurn               bool // the first turn's banner waits for the battle's opening to finish
	bannerT, clockMS          float64
	shakeT                    float64
	groundFX                  bool // spell looks lie in the field's ground marks (groundSpells)
	hitFlashT                 float64
	hitFlashCrit              bool
	camX, camY                float64
	pan                       *camPan
	follow                    string // the unit whose walk the camera eases after during a replay (camera_pan.go)
	zoom                      float64
	zoomFrom, zoomTo          float64
	zoomT                     float64
	speed                     int
	dragging, dragMoved       bool
	pinched                   bool // two fingers are zooming the battle; it settles on a step of zooms when they lift
	dragX, dragY              int
	dragButton                ebiten.MouseButton
	shownNode                 string
	held                      *core.Observation // the story as it stood when a gift's window opened over it, shown until it closes
	lineIndex                 int
	lineTyping                typing // how the current line is coming out
	dialogueAnim              dialogueAnim
	narrationLeaving          bool
	narrationExitAt           float64
	autoKey                   string        // the line the auto setting is waiting on
	autoAt                    float64       // when that line was out
	skipping                  *scenes.Scene // the scene the skip button is paging through
	skipUntil                 float64       // clicks wait until then after a skip stops
	Verify                    bool
	Debug                     bool  // the tools for checking the game: duels, the map editor, damage and XP scales
	Shots                     []int // capture the screen at these ticks while autoplaying, then quit
	verified                  map[string]bool
	verifyDone                int
	S                         *session.Session
	font                      *text.GoTextFaceSource
	brushFont, serifFont      *text.GoTextFaceSource // display faces beside the body font
	buttons                   []button
	buttonClip                image.Rectangle // when set, buttons added are cut to it (a scrolled area)
	blocks                    []image.Rectangle
	wheelAreas                []wheelArea
	unitWin                   unitWindowState
	overlayActs               []action // the actions an older window's content puts beside 닫기
	ord                       order
	menuAnim, subAnim         panelAnim
	hover                     hoverCell
	threatAll                 bool
	threats                   map[string]core.Ranges
	itemEffects               map[string]itemEffect // itemEffect results by revision, officer, slot and piece
	equip                     equipState
	waypoints                 waypointList
	jumpNode                  string // the waypoint the difficulty window starts at; none starts a new game
	question, yes, no         string
	confirmTitle              string
	answer                    func()
	endAsked                  string          // the battle turn whose "all allies ordered" prompt was shown
	hudRect, hudBody          image.Rectangle // the HUD panel and its content as last drawn
	hudInfo                   image.Rectangle // the info panel at the bottom right as last drawn
	hudFolded, hudAnimated    bool            // the panel (or the right corner panel) shows its head only; it has been drawn before
	leftFolded                bool            // the left corner panel shows its head only
	party                     partyView       // the party window's tab and what it shows
	hudH, hudAlpha, hudAnimAt float64         // its drawn height and opacity, and when they last eased
	hudLayer                  *ebiten.Image   // where it is drawn while faded
	trans                     *transition     // the screen change fading through black
	shownScreen               string          // the screen the last update showed (screenKey)
	lastFrame                 *ebiten.Image   // the frame drawn last, held still when a change darkens it
	darkness                  float64         // how black the world went in the frame drawn last
	hudShown                  string          // the tab it showed
	hudBase, hudPeek          string          // the tab chosen by hand; the one looked at while an officer is selected
	unitsList, rosterList     unitList        // the HUD tab's unit list and the roster window's
	hudTurn, hudBefore        string          // the side that moved last, and the tab to return to after the enemy's turn
	traitOf                   string
	traitTop                  int
	detailTab                 int
	detailPage                int
	toasts                    []*toast
	history                   []string
	logPending                []string
	logStage                  string
	logThumbDragging          bool
	logThumbOffset            int
	logHeight, logScroll      int
	logDragging               bool
	logDragOffset             int
	recoveryEngine            *core.Engine
	recoveryRevision          uint64
	recoveryID                string
	recovery                  core.RecoveryPreview
	sandboxView               core.ObservationCache
	mini                      miniGround
	modals                    []*modal
	scenes                    *scenes.Book
	names                     map[string]string // display name -> character id
	learn                     []core.Learner    // learning windows open now, in order
	shopView                  shopView
	tableDrag                 *tableDrag // the press held on a table (table.go)
	scene                     *scene
	tour                      *tourWalk    // the walk round the format 2 scene maps (tour.go, cmd/_town2)
	world                     *worldWalk   // the walk over the world map (worldmap.go, cmd/_cnmap)
	explore                   *exploreWalk // the story's walk between battles (explore.go)
	shopFrom                  string       // the person of the walk whose shop is open
	intro                     *officerIntro
	introImg                  introImages
	resultSkip                bool    // the victory window's show was clicked through
	meritHeard                float64 // seconds into the victory window's show whose sounds have played
	subsMet                   subWatch
	saves                     saveList
	selected                  string
	overlays                  []string // the open windows, the top one last
	notice, entry             string
	page, slot, ticks         int
	input, autoplay           bool
	capture                   bool
	seed                      uint64
	log                       *os.File
	auditDir                  string
	gear                      gearGuide // the green guide to wearing the pieces an event of the walk handed over
}

func loadFont(path string) (*text.GoTextFaceSource, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return text.NewGoTextFaceSource(f)
}

func New(s *session.Session, fontPath string, seed uint64, auditDir string) (*Game, error) {
	source, err := loadFont(fontPath)
	if err != nil {
		return nil, err
	}
	brush, err := loadFont(filepath.Join(filepath.Dir(fontPath), "NanumBrushScript.ttf"))
	if err != nil {
		return nil, err
	}
	serif, err := loadFont(filepath.Join(filepath.Dir(fontPath), "NotoSerifKR.ttf"))
	if err != nil {
		return nil, err
	}
	g := &Game{S: s, font: source, brushFont: brush, serifFont: serif, seed: seed, verified: map[string]bool{}, slot: 1, auditDir: auditDir,
		vis: map[string]*unitVis{}, zoom: 1, zoomFrom: 1, zoomTo: 1, zoomT: .5, speed: savedSpeed(s.Settings.Speed), threats: map[string]core.Ranges{}, itemEffects: map[string]itemEffect{}}
	g.root = filepath.Dir(filepath.Dir(filepath.Dir(fontPath)))
	g.player.g = g
	g.names = nameIndex(s.Data)
	g.scenes, _ = scenes.Load(filepath.Join(filepath.Dir(fontPath), "..", "scenes"))
	g.assets, err = LoadAssets(filepath.Join(filepath.Dir(fontPath), "..", "graphics"), source, s.Data)
	if err != nil {
		return nil, err
	}
	if err := g.validateUIContent(); err != nil {
		return nil, err
	}
	for id, script := range g.assets.Duels {
		for _, actor := range script.Supports {
			if _, ok := s.Data.Characters[actor]; !ok {
				return nil, fmt.Errorf("duel %s: unknown support %q", id, actor)
			}
		}
	}

	if auditDir != "" {
		if err = os.MkdirAll(auditDir, 0755); err != nil {
			return nil, err
		}
		g.log, err = os.Create(filepath.Join(auditDir, "gui-input.jsonl"))
		if err != nil {
			return nil, err
		}
	}
	return g, nil
}
func (g *Game) Close() {
	fields := []*Field{g.field}
	if g.editor != nil {
		fields = append(fields, g.editor.field)
	}
	if g.editorReturn != nil {
		fields = append(fields, g.editorReturn.field)
	}
	seen := map[*Field]bool{}
	for _, f := range fields {
		if f != nil && !seen[f] {
			f.dispose()
			seen[f] = true
		}
	}
	if g.log != nil {
		g.log.Close()
	}
}

// send passes one request to the session: logging, confirmations, autosave and the
// battle log. It does not replay anything.
func (g *Game) send(q session.Request) session.Response {
	var before *core.Observation
	c := q.Command
	if g.S.Engine != nil {
		o := g.S.View()
		before = &o
		q.Revision = &o.Revision
	}
	r := g.S.Handle(q)
	if g.log != nil {
		_ = json.NewEncoder(g.log).Encode(struct {
			Time     string
			Request  session.Request
			Response session.Response
		}{time.Now().UTC().Format(time.RFC3339Nano), q, r})
	}
	if !r.OK {
		switch r.Error {
		case "OverwriteRequired":
			g.ask("기존 저장을 덮어쓰시겠습니까?", func() { q.Overwrite = true; g.request(q) })
		case "DiscardRequired":
			g.ask("저장하지 않은 진행을 버리시겠습니까?", func() { q.Discard = true; g.request(q) })
		case "ChangedContent":
			g.ask("게임 데이터가 바뀌기 전의 저장입니다.\n그래도 불러오시겠습니까?", func() { q.Changed = true; g.request(q) })
		default:
			g.notice = r.Error
			g.toast(koreanError(r.Error), bad)
		}
		return r
	}
	switch q.Op {
	case "load", "retry", "new", "title":
		g.player.Skip()
		g.vis = map[string]*unitVis{}
		g.popups, g.effects, g.modals = nil, nil, nil
		g.closeOverlays()
		g.clearOrder()
		g.selected, g.endAsked = "", ""
		if g.field != nil {
			g.field.dispose()
		}
		g.field, g.scene, g.intro = nil, nil, nil
		g.resetBattleLog()
		g.resetDeployment()
	}
	g.notice = "완료"
	switch q.Op {
	case "save":
		g.overlays = nil // the save's bell, not the window's closing
		g.sfx("save")
		g.toast(fmt.Sprintf("슬롯 %s 저장 완료", q.Slot), gold)
	case "load":
		g.toast("불러오기 완료", gold)
	}
	if q.Op == "command" && g.selected == c.Actor {
		switch c.Kind {
		case "attack", "skill", "item", "wait":
			g.selected = ""
		}
	}
	g.logCommand(c, r.Events)
	g.record(before, r.Events)
	return r
}

// request is send plus the replay of a battle command.
func (g *Game) request(q session.Request) session.Response {
	var before *core.Observation
	if g.S.Engine != nil {
		o := g.S.View()
		before = &o
	}
	r := g.send(q)
	if r.OK && before != nil && before.Phase == "battle" && q.Op == "command" {
		g.replayOne(*before, q.Command, r.Events)
	}
	if r.OK && g.S.Engine != nil && before != nil && (before.Phase == "deployment" || g.S.Engine.Deploying()) {
		if before.Phase != "deployment" || !g.S.Engine.Deploying() {
			g.resetDeployment()
		}
		if before.Phase != "deployment" && g.S.Engine.Deploying() {
			g.placeSortie()
		}
		g.sync(g.S.View())
	}
	if r.OK && before != nil && before.Phase != "battle" && q.Op == "command" && g.S.View().Phase == "battle" {
		g.sfx("sortie")
		g.replayOpening(q.Command, r.Events)
		g.openingTurn = true
	}
	return r
}

// replayOpening plays what the battle's first turn brought with it (a cutscene, a fire, lines) when
// the command that began the battle (start, the last deployment) also set it off: the units stand
// where their first walk begins, and the replay takes them from there.
func (g *Game) replayOpening(c core.Command, events []core.Event) {
	final := g.S.View()
	shown := false
	for _, e := range events {
		switch e.Kind {
		case "cutscene", "move", "speech", "terrain", "duel", "unit-arrival":
			shown = true
		}
	}
	if !shown || g.mapArt(final.Map.ID) == nil {
		return
	}
	g.ensureField(final)
	g.sync(final)
	moved := map[string]bool{}
	rewind := func(id string, path []content.Point) {
		if v := g.vis[id]; v != nil && len(path) > 0 && !moved[id] {
			moved[id] = true
			v.u, v.v = float64(path[0].X), float64(path[0].Y)
			v.lift = g.field.Lift(path[0].X, path[0].Y)
		}
	}
	for _, e := range events {
		switch e.Kind {
		case "unit-arrival":
			delete(g.vis, e.Actor) // comes in at its arrival beat, not before the opening plays
		case "move":
			rewind(e.Actor, e.Path)
		case "cutscene":
			for _, group := range e.Cutscene {
				for _, st := range group {
					if st.Kind == "move" {
						rewind(st.Unit, st.Path)
					}
				}
			}
		}
	}
	g.replayOne(final, c, events)
}

// record turns events into log lines, announcements, level-up and acquire windows and battle XP.
func (g *Game) record(before *core.Observation, events []core.Event) {
	levels := map[string][2]int{}
	order := []string{}
	for _, e := range events {
		line, important := g.eventText(e)
		if e.Kind == "level" {
			l, ok := levels[e.Actor]
			if !ok {
				l[0] = e.Amount - 1
				order = append(order, e.Actor)
			}
			l[1] = e.Amount
			levels[e.Actor] = l
		}
		if line == "" || !important && !g.S.Settings.Detail {
			continue
		}
		g.notice = line
	}
	for _, id := range order {
		l := levels[id]
		up := levelUp{id: id, from: l[0], to: l[1]}
		if after, err := g.S.Engine.CharacterStats(id); err == nil {
			up.before, up.after = after, after
		}
		o := g.S.View()
		if u := g.unitView(o, id); u != nil {
			up.after, up.progress = u.Stats, u.XPProgress
		}
		for _, c := range o.Characters {
			if c.ID == id && up.progress == 0 {
				up.progress = core.ExperienceProgress(c.XP)
			}
		}
		if before != nil {
			if u := g.unitView(*before, id); u != nil {
				up.before = u.Stats
			}
		}
		g.modals = append(g.modals, &modal{kind: "levelup", level: up})
		g.sfx("levelup")
	}
	for _, e := range events {
		if e.Kind == "promotion" {
			id := e.Actor
			if e.Text != "" {
				id = e.Text // the officer or the deputy riding with it
			}
			g.modals = append(g.modals, &modal{kind: "promotion", promo: g.promotionUp(before, id)})
			if before == nil || before.Phase != "battle" { // on the field the promotion's own show has sounded it
				g.sfx("promote")
			}
		}
	}
	g.noteGearGrants(events)
	if gains := acquireGains(before, events); len(gains) > 0 {
		g.modals = append(g.modals, &modal{kind: "acquire", gains: gains})
		if before != nil && before.Phase == "scenario" && g.held == nil {
			g.held = before
		}
	}
}

// promotionUp is the promotion window's content for unit id: its class and look before the events, and after.
func (g *Game) promotionUp(before *core.Observation, id string) promotionUp {
	return g.promotionBetween(before, g.S.View(), id)
}

// promotionBetween is unit id's promotion from observation before to after; a deputy, who has no unit,
// is shown by its own record and stats.
func (g *Game) promotionBetween(before *core.Observation, after core.Observation, id string) promotionUp {
	p := promotionUp{id: id}
	if g.unitView(after, id) == nil {
		return g.characterPromotion(before, after, id)
	}
	if u := g.unitView(after, id); u != nil {
		p.to, p.after, p.new = u.Class, u.Stats, g.battleClassArt(u.Def, u.Class, u.Weapon, u.Mount, u.Color)
		p.from, p.before, p.old = p.to, p.after, p.new
	}
	if before != nil {
		if u := g.unitView(*before, id); u != nil {
			p.from, p.before, p.old = u.Class, u.Stats, g.battleClassArt(u.Def, u.Class, u.Weapon, u.Mount, u.Color)
		}
	}
	return p
}

// characterRecord is the character's record in observation o, or nil.
func (g *Game) characterRecord(o core.Observation, id string) *core.Character {
	for i := range o.Characters {
		if o.Characters[i].ID == id {
			return &o.Characters[i]
		}
	}
	return nil
}

// mountOf is the horse a mounted character rides, or "".
func (g *Game) mountOf(ch core.Character) string {
	if slices.Contains(g.S.Data.Classes[ch.Class].Tags, content.TagMounted) {
		return ch.Equipment["accessory"]
	}
	return ""
}

// characterPromotion is the promotion of a character with no unit on the field (a deputy).
func (g *Game) characterPromotion(before *core.Observation, after core.Observation, id string) promotionUp {
	p := promotionUp{id: id}
	ch := g.characterRecord(after, id)
	if ch == nil {
		return p
	}
	p.from, p.to = ch.Class, ch.Class
	if before != nil {
		if b := g.characterRecord(*before, id); b != nil {
			p.from = b.Class
		}
	}
	art := func(class string) *sprite.Art {
		c := *ch
		c.Class = class
		return g.battleClassArt(ch.Def, class, g.S.Data.Equipment[ch.Equipment["weapon"]].Type, g.mountOf(c), g.S.Data.Characters[ch.Def].Look.Faction)
	}
	p.old, p.new = art(p.from), art(p.to)
	p.before, _ = g.engine().CharacterStatsAs(id, p.from)
	p.after, _ = g.engine().CharacterStatsAs(id, p.to)
	return p
}

// replayOne shows a single applied command (the player's, or one AI step) as a one-action replay.
func (g *Game) replayOne(before core.Observation, c core.Command, events []core.Event) {
	a := replay.New(before, c.Actor)
	a.Add(c, events)
	actions := []*replay.Action{a}
	replay.Link(actions)
	g.player.Play(actions, g.S.View())
}

// resolvePhase runs the current faction's whole phase with the AI and replays it.
func (g *Game) resolvePhase() {
	g.clearOrder()
	g.selected = ""
	before := g.S.View()
	actions, err := g.S.ResolvePhase()
	if err != nil {
		g.toast(err.Error(), bad)
	}
	if g.log != nil {
		_ = json.NewEncoder(g.log).Encode(struct {
			Time    string
			Op      string
			Actions []*replay.Action
			Error   string `json:",omitempty"`
		}{time.Now().UTC().Format(time.RFC3339Nano), "resolve-phase", actions, errorText(err)})
	}
	events := []core.Event{}
	for _, a := range actions {
		for _, s := range a.Steps {
			events = append(events, s.Events...)
		}
	}
	g.record(&before, events)
	g.player.PlayAI(before, actions, g.S.View())
}

func (g *Game) command(c core.Command) {
	if g.busy() {
		return
	}
	g.request(session.Request{Op: "command", Command: c})
	g.page = 0
}

// assist plays one recommended command (Space, autoplay and the audit runs).
func (g *Game) assist() {
	if g.busy() || g.S.Engine == nil {
		return
	}
	g.clearOrder()
	if c, ok := ai.Learning(g.S.Engine); ok {
		g.command(c)
		return
	}
	o := g.S.View()
	if o.Phase == "scenario" {
		switch o.Dialogue.Kind {
		case "say", "narrate":
			g.command(core.Command{Kind: "next"})
		case "choice":
			for _, key := range sorted(o.Dialogue.Choices) {
				g.command(core.Command{Kind: "choose", Option: key})
				break
			}
		case "battle":
			if c, ok := g.S.Engine.OwnerPiece(); ok && (g.autoplay || g.auditDir != "") { // Space leaves the gear to the player
				g.command(c)
				return
			}
			g.command(core.Command{Kind: "start"})
		case "explore":
			if plan := g.S.Engine.ExploreNext(); len(plan) > 0 {
				g.command(plan[0])
			}
		}
		return
	}
	if o.Phase == "result" {
		if o.Result == "victory" {
			g.command(core.Command{Kind: "continue"})
		} else {
			g.autoplay = false
		}
		return
	}
	if o.Phase == "deployment" {
		c, err := ai.Next(g.S.Engine)
		if err != nil {
			g.toast(koreanError(err.Error()), bad)
			g.autoplay = false
			return
		}
		g.command(c)
		return
	}
	if o.Phase != "battle" {
		g.autoplay = false
		return
	}
	if o.Turn != "ally" || g.autoplay {
		g.resolvePhase()
		return
	}
	for _, u := range o.UnitViews {
		if u.Faction == "ally" && !u.AI && u.HP > 0 && !u.Done {
			c, err := ai.Choose(g.S.Engine, u.ID)
			if err != nil {
				g.toast(err.Error(), bad)
				return
			}
			g.command(c)
			return
		}
	}
	g.command(core.Command{Kind: "end"})
}

// nextSpeed is the replay speed the F key steps to.
var nextSpeed = map[int]int{1: 2, 2: 4, 4: 8, 8: 1}

// savedSpeed is the settings' replay speed, ×1 when none or not one of the steps.
func savedSpeed(n int) int {
	if _, ok := nextSpeed[n]; ok {
		return n
	}
	return 1
}

// setSpeed sets the replay speed and keeps it in the settings, so it holds across saves and restarts.
func (g *Game) setSpeed(n int) {
	g.speed = n
	v := g.S.Settings
	v.Speed = n
	g.request(session.Request{Op: "settings", Settings: &v})
}

func (g *Game) openObjective() { g.modals = append(g.modals, &modal{kind: "objective", manual: true}) }

func sorted[V any](m map[string]V) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// rightClick cancels the current window or order step, else clears the selection.
func (g *Game) rightClick() {
	switch {
	case g.overlay() != "":
		g.closeOverlay()
	case g.popupPending():
	case g.deploying():
		g.deploymentSelected = ""
	case g.cancel():
	default:
		g.selected = ""
	}
}

func (g *Game) Update() error {
	g.updatePointer()
	g.ticks++
	g.updateMusic(1.0 / 60)
	defer g.watchScreen()
	if g.holdForTransition() {
		return nil
	}
	if g.gallery != nil {
		return g.updateDuelGallery()
	}
	if g.editor != nil {
		return g.updateEditor()
	}
	if g.tour != nil {
		return g.updateTour()
	}
	if g.world != nil {
		return g.updateWorld()
	}
	g.advance(1.0 / 60)
	g.tickMoney()
	g.turnPage()
	g.syncBattleLogStage()
	g.learn = nil
	if g.S.Screen == "playing" {
		g.learn = g.S.View().Learners
	}
	g.updateLearning()
	if g.Verify {
		if err := g.verifyFrame(); err != nil {
			if g.verifyDone > 0 {
				return ebiten.Termination
			}
			return err
		}
		g.modals = nil
		return nil
	}
	if g.S.Screen == "quit" {
		return ebiten.Termination
	}
	if len(g.Shots) > 0 {
		if g.S.Engine == nil {
			g.request(session.Request{Op: "new", Seed: g.seed})
		}
		g.autoplay, g.speed = true, 4
		if g.ticks > g.Shots[len(g.Shots)-1]+1 {
			return ebiten.Termination
		}
		for _, t := range g.Shots {
			g.capture = g.capture || g.ticks == t
		}
	}
	if g.input {
		g.cliInput()
		return nil
	}
	if g.duelPreview != nil {
		g.updateDuelPreview()
	}
	if g.player.Duel == nil && g.duelCamHeld {
		g.camX, g.camY, g.duelCamHeld = g.duelCam[0], g.duelCam[1], false
	}
	if g.player.Duel != nil {
		g.updateDuel()
		return nil
	}
	if g.exploring() {
		g.updateExplore()
		if !g.popupPending() {
			g.exploreInput()
		}
	}
	logInput := g.updateTableDrag() || g.updateWheelLists() || g.updateBattleLog() || g.updateUnitList() || g.updateWaypoints() || g.updateShopList()
	g.updateDeploymentPanel()
	g.updateHover()
	g.keys()
	g.edgeOn = false
	if !logInput {
		g.camera()
	}
	if !logInput && mouseJustPressed(ebiten.MouseButtonLeft) {
		g.leftClick()
	}
	playing := g.S.Screen == "playing"
	if playing && !g.autoplay && !g.popupPending() {
		if o := g.S.View(); o.Phase == "scenario" && o.Dialogue.Kind == "battle" && g.sortieForced(o) {
			g.startSortie()
		}
	}
	if g.autoplay && len(g.modals) > 0 && g.modals[0].t > 1.2 {
		g.closeModal()
	}
	if playing && g.overlay() == "" && len(g.modals) == 0 && g.ticks%12 == 0 {
		if g.autoplay {
			g.assist()
		} else {
			o := g.S.View()
			if o.Phase == "battle" && o.Turn != "ally" && !g.busy() && len(g.learn) == 0 {
				g.resolvePhase()
			}
			g.offerEnd(o)
		}
	}
	return nil
}

// offerEnd asks once per turn to end the allied phase when every ally has been ordered.
func (g *Game) offerEnd(o core.Observation) {
	if o.Phase != "battle" || o.Turn != "ally" || g.busy() || g.ord.actor != "" || g.popupPending() || len(o.Learners) > 0 {
		return
	}
	key := fmt.Sprint(o.Stage, "-", o.Round)
	if left, total := g.pending(o); left > 0 || total == 0 || g.endAsked == key {
		return
	}
	if g.askWith("모든 아군에게 명령을 내렸습니다.\n턴을 넘기겠습니까?", "예", "아니오", g.endTurn) {
		g.endAsked = key
	}
}

// cliInput is the debug command line (backquote): CLI syntax, one command.
func (g *Game) cliInput() {
	g.entry = string(ebiten.AppendInputChars([]rune(g.entry)))
	if keyJustPressed(ebiten.KeyBackspace) {
		if r := []rune(g.entry); len(r) > 0 {
			g.entry = string(r[:len(r)-1])
		}
	}
	if keyJustPressed(ebiten.KeyEnter) {
		c, err := cli.Parse(g.entry)
		if err != nil {
			g.toast(err.Error(), bad)
		} else {
			g.clearOrder()
			g.command(c)
			g.entry, g.input = "", false
		}
	}
	if keyJustPressed(ebiten.KeyEscape) {
		g.input = false
	}
}

func (g *Game) keys() {
	pressed := keyJustPressed
	if pressed(ebiten.KeyF12) {
		g.capture = true
	}
	if pressed(ebiten.KeyF5) && g.S.Engine != nil && !g.popupPending() {
		g.request(session.Request{Op: "save", Slot: fmt.Sprint(g.slot)})
	}
	if pressed(ebiten.KeyF9) && !g.popupPending() {
		g.request(session.Request{Op: "load", Slot: fmt.Sprint(g.slot)})
	}
	if pressed(ebiten.KeyEscape) {
		before := g.sfxCalls
		defer g.sfxUnless(before, "ui-back")
		switch {
		case g.equip.info.id != "":
			g.equip.info = equipChip{}
		case g.overlay() == "shop" && g.shopView.info.id != "":
			g.shopView.info = equipChip{}
		case g.traitOverWindow():
			g.closeModal()
		case g.overlay() == "party" && len(g.party.insu.preview) > 0:
			g.party.insu.preview = nil
		case g.overlay() != "":
			g.closeOverlay()
		case len(g.modals) > 0:
			g.closeModal()
		case len(g.learn) > 0:
		case g.deploying() && g.deploymentSelected != "":
			g.deploymentSelected = ""
		case g.cancel():
		default:
			g.openOverlay("menu")
		}
		return
	}
	if g.overlay() != "" || g.S.Screen != "playing" || g.S.Engine == nil {
		return
	}
	if len(g.modals) > 0 {
		if pressed(ebiten.KeyEnter) || pressed(ebiten.KeySpace) {
			g.closeModal()
		}
		return
	}
	if len(g.learn) > 0 {
		return
	}
	o := g.S.View()
	dialogue := o.Phase == "scenario" && isTalk(o.Dialogue.Kind)
	if pressed(ebiten.KeyGraveAccent) {
		g.input, g.entry = true, ""
	}
	if pressed(ebiten.KeySpace) {
		if dialogue {
			g.advanceDialogue()
		} else {
			g.assist()
		}
	}
	if pressed(ebiten.KeyEnter) {
		if dialogue {
			g.advanceDialogue()
		}
		g.player.Skip()
	}
	if pressed(ebiten.KeyF) {
		g.setSpeed(nextSpeed[g.speed])
		g.toast(fmt.Sprintf("재생 속도 ×%d", g.speed), gold)
	}
	if o.Phase == "deployment" {
		g.deploymentKeys()
	}
}

// leftClick presses a button at once; on the battlefield it waits for the release, since
// holding the button and dragging scrolls the map instead (camera).
func (g *Game) leftClick() {
	g.rushSpeech()
	x, y := cursorPos()
	for i := len(g.buttons) - 1; i >= 0; i-- {
		if b := g.buttons[i]; image.Pt(x, y).In(b.rect) {
			if b.denied {
				g.sfx("ui-denied")
			}
			before := g.sfxCalls
			b.click()
			g.sfxUnless(before, "ui-select")
			return
		}
	}
	if g.overlay() != "" || len(g.modals) > 0 || len(g.learn) > 0 || g.S.Screen != "playing" || g.S.Engine == nil || g.field == nil || g.dragging {
		return
	}
	if g.exploring() {
		if !g.overHUD(x, y) { // not through the corner panels
			g.exploreClick(g.toCanvas(cursorWorld()))
		}
		return
	}
	if o := g.S.View(); o.Phase != "battle" && o.Phase != "deployment" {
		return
	}
	if !g.onMap() || g.edgeOn {
		return
	}
	g.dragging, g.dragMoved, g.dragButton, g.dragX, g.dragY = true, false, ebiten.MouseButtonLeft, x, y
}

// mapRelease is a left click on the battlefield that did not become a drag.
func (g *Game) mapRelease() {
	if g.overlay() != "" || len(g.modals) > 0 || len(g.learn) > 0 || g.S.Engine == nil || g.field == nil || g.busy() {
		return
	}
	if !g.onMap() {
		return
	}
	if g.hover.ok {
		before := g.sfxCalls
		g.mapClick(g.hover.x, g.hover.y)
		g.sfxUnless(before, "ui-select")
	} else if g.cancel() {
		g.sfx("ui-back")
	} else {
		g.selected = ""
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	lw, lh := g.LayoutF(float64(w), float64(h))
	return int(math.Ceil(lw)), int(math.Ceil(lh))
}

// LayoutF renders the whole window at its device resolution.
func (g *Game) LayoutF(w, h float64) (float64, float64) {
	d := ebiten.Monitor().DeviceScaleFactor()
	g.fit(w*d, h*d)
	return w * d, h * d
}

// Draw draws the frame; a fixed-layout screen goes to its own canvas, centred in the window.
func (g *Game) Draw(dst *ebiten.Image) {
	if !g.fixedLayout() {
		g.drawFrame(dst)
		return
	}
	w, h := int(math.Ceil(Width*uiPx)), int(math.Ceil(Height*uiPx))
	if g.fixedCanvas == nil || g.fixedCanvas.Bounds().Dx() != w || g.fixedCanvas.Bounds().Dy() != h {
		if g.fixedCanvas != nil {
			g.fixedCanvas.Deallocate()
		}
		g.fixedCanvas = newLayer(w, h)
	}
	g.fixedCanvas.Clear()
	g.drawFrame(g.fixedCanvas)
	dst.Fill(color.NRGBA{R: 16, G: 13, B: 12, A: 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(fixedAt.X), float64(fixedAt.Y))
	dst.DrawImage(g.fixedCanvas, op)
}

func (g *Game) drawScreen(dst *ebiten.Image) {
	if g.gallery != nil {
		dst.Fill(color.NRGBA{R: 16, G: 13, B: 12, A: 255})
		g.buttons, g.blocks = nil, nil
		g.drawDuelGallery(dst)
		if g.capture {
			g.capture = false
			g.screenshot(dst)
		}
		return
	}
	if g.editor == nil {
		g.publishBattleLog()
	}
	dst.Fill(color.NRGBA{R: 16, G: 13, B: 12, A: 255})
	g.buttons, g.blocks, g.wheelAreas = nil, nil, nil
	if g.player.Duel != nil {
		g.drawDuel(dst)
		g.showWindows(dst) // a window open when the duel began closes over it
		if g.capture {
			g.capture = false
			g.screenshot(dst)
		}
		return
	}
	if g.editor != nil {
		g.drawEditor(dst)
	} else if g.tour != nil {
		g.drawTour(dst)
	} else if g.world != nil {
		g.drawWorld(dst)
	} else if g.S.Screen == "title" || g.S.Engine == nil {
		g.title(dst)
	} else {
		g.play(dst)
	}
	if g.editorReturn != nil && !g.deploying() {
		g.btn(dst, image.Rect(uiW-230, 12, uiW-14, 44), "에디터로 돌아가기", g.endEditorTrial)
	}

	if g.overlay() != "" {
		g.drawOverlay(dst)
		if g.traitOverWindow() {
			g.drawModal(dst)
		}
	} else if len(g.modals) > 0 && (!g.busy() || g.briefed()) && g.S.Engine != nil {
		g.drawModal(dst)
	}
	g.showWindows(dst)
	g.drawEdgeArrow(dst)
	if g.input {
		rect(dst, 0, float32(uiH-40), float32(uiW), 40, color.NRGBA{A: 220})
		g.label(dst, "> "+g.entry+"_", 16, float64(uiH-34), 18, gold)
	}
	if g.capture {
		g.capture = false
		g.screenshot(dst)
	}
}

func (g *Game) screenshot(dst *ebiten.Image) {
	dir := g.auditDir
	if dir == "" {
		dir = "reports/gui"
	}
	_ = os.MkdirAll(dir, 0755)
	img := image.NewRGBA(dst.Bounds())
	dst.ReadPixels(img.Pix)
	file, err := os.Create(filepath.Join(dir, fmt.Sprintf("screen-%06d.png", g.ticks)))
	if err == nil {
		err = png.Encode(file, img)
		file.Close()
	}
	if err != nil {
		g.toast(err.Error(), bad)
	}
}

// title is the start screen: the dragon pattern drifting slowly, darker toward the edges,
// under the menu.
func (g *Game) title(dst *ebiten.Image) {
	drift := g.uiSeconds * 7
	g.drawDragon(dst, 0, 0, float64(uiW), float64(uiH), drift, drift, dragonTeal)
	for i := float32(1); i <= 20; i++ {
		t := i * 12
		shade := color.NRGBA{A: 8}
		rect(dst, 0, 0, float32(uiW), t, shade)
		rect(dst, 0, float32(uiH)-t, float32(uiW), t, shade)
		rect(dst, 0, 0, t, float32(uiH), shade)
		rect(dst, float32(uiW)-t, 0, t, float32(uiH), shade)
	}
	g.inFrame(dst, Width, Height, image.Rect(0, 0, Width, Height), 0, g.titleMenu)
}

// titleCast is the three officers the title shows: the starting roster, then those who join in story order.
func (g *Game) titleCast() []string {
	d := g.S.Data
	cast := slices.Clone(d.Campaign.Roster)
	for _, c := range d.Chapters {
		for _, id := range append(slices.Clone(c.Nodes), c.Events...) {
			if n := d.Node(id); n.Kind == "join" && !slices.Contains(cast, n.Character) {
				cast = append(cast, n.Character)
			}
		}
	}
	return cast[:min(3, len(cast))]
}

// titleMenu is the title screen's content, laid out on the Width×Height design.
func (g *Game) titleMenu(dst *ebiten.Image) {
	g.label(dst, g.S.Data.Title, 96, 70, 64, gold)
	for i, id := range g.titleCast() {
		g.portrait(dst, id, "", float64(650+i*190), float64(205+i%2*50), 170)
	}
	items := []menuItem{{"새로 시작", func() {
		if len(g.S.Data.Rules.Difficulties) == 0 {
			g.request(session.Request{Op: "new", Seed: g.seed})
			return
		}
		g.jumpNode = ""
		g.openOverlay("difficulty")
	}}, {"장면 이동", func() { g.waypoints.top = 0; g.openOverlay("waypoints") }}}
	cont := menuItem{label: "이어하기"}
	if slot, ok := g.S.Store.Latest(); ok {
		cont.click = func() { g.request(session.Request{Op: "load", Slot: slot}) }
	}
	items = append(items[:1], append([]menuItem{cont}, items[1:]...)...)
	items = append(items, menuItem{"불러오기", func() { g.openOverlay("load") }})
	if g.Debug {
		items = append(items, menuItem{"일기토", func() { _ = g.OpenDuel("", "", true) }},
			menuItem{"맵에디터", func() { g.editor = &mapEditor{selected: -1, level: 1, facing: "NW", zoom: 1} }})
	}
	items = append(items, menuItem{"설정", func() { g.openOverlay("settings") }}, menuItem{"종료", func() { g.request(session.Request{Op: "quit"}) }})
	r := image.Rect(96, 290, 420, 290+54+len(items)*56)
	g.window(dst, r)
	for i, it := range items {
		g.btn(dst, image.Rect(r.Min.X+28, r.Min.Y+30+i*56, r.Max.X-28, r.Min.Y+30+i*56+44), it.label, it.click)
	}
}

func (g *Game) play(dst *ebiten.Image) {
	o := g.S.View()
	if g.held != nil && !slices.ContainsFunc(g.modals, func(m *modal) bool { return m.kind == "acquire" }) {
		g.held = nil
	}
	if g.held != nil {
		o = *g.held
	}
	if o.Phase != "scenario" {
		g.dialogueAnim = dialogueAnim{} // the next story opens a fresh box
	}
	switch o.Phase {
	case "complete":
		g.complete(dst, o)
	case "scenario":
		g.scenario(dst, o)
		if o.Dialogue.Kind == "battle" && !g.sortieForced(o) {
			g.drawSortie(dst, o)
			return
		}
		g.drawIntro(dst)
		g.storyPanels(dst, o)
	default:
		g.battle(dst, o)
		if o.Phase == "result" && !g.busy() {
			g.result(dst, o)
		}
	}
}

func (g *Game) complete(dst *ebiten.Image, o core.Observation) {
	g.backdrop(dst, o)
	g.inDesign(dst, image.Rect(Width/2-320, 180, Width/2+320, 620), func(dst *ebiten.Image) { g.completeWindow(dst, o) })
}

// lastWon names the last battle of the campaign the player has won; a story started past every battle
// (a jump to a later chapter) has none, and the window speaks of the story instead.
func (g *Game) lastWon() string {
	stages := g.S.Data.Stages
	for i := len(stages) - 1; i >= 0; i-- {
		if g.S.Engine.Holds([]string{"won:" + stages[i].ID}) {
			return stages[i].Name
		}
	}
	return "이야기"
}

// completeWindow is the campaign's closing window, laid out on the Width×Height design.
func (g *Game) completeWindow(dst *ebiten.Image, o core.Observation) {
	r := image.Rect(uiW/2-320, 180, uiW/2+320, 620)
	if end := g.S.Engine.Ending(); end != "" { // the hero fell: the story ends here, a save takes it back
		g.titled(dst, r, "끝")
		g.centered(dst, end, float64(uiW/2), 220, 34, gold)
		g.centered(dst, "유비의 이야기가 여기서 끝났습니다. 저장한 곳에서 다시 할 수 있습니다.", float64(uiW/2), 276, 19, ink)
	} else {
		g.titled(dst, r, "완료")
		g.centered(dst, g.lastWon()+" 완료", float64(uiW/2), 220, 34, gold)
		g.centered(dst, fmt.Sprintf("%d번의 싸움을 마쳤습니다.", len(g.S.Data.Stages)), float64(uiW/2), 276, 19, ink)
	}
	y := 336
	for _, c := range g.roster(o)[:min(4, len(g.roster(o)))] {
		g.portrait(dst, c.ID, c.Class, float64(r.Min.X+80), float64(y), 48)
		g.label(dst, fmt.Sprintf("%s   Lv%d   %s", g.name(c.ID), c.Level, g.className(c.Class)), float64(r.Min.X+148), float64(y+12), 19, ink)
		y += 62
	}
	g.autoplay = false
	g.btn(dst, image.Rect(r.Max.X-220, r.Max.Y-60, r.Max.X-32, r.Max.Y-24), "시작 메뉴", func() { g.request(session.Request{Op: "title"}) })
}

// Layers of the battle screen; the duel transition draws and fades them apart.
const (
	layerAll = iota
	layerMap
	layerUI
)

// battle draws the field full-screen with the HUD over it.
func (g *Game) battle(dst *ebiten.Image, o core.Observation) {
	if g.layer != layerUI {
		restore := g.battleMap(dst, o)
		defer restore()
	}
	if g.layer == layerMap {
		return
	}
	g.battleHUD(dst, o)
}

// battleMap draws the field with its units, effects and popups. The returned func undoes
// the camera shake applied while drawing.
func (g *Game) battleMap(dst *ebiten.Image, o core.Observation) func() {
	g.ensureField(o)
	units := make([]*unitVis, 0, len(g.vis))
	for _, u := range o.UnitViews {
		if v, ok := g.vis[u.ID]; ok {
			units = append(units, v)
		}
	}
	vp := g.battleViewport()
	if least := g.zoomSteps()[0]; g.zoom < least && g.zoomTo < least {
		g.setZoomImmediate(least) // a battle opening (or a window grown) wider than the map
	}
	if g.scene == nil {
		g.camX, g.camY = g.clampCam(g.camX, g.camY) // also while a popup holds the camera input back
	}
	hw, hh := float64(vp.Dx())/2/g.zoom, float64(vp.Dy())/2/g.zoom
	g.field.prepareRender(g.zoom, g.camX-hw, g.camY-hh, vp.Dx(), vp.Dy())
	g.field.DrawBounds = image.Rect(int(g.camX-hw)-64, int(g.camY-hh)-96, int(g.camX+hw)+64, int(g.camY+hh)+96)
	g.field.setDeployment(o.Phase == "deployment", o.Map.Allies)
	if g.player.looks != nil {
		g.field.SetLooks(g.player.looks)
	} else {
		g.field.SetLooks(LooksOf(g.S.Data, o))
	}
	g.field.setNight(o.Map.Night, lightCells(o.Map.Lights))
	g.field.setGoals(g.goalShapes(o))
	g.field.RangeBorder = g.rangeBorder(o)
	g.field.Draw(g.clockMS, units, g.marks(o))
	g.drawDeploymentPreview(o)
	if g.field.Map.Native && g.shakeT > 0 {
		inWorld(func() {
			g.field.background(dst, vp, g.zoom, -g.camX*g.zoom+float64(vp.Min.X+vp.Dx()/2), -g.camY*g.zoom+float64(vp.Min.Y+vp.Dy()/2), false)
		})
	}
	sx, sy := g.shake()
	g.camX, g.camY = g.camX-sx/g.zoom, g.camY-sy/g.zoom
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-g.camX, -g.camY)
	op.GeoM.Scale(g.zoom, g.zoom)
	op.GeoM.Translate(float64(vp.Min.X+vp.Dx()/2), float64(vp.Min.Y+vp.Dy()/2))
	if g.field.Map.Native {
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(sx, sy)
	}
	op.Filter = ebiten.FilterPixelated
	inWorld(func() {
		drawScaled(dst, g.field.Canvas, op)
		g.cursor(dst)
		g.drawEffects(dst)
	})
	// popups stand over their unit but are UI: the UI size enlarges them on top of the zoom
	for _, p := range g.popups {
		x, y := g.field.Center(p.unit.u, p.unit.v)
		if p.kind != 0 {
			sx, sy := worldToUI(g.toScreen(x+p.unit.shakeOffset(), y-p.unit.lift-p.unit.top-10-float64(p.row)*28/g.zoom))
			g.drawNumber(dst, p, sx, sy)
			continue
		}
		sx, sy := worldToUI(g.toScreen(x+p.unit.shakeOffset(), y-p.unit.lift-p.unit.top-10-p.t*16-float64(p.row)*18/g.zoom))
		g.outlined(dst, p.text, sx-g.width(p.text, 16)/2, sy, 16)
	}
	g.drawHitFlash(dst)
	return func() { g.camX, g.camY = g.camX+sx/g.zoom, g.camY+sy/g.zoom }
}

func (g *Game) battleHUD(dst *ebiten.Image, o core.Observation) {
	g.hudRect = image.Rectangle{}
	if g.bannerT > 0 {
		w := g.width(g.banner, 30)
		rect(dst, float32(float64(uiW)/2-w/2-30), 300, float32(w+60), 58, color.NRGBA{R: 40, G: 14, B: 10, A: 200})
		rect(dst, float32(float64(uiW)/2-w/2-30), 300, float32(w+60), 2, gold)
		rect(dst, float32(float64(uiW)/2-w/2-30), 356, float32(w+60), 2, gold)
		g.label(dst, g.banner, float64(uiW/2)-w/2, 308, 30, gold)
	}
	if o.Phase == "deployment" {
		g.drawDeployment(dst, o)
		g.messages(dst)
		return
	}
	g.battleLeftPanel(dst, o, g.chapterName(o.Node), g.endSlot(g.player.shownTurn(o)))
	g.battleHUDPanel(dst, g.player.shownTurn(o))
	g.drawMenu(dst)
	g.forecast(dst)
	g.brief(dst, o)
	g.messages(dst)
	g.drawBattleSpeech(dst)
	g.drawTurnBanner(dst)
}

// mapArt is the map of that id: a scene map from the graphics, or a battle map turned into map art from its
// document the first time it is asked for, so starting the game does not build every battle's map.
func (g *Game) mapArt(id string) *MapArt {
	if art := g.assets.Maps[id]; art != nil {
		return art
	}
	doc, ok := g.S.Data.Maps[id]
	if !ok {
		return nil
	}
	art := artFromDocument(doc, g.S.Data)
	g.assets.Maps[id] = art
	return art
}
