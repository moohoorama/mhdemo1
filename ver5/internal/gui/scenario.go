package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
)

// line is one speaker's part of a dialogue node. Content keeps several speakers in one
// text ("유비: … 관우·장비: …"); the screen shows them one by one.
type line struct {
	speakers []string
	text     string
	portrait string // portrait key for a speaker who is no character (a town's smith); "" finds the character's
}

var speakerMark = regexp.MustCompile(`(?:^|\s)([가-힣]{1,4}(?:·[가-힣]{1,4})*):\s*`)

// isTalk is a story node the player pages through: a line of speech or narration.
func isTalk(kind string) bool { return kind == "say" || kind == "narrate" }

// nodeLines is what the node shows: a say node is one line by its speakers, a narrate node one line
// without a speaker; other nodes (choices, the sortie screen) keep marking speakers inside their text.
func (g *Game) nodeLines(n content.Node) []line {
	switch n.Kind {
	case "say":
		names := []string{}
		for _, id := range strings.Split(n.Speaker, "·") {
			names = append(names, g.S.Data.Characters[id].Name)
		}
		return []line{{speakers: names, text: n.Text}}
	case "narrate":
		return []line{{text: n.Text}}
	}
	return splitLines(n.Text)
}

func splitLines(s string) []line {
	marks := speakerMark.FindAllStringSubmatchIndex(s, -1)
	if len(marks) == 0 {
		return []line{{text: s}}
	}
	out := []line{}
	if head := strings.TrimSpace(s[:marks[0][0]]); head != "" {
		out = append(out, line{text: head})
	}
	for i, m := range marks {
		end := len(s)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}
		out = append(out, line{speakers: strings.Split(s[m[2]:m[3]], "·"), text: strings.TrimSpace(s[m[1]:end])})
	}
	return out
}

// letterMS is how long each letter of a line takes to come out, at a speaking pace; a click
// hurries the rest rushFactor times as fast.
const (
	letterMS   = 45
	rushFactor = 8
)

// typing is a line coming out letter by letter from start (UI seconds), hurried from rush on
// (+Inf until the player clicks).
type typing struct{ start, rush float64 }

func startTyping(now float64) typing { return typing{start: now, rush: math.Inf(1)} }

// typedAll is a typing that is already out.
var typedAll = typing{start: math.Inf(-1), rush: math.Inf(1)}

// letters is how many letters of an n-letter line are out at now; all of them at once when
// the player turned typing off.
func (g *Game) letters(t typing, n int, now float64) int {
	if g.S.Settings.InstantText {
		return n
	}
	s := now - t.start
	if now > t.rush {
		s = t.rush - t.start + (now-t.rush)*rushFactor
	}
	return max(0, min(n, int(s*1000/letterMS)))
}

// hurry speeds up the rest of a line, or shows it whole if it is already hurried.
func (t *typing) hurry(now float64) {
	if math.IsInf(t.rush, 1) {
		t.rush = now
	} else {
		*t = typedAll
	}
}

// shownText is the typed-out part of the current line.
func (g *Game) shownText(l line) (string, bool) {
	r := []rune(l.text)
	n := g.letters(g.lineTyping, len(r), g.textSeconds)
	return string(r[:n]), n == len(r)
}

// lineKey names a shown line, so the box knows when the line under it changed.
func lineKey(speakers []string, node string, i int) string {
	return fmt.Sprintf("%s|%s|%d", strings.Join(speakers, "·"), node, i)
}

// advanceDialogue hurries the typing (a second click finishes it), then shows the next speaker,
// then the next node.
func (g *Game) advanceDialogue() {
	if g.narrationLeaving || g.uiSeconds < g.skipUntil || g.scene != nil && g.scene.leaving {
		return
	}
	o := g.S.View()
	if o.Phase != "scenario" || !isTalk(o.Dialogue.Kind) {
		return
	}
	if g.sceneBusy() {
		if !slices.Contains(g.scene.def.NoSkip, g.scene.node) {
			g.skipBeats()
		}
		return
	}
	if g.busy() {
		return
	}
	if o.Dialogue.Kind == "narrate" {
		a := &g.dialogueAnim
		if a.key == lineKey(nil, o.Node, 0) && g.uiSeconds-a.shownAt >= riseTime(o.Dialogue) {
			g.narrationLeaving, g.narrationExitAt = true, g.uiSeconds
		}
		return
	}
	lines := g.nodeLines(o.Dialogue)
	if _, done := g.shownText(lines[min(g.lineIndex, len(lines)-1)]); !done {
		g.lineTyping.hurry(g.textSeconds)
		return
	}
	if g.lineIndex+1 < len(lines) {
		g.lineIndex++
		g.lineTyping = startTyping(g.textSeconds)
		g.startLine(g.lineIndex)
		return
	}
	g.nextNode()
}

