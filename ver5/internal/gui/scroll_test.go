package gui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// press puts the pointer at (x, y) as a press that has just begun, or held when the press has begun earlier.
func press(x, y int, just bool) {
	cursorPos = func() (int, int) { return x, y }
	pointer.held, pointer.justPressed = true, just
}

// release lets the press go with the pointer at (x, y).
func release(x, y int) {
	cursorPos = func() (int, int) { return x, y }
	pointer.held, pointer.justPressed = false, false
}

// setPointer restores the pointer state the tests change.
func setPointer(t *testing.T) {
	oldCursor, oldPointer := cursorPos, pointer
	t.Cleanup(func() { cursorPos, pointer = oldCursor, oldPointer })
	pointer = touchPointer{}
}

// areaWithChild is a scroll area of 200×200 from (0,0) with one child button at (10,10)-(110,60) that counts its
// clicks: the area's ground, then the child, as drawScreen registers them.
func areaWithChild(g *Game, l *tableList, clicks *int) (child int, area int) {
	area = g.beginScroll(image.Rect(0, 0, 200, 200), l, 50)
	child = len(g.buttons)
	g.addButton(image.Rect(10, 10, 110, 60), "", func() { *clicks++ })
	g.endScroll(area)
	return child, area
}

func TestScrollChildButtonPress(t *testing.T) {
	setPointer(t)
	t.Run("a tap on a child runs it on release, not at the press", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 4}
		clicks := 0
		child, _ := areaWithChild(g, l, &clicks)
		press(50, 30, true)
		g.pressButton(child)
		assert.Equal(t, 0, clicks, "held until the press is let go")
		release(50, 31)
		assert.True(t, g.updateTableDrag())
		assert.Equal(t, 1, clicks)
		assert.Equal(t, 4, l.top)
		assert.Nil(t, g.tableDrag)
	})
	t.Run("a drag from a child scrolls the list and runs nothing", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 0}
		clicks := 0
		child, _ := areaWithChild(g, l, &clicks)
		press(50, 30, true)
		g.pressButton(child)
		press(50, 30-60, false)
		g.updateTableDrag()
		assert.Equal(t, 1, l.top, "up 60px is one 50px row further down")
		release(50, 30-60)
		g.updateTableDrag()
		assert.Equal(t, 0, clicks, "the child is not clicked after a drag")
	})
	t.Run("a press within the threshold is still a tap", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 2}
		clicks := 0
		child, _ := areaWithChild(g, l, &clicks)
		press(50, 30, true)
		g.pressButton(child)
		press(50, 30+dragStartPx, false)
		g.updateTableDrag()
		assert.Equal(t, 2, l.top)
		release(50, 30+dragStartPx)
		g.updateTableDrag()
		assert.Equal(t, 1, clicks)
	})
	t.Run("the scrollbar's own button runs at once", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		clicks := 0
		area := g.beginScroll(image.Rect(0, 0, 200, 200), &tableList{}, 50)
		i := len(g.buttons)
		g.addGrab(image.Rect(190, 0, 200, 200), func() { clicks++ })
		g.endScroll(area)
		press(195, 10, true)
		g.pressButton(i)
		assert.Equal(t, 1, clicks)
		assert.Nil(t, g.tableDrag)
	})
	t.Run("a button outside every area runs at once", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		clicks := 0
		g.addButton(image.Rect(0, 0, 10, 10), "", func() { clicks++ })
		g.pressButton(0)
		assert.Equal(t, 1, clicks)
		assert.Nil(t, g.tableDrag)
	})
	t.Run("the innermost area owns its buttons", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		outer, inner := &tableList{}, &tableList{}
		o := g.beginScroll(image.Rect(0, 0, 300, 300), outer, 40)
		b1 := len(g.buttons)
		g.addButton(image.Rect(0, 0, 10, 10), "", func() {})
		i := g.beginScroll(image.Rect(0, 0, 100, 100), inner, 20)
		b2 := len(g.buttons)
		g.addButton(image.Rect(0, 0, 10, 10), "", func() {})
		g.endScroll(i)
		g.endScroll(o)
		assert.Same(t, outer, g.scrollAreaOf(b1).l)
		assert.Same(t, inner, g.scrollAreaOf(b2).l)
		assert.Nil(t, g.scrollAreaOf(len(g.buttons)))
	})
}

