package gui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/core"
)

// The victory window ranks the officers by the experience the battle brought (core.Merit, kept in the state so a
// game loaded in the middle of the battle still has it). The best three step onto a podium one after another —
// third at the lower right, second at the lower left, first at the top centre — each dropping in with a flash of its
// medal colour; the rest follow as short lines, then what the battle won (its reward, the bonus goals' rewards and
// money). A click on the window plays the rest at once.
const (
	meritDrop  = .42 // seconds a podium card takes to drop in
	meritBurst = .55 // seconds its flash ring takes to fade
)

var (
	medalGold   = color.NRGBA{R: 240, G: 198, B: 80, A: 255}
	medalSilver = color.NRGBA{R: 206, G: 212, B: 222, A: 255}
	medalBronze = color.NRGBA{R: 205, G: 132, B: 70, A: 255}
)

// meritRow is one officer of the victory window.
type meritRow struct {
	unit     core.UnitView
	xp, from int
}

// meritRows are the player's officers of the battle, the most experience first.
func meritRows(o core.Observation) []meritRow {
	rows := []meritRow{}
	for _, u := range resultUnits(o) {
		id := u.Character
		if id == "" {
			id = u.ID
		}
		r := meritRow{unit: u, from: u.Level}
		if m := o.Merit; m != nil {
			r.xp = m.XP[id]
			if from, ok := m.From[id]; ok {
				r.from = from
			}
		}
		rows = append(rows, r)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].xp > rows[j].xp })
	return rows
}

// meritTimes are when the podium's places (third, second, first), the other officers and the spoils come in.
func meritTimes() (places [3]float64, rest, spoils float64) {
	return [3]float64{1.45, .85, .3}, 2.05, 2.35
}

// meritClock is the seconds since the victory window opened; the whole show once a click has skipped it.
func (g *Game) meritClock() float64 {
	if g.resultSkip {
		return 99
	}
	if a := g.windows["result:victory"]; a != nil {
		return g.uiSeconds - a.openedAt
	}
	return 0
}

// victoryWindow draws the victory side of the result window in r.
func (g *Game) victoryWindow(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	st := g.S.Data.Stages[o.Stage]
	g.titled(dst, r, "승리")
	g.centered(dst, st.Name+" 승리", float64(uiW/2), 160, 30, gold)
	t := g.meritClock()
	places, restAt, spoilsAt := meritTimes()
	rows := meritRows(o)
	g.meritSounds(len(rows), t)
	cards := []struct {
		box   image.Rectangle
		medal color.NRGBA
		size  float64
	}{
		{image.Rect(uiW/2-95, 226, uiW/2+95, 396), medalGold, 72},
		{image.Rect(r.Min.X+36, 318, r.Min.X+216, 478), medalSilver, 60},
		{image.Rect(r.Max.X-216, 318, r.Max.X-36, 478), medalBronze, 60},
	}
	for i := 2; i >= 0; i-- {
		if i < len(rows) {
			g.podiumCard(dst, rows[i], i, cards[i].box, cards[i].medal, cards[i].size, t-places[i])
		}
	}
	y := 494
	if rest := rows[min(3, len(rows)):]; len(rest) > 0 && t >= restAt {
		a := phase(t, restAt, .3)
		const perCol, rowH = 4, 22
		for i, row := range rest {
			if i == 2*perCol-1 && len(rest) > 2*perCol {
				g.label(dst, fmt.Sprintf("외 %d명", len(rest)-i), float64(r.Min.X+40+(i/perCol)*296), float64(y+(i%perCol)*rowH), 13, fade(muted, a))
				break
			}
			x, ry := float64(r.Min.X+40+(i/perCol)*296), float64(y+(i%perCol)*rowH)
			g.label(dst, fmt.Sprintf("%d. %s", i+4, row.unit.Name), x, ry, 13, fade(ink, a))
			g.label(dst, meritLevel(row), x+120, ry, 13, fade(gold, a))
			g.label(dst, "+"+xpText(row.xp), x+210, ry, 13, fade(muted, a))
		}
	}
	g.spoils(dst, o, image.Rect(r.Min.X+40, r.Max.Y-158, r.Max.X-40, r.Max.Y-76), phase(t, spoilsAt, .3))
	g.btn(dst, image.Rect(r.Min.X+40, r.Max.Y-64, r.Max.X-40, r.Max.Y-26), "다음 이야기", func() {
		g.command(core.Command{Kind: "continue"})
	})
	if t < spoilsAt+.3 {
		g.addButton(r, "", func() { g.resultSkip = true })
	}
}