// nextNode goes on to the next node; when that node opens another scene or goes back to the walk, the scene on
// screen first darkens to black (the new one then opens out of black). Beats set after the node's last line play
// before that.
func (g *Game) nextNode() {
	if s := g.scene; s != nil && s.node == g.S.View().Node {
		if s.leaving {
			return
		}
		if n := len(g.nodeLines(g.S.View().Dialogue)); g.lineIndex < n && len(s.def.Line(s.node, n)) > 0 {
			g.lineIndex = n // the beats after the node's last line play out first (closingBeats)
			g.startLine(n)
			return
		}
		next := g.S.Data.After(s.node)
		if def, _ := g.scenes.For(next); def != nil && def != s.def || g.backToWalk(next) {
			s.leaving = true
			return
		}
	}
	g.command(core.Command{Kind: "next"})
}

// backToWalk reports whether the story goes from node back to the walk with nothing shown on the way: the walk is
// laid out anew (people come and go with the flags the story set), so the screen goes through black.
func (g *Game) backToWalk(node string) bool {
	for node != "" {
		switch g.S.Data.Node(node).Kind {
		case "back", "explore":
			return true
		case "set", "join", "grant", "leave":
			node = g.S.Data.After(node)
		default:
			return false
		}
	}
	return false
}

// turnPage goes on to the next node once a narration has faded out, and turns the page when the
// auto setting says so.
func (g *Game) turnPage() {
	g.skipScene()
	if g.narrationLeaving && g.uiSeconds-g.narrationExitAt >= .43 {
		g.narrationLeaving = false
		g.nextNode()
	}
	g.closingBeats()
	g.autoPage()
}

// closingBeats goes on to the next node once the beats after a node's last line have played out.
func (g *Game) closingBeats() {
	o := g.S.View()
	s := g.scene
	if s == nil || s.leaving || o.Phase != "scenario" || !isTalk(o.Dialogue.Kind) || g.shownNode != o.Node || s.node != o.Node {
		return
	}
	if g.lineIndex >= len(g.nodeLines(o.Dialogue)) && !g.sceneBusy() {
		g.nextNode()
	}
}

// autoWait is how long a line that is out stays before the auto setting turns the page.
const autoWait = 2.0

// autoPage turns the page as a click would once the line on screen has been out autoWait seconds,
// when the player turned it on.
func (g *Game) autoPage() {
	key, next := g.pageReady()
	if !g.S.Settings.AutoText && !g.campaignActive() || next == nil {
		g.autoKey = ""
		return
	}
	if g.autoKey != key {
		g.autoKey, g.autoAt = key, g.textSeconds
		return
	}
	if g.textSeconds-g.autoAt >= autoWait {
		g.autoKey = ""
		next()
	}
}

// pageReady names the line on screen and how to turn past it, when it is out and nothing else holds the page.
func (g *Game) pageReady() (string, func()) {
	if g.S.Engine == nil || g.S.Screen != "playing" || g.overlay() != "" || len(g.modals) > 0 || len(g.learn) > 0 {
		return "", nil
	}
	if g.exploring() {
		w := g.explore
		if w == nil || w.talk == nil {
			return "", nil
		}
		if _, done := g.shownText(w.talk[w.talkI]); !done {
			return "", nil
		}
		return fmt.Sprintf("explore|%s|%d", w.talkTo, w.talkI), g.nextExploreLine
	}
	o := g.S.View()
	if o.Phase != "scenario" || !isTalk(o.Dialogue.Kind) || g.shownNode != o.Node || g.narrationLeaving || g.sceneBusy() || g.busy() {
		return "", nil
	}
	if o.Dialogue.Kind == "narrate" {
		a := &g.dialogueAnim
		if a.key != lineKey(nil, o.Node, 0) || g.uiSeconds-a.shownAt < riseTime(o.Dialogue) {
			return "", nil
		}
		return a.key, g.advanceDialogue
	}
	lines := g.nodeLines(o.Dialogue)
	l := lines[min(g.lineIndex, len(lines)-1)]
	if _, done := g.shownText(l); !done {
		return "", nil
	}
	return lineKey(l.speakers, o.Node, g.lineIndex), g.advanceDialogue
}