func TestScrollGlide(t *testing.T) {
	setPointer(t)
	t.Run("a fast release glides on and stops at the last row", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 0, end: 6}
		press(100, 300, true)
		g.pressTable(l, 40, nil)
		for _, y := range []int{280, 250, 215, 185} {
			press(100, y, false)
			g.updateTableDrag()
		}
		require.Greater(t, l.top, 0)
		release(100, 185)
		g.updateTableDrag()
		require.Same(t, l, g.flinging, "the release left the list gliding")
		require.Greater(t, l.fling, 0.0, "dragged up: the rows go on toward the end")
		frames := 0
		for g.flinging != nil && frames < 1000 {
			assert.False(t, g.updateFlings(), "a glide never takes the input")
			frames++
		}
		assert.Nil(t, g.flinging)
		assert.Equal(t, l.end, l.top, "stops at the last row")
		assert.Less(t, frames, 200)
	})
	t.Run("a slow release does not glide", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 5, end: 20}
		press(100, 300, true)
		g.pressTable(l, 40, nil)
		press(100, 298, false)
		g.updateTableDrag()
		release(100, 298)
		g.updateTableDrag()
		assert.Nil(t, g.flinging)
	})
	t.Run("a press stops the glide and is a press", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 3, end: 20, fling: 1}
		g.flinging = l
		press(100, 300, true)
		g.updateFlings()
		assert.Nil(t, g.flinging)
		assert.Equal(t, 0.0, l.fling)
	})
	t.Run("a glide stops at the first row too", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 1, end: 20, fling: -2}
		g.flinging = l
		for range 50 {
			g.updateFlings()
		}
		assert.Nil(t, g.flinging)
		assert.Equal(t, 0, l.top)
	})
	t.Run("the battle log glides by its lines and stops at the newest", func(t *testing.T) {
		g := &Game{logScroll: 6, logFling: -3}
		for range 200 {
			g.glideLog(10)
			if g.logFling == 0 {
				break
			}
		}
		assert.Equal(t, 0, g.logScroll)
		assert.Equal(t, 0.0, g.logFling)
		g = &Game{logScroll: 1, logFling: 4}
		g.glideLog(10)
		g.glideLog(10)
		g.glideLog(10)
		assert.LessOrEqual(t, g.logScroll, 10)
	})
}

func TestScrollListRows(t *testing.T) {
	setPointer(t)
	dst := ebiten.NewImage(300, 300)
	defer dst.Deallocate()
	t.Run("the rows stay within the list and the press on a row picks it", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 99}
		picked := -1
		g.listRows(dst, l, image.Rect(0, 0, 200, 120), 10, 40, func(i int, rr image.Rectangle) {}, func(i int) { picked = i })
		assert.Equal(t, 7, l.top, "clamped to the last page of 3 rows")
		assert.Equal(t, 7, l.end)
		row := -1
		for i, b := range g.buttons {
			if b.rect.Min.Y == 40 && b.rect.Dy() == 40 {
				row = i
			}
		}
		require.GreaterOrEqual(t, row, 0)
		press(50, 60, true)
		g.pressButton(row)
		release(50, 60)
		g.updateTableDrag()
		assert.Equal(t, 8, picked, "the second visible row is the eighth")
	})
	t.Run("a press on empty rows drags them", func(t *testing.T) {
		pointer = touchPointer{}
		g := &Game{}
		l := &tableList{top: 0}
		g.listRows(dst, l, image.Rect(0, 0, 200, 120), 10, 40, func(i int, rr image.Rectangle) {}, nil)
		press(50, 100, true)
		assert.True(t, g.pressArea(50, 100))
		press(50, 40, false)
		g.updateTableDrag()
		assert.Equal(t, 1, l.top)
		release(50, 40)
		g.updateTableDrag()
	})
}

func TestSortieColumns(t *testing.T) {
	assert.Equal(t, sortieCols, sortieColumns(10*sortieCellW), "wide enough for the most cards")
	assert.Equal(t, 1, sortieColumns(0))
	assert.Equal(t, 1, sortieColumns(sortieCellW-1))
	assert.Equal(t, 2, sortieColumns(2*sortieCellW))
}