// meritLevel is an officer's level, with the level the battle started at when it rose.
func meritLevel(row meritRow) string {
	if row.from < row.unit.Level {
		return fmt.Sprintf("Lv %d→%d", row.from, row.unit.Level)
	}
	return fmt.Sprintf("Lv %d", row.unit.Level)
}

// podiumCard draws place i's card t seconds after it starts to come in: it drops from above onto its spot with a
// little overshoot, flashes white and sends out a ring of its medal colour.
func (g *Game) podiumCard(dst *ebiten.Image, row meritRow, i int, box image.Rectangle, medal color.NRGBA, size, t float64) {
	if t < 0 {
		return
	}
	k := phase(t, 0, meritDrop)
	dy := -(1 - easeOutBack(k)) * 48
	b := box.Add(image.Pt(0, int(dy)))
	x, y, w, h := float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy())
	rect(dst, x, y, w, h, color.NRGBA{R: 30, G: 22, B: 16, A: 240})
	strokeRect(dst, x+1, y+1, w-2, h-2, 2, medal)
	cx := float64(b.Min.X + b.Dx()/2)
	g.portrait(dst, row.unit.ID, row.unit.Class, cx-size/2, float64(b.Min.Y)+22, size)
	ty := float64(b.Min.Y) + 30 + size
	g.centered(dst, row.unit.Name, cx, ty, 17, ink)
	g.centered(dst, meritLevel(row), cx, ty+26, 14, gold)
	g.centered(dst, "경험치 +"+xpText(row.xp), cx, ty+46, 13, muted)
	// the medal on the card's top edge
	fillCircle(dst, float32(cx), y, 17, color.NRGBA{R: 20, G: 14, B: 10, A: 255})
	fillCircle(dst, float32(cx), y, 15, medal)
	g.centered(dst, fmt.Sprint(i+1), cx, float64(y)-11, 17, color.NRGBA{R: 40, G: 26, B: 12, A: 255})
	if land := t - meritDrop*.7; land >= 0 && land < meritBurst {
		p := land / meritBurst
		rect(dst, x, y, w, h, fade(white, .55*(1-p)))
		strokeRect(dst, x-float32(p*26), y-float32(p*26), w+float32(p*52), h+float32(p*52), 3, fade(medal, 1-p))
	}
}

// spoils lays out what the battle won in r at alpha a: an icon with the name and count of each piece, money first.
func (g *Game) spoils(dst *ebiten.Image, o core.Observation, r image.Rectangle, a float64) {
	won := g.S.Data.Stages[o.Stage].Reward // a game saved before the merit was kept
	if o.Merit != nil {
		won = o.Merit.Spoils
	}
	g.label(dst, "획득", float64(r.Min.X), float64(r.Min.Y), 14, gold)
	if a <= 0 {
		return
	}
	ids := make([]string, 0, len(won))
	for id := range won {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(p, q string) int {
		if (p == content.Money) != (q == content.Money) {
			if p == content.Money {
				return -1
			}
			return 1
		}
		if p < q {
			return -1
		}
		return 1
	})
	if len(ids) == 0 {
		g.label(dst, "없음", float64(r.Min.X+48), float64(r.Min.Y), 14, fade(muted, a))
		return
	}
	x, y := r.Min.X+48, r.Min.Y-6
	for _, id := range ids {
		name := fmt.Sprintf("%s ×%d", g.itemName(id), won[id])
		if id == content.Money {
			name = fmt.Sprintf("돈 %d", won[id])
		}
		w := 40 + int(g.width(name, 14)) + 18
		if x+w > r.Max.X && x > r.Min.X+48 {
			x, y = r.Min.X+48, y+40
		}
		g.pieceIcon(dst, id, float64(x), float64(y), 32, a)
		g.label(dst, name, float64(x+40), float64(y+6), 14, fade(ink, a))
		x += w
	}
}