// skipScene pages through the scene being skipped at once, its actors settling where its beats would take them,
// and stops on the first line of whatever comes next: another scene, a choice, a battle, the walk. A window or a
// learning holds it until closed.
func (g *Game) skipScene() {
	for g.skipping != nil {
		if g.S.Screen != "playing" || g.overlay() != "" || len(g.modals) > 0 {
			return
		}
		o := g.S.View()
		if len(o.Learners) > 0 {
			return
		}
		s := g.scene
		if def, _ := g.scenes.For(o.Node); o.Phase != "scenario" || !isTalk(o.Dialogue.Kind) || def != g.skipping || s == nil || s.def != def {
			g.skipping, g.skipUntil = nil, g.uiSeconds+skipGuard
			return
		}
		if s.node == o.Node {
			for g.sceneBusy() {
				g.skipBeats()
			}
		} else {
			g.settleNode(o.Node)
			s.node, s.groups = o.Node, nil
		}
		g.intro, g.narrationLeaving = nil, false
		g.command(core.Command{Kind: "next"})
		if g.S.View().Node == o.Node {
			g.skipping = nil
		}
	}
}

func (g *Game) backdrop(dst *ebiten.Image, o core.Observation) {
	for i := 0; i < uiH; i += 10 {
		k := float64(i) / float64(uiH)
		rect(dst, 0, float32(i), float32(uiW), 10, color.NRGBA{R: uint8(36 - 20*k), G: uint8(28 - 14*k), B: uint8(26 - 10*k), A: 255})
	}
	if i := g.S.Data.ChapterIndex(o.Node); i >= 0 {
		g.label(dst, fmt.Sprintf("제 %d 장", i+1), 60, 40, 22, gold)
	}
	g.label(dst, o.Place, 60, 74, 16, muted)
}

func (g *Game) scenario(dst *ebiten.Image, o core.Observation) {
	if o.Dialogue.Kind == "explore" {
		g.drawExplore(dst, o)
		return
	}
	g.backdrop(dst, o)
	if g.shownNode != o.Node {
		g.shownNode, g.lineIndex, g.lineTyping = o.Node, 0, startTyping(g.textSeconds)
		if g.scene != nil {
			g.scene.node = ""
		}
	}
	if o.Dialogue.Kind == "battle" { // the sortie preparation (drawSortie) or, forced, startSortie
		return
	}
	lines := g.nodeLines(o.Dialogue)
	l := lines[min(g.lineIndex, len(lines)-1)]
	staged := g.enterScene(o)
	if isTalk(o.Dialogue.Kind) {
		g.addButton(image.Rect(0, 0, uiW, uiH), "", g.advanceDialogue)
	}
	if staged {
		g.drawScene(dst, l.speakers)
		g.blackOut(dst, g.scene.fade)
		if g.sceneBusy() { // the line waits for its beats
			g.lineTyping = startTyping(g.textSeconds)
			g.label(dst, "클릭: 넘기기", float64(uiW-150), float64(uiH-40), 13, muted)
			return
		}
	}
	if staged && g.lineIndex >= len(lines) { // the node's closing beats: the box stays closed
		return
	}
	if g.held != nil { // a gift's or a joining officer's window is up: the box closes and the next line opens a fresh one
		g.dialogueAnim = dialogueAnim{}
		return
	}
	key := lineKey(l.speakers, o.Node, g.lineIndex)
	if o.Dialogue.Card {
		g.drawCard(dst, l, key)
	} else if o.Dialogue.Kind == "narrate" {
		if !staged || !g.scene.leaving { // it already faded out before the scene darkens
			g.drawNarration(dst, l, key)
		}
	} else {
		text, done := g.shownText(l)
		g.drawDialogue(dst, l, text, done, key)
	}
	if o.Dialogue.Kind == "choice" {
		y := uiH/2 - 45
		box := image.Rect(uiW/2-180, y-30, uiW/2+180, y+20+len(o.Dialogue.Choices)*52)
		g.titled(dst, box, "선택")
		for _, option := range sorted(o.Dialogue.Choices) {
			key := option
			g.btn(dst, image.Rect(box.Min.X+24, y, box.Max.X-24, y+40), key, func() { g.command(core.Command{Kind: "choose", Option: key}) })
			y += 52
		}
		return
	}
}

// roster is the player's characters who can be sent out.
func (g *Game) roster(o core.Observation) []core.Character {
	out := []core.Character{}
	for _, c := range o.Characters {
		if g.S.Data.Characters[c.Def].Playable && !c.Temp {
			out = append(out, c)
		}
	}
	return out
}

func containsID(cs []core.Character, id string) bool {
	for _, c := range cs {
		if c.ID == id {
			return true
		}
	}
	return false
}