// sortieCardColumns draws the sortie preparation on a w×h screen at the UI size and counts the cards on the first row
// (their lead buttons' distinct x), and the card buttons' rectangles.
func sortieCardColumns(t *testing.T, sc image.Point, scale int) (int, []image.Rectangle) {
	d, err := NewDemo("../..", DemoOptions{Name: "sortie-deputy", Node: "c20-battle"})
	require.NoError(t, err)
	g := d.g
	d.step(1)
	g.S.Settings.UIScale = scale
	g.fit(float64(sc.X), float64(sc.Y))
	cursorPos = func() (int, int) { return -1000, -1000 }
	g.advance(1.0 / 60)
	dst := ebiten.NewImage(sc.X, sc.Y)
	defer dst.Deallocate()
	g.Draw(dst)
	var leads []image.Rectangle
	for _, b := range g.buttons {
		if b.rect.Dx() == sortieLeadW && b.rect.Dy() == sortieLeadH {
			leads = append(leads, b.rect)
		}
	}
	if len(leads) == 0 {
		return 0, nil
	}
	top := leads[0].Min.Y
	xs := map[int]bool{}
	for _, r := range leads {
		if r.Min.Y == top {
			xs[r.Min.X] = true
		}
	}
	return len(xs), leads
}

func TestSortieCardColumnsOnScreen(t *testing.T) {
	for _, c := range []struct {
		name  string
		sc    image.Point
		scale int
	}{{"phone 2400x1080 at 130%", image.Pt(2400, 1080), 130}, {"desktop 1280x850 at 100%", image.Pt(1280, 850), 100}} {
		cols, leads := sortieCardColumns(t, c.sc, c.scale)
		t.Logf("%s: %d columns, %d cards", c.name, cols, len(leads))
		assert.Equal(t, sortieCols, cols, c.name)
	}
}

// TestSortieCardTapAndDrag presses a real sortie card on the preparation screen. In a short window the grid shows one
// row of its two, so it scrolls: a tap on the card selects it, a drag from the card scrolls the grid and leaves the
// selection alone.
func TestSortieCardTapAndDrag(t *testing.T) {
	setPointer(t)
	d, err := NewDemo("../..", DemoOptions{Name: "scrollbar", Node: "c25-battle"})
	require.NoError(t, err)
	g := d.g
	d.step(1)
	draw := func(sc image.Point) {
		g.fit(float64(sc.X), float64(sc.Y))
		g.advance(1.0 / 60)
		dst := ebiten.NewImage(sc.X, sc.Y)
		defer dst.Deallocate()
		g.Draw(dst)
	}
	cursorPos = func() (int, int) { return -1000, -1000 }
	draw(image.Pt(1280, 420))
	var leads []image.Rectangle
	for _, b := range g.buttons {
		if b.rect.Dx() == sortieLeadW && b.rect.Dy() == sortieLeadH {
			leads = append(leads, b.rect)
		}
	}
	require.GreaterOrEqual(t, len(leads), 2)
	ids := g.placeable(g.S.View())
	require.GreaterOrEqual(t, len(ids), 2)
	first, second := leads[0], leads[1]
	require.Equal(t, first.Min.Y, second.Min.Y, "two cards in the first row")
	g.selected = ids[1]
	l := &g.party.sortieSlots

	t.Run("a tap on a card selects it", func(t *testing.T) {
		pointer = touchPointer{}
		at := first.Min.Add(first.Size().Div(2))
		press(at.X, at.Y, true)
		g.leftClick()
		assert.Equal(t, ids[1], g.selected, "nothing changes while the press is held")
		release(at.X, at.Y)
		g.updateTableDrag()
		assert.Equal(t, ids[0], g.selected)
	})
	t.Run("a drag from a card scrolls the grid and keeps the selection", func(t *testing.T) {
		pointer = touchPointer{}
		g.selected = ids[1]
		draw(image.Pt(1280, 420))
		at := second.Min.Add(second.Size().Div(2))
		top := l.top
		press(at.X, at.Y, true)
		g.leftClick()
		press(at.X, at.Y-160, false)
		g.updateTableDrag()
		release(at.X, at.Y-160)
		g.updateTableDrag()
		assert.Greater(t, l.top, top, "dragged up: the grid moves to the next row")
		assert.Equal(t, ids[1], g.selected, "a drag selects nothing")
	})
}
