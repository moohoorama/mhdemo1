package gui

import "fmt"

func init() { demos["sound-audit"] = demoSoundAudit }

func demoSoundAudit(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	g.selected = d.firstAlly()
	g.begin(*g.unitView(g.S.View(), g.selected))
	g.ord.stage = "menu"
	d.look(g.selected)
	click := func(match func(button) bool) {
		for i := len(g.buttons) - 1; i >= 0; i-- {
			b := g.buttons[i]
			if !match(b) {
				continue
			}
			cursorPos = func() (int, int) { return (b.rect.Min.X + b.rect.Max.X) / 2, (b.rect.Min.Y + b.rect.Max.Y) / 2 }
			g.leftClick()
			cursorPos = func() (int, int) { return -100, -100 }
			return
		}
		d.err = fmt.Errorf("sound audit: no matching button at frame %d", d.frame)
	}
	d.step = func(f int) {
		switch f {
		case 90:
			click(func(b button) bool { return b.denied })
		case 150:
			g.clearOrder()
			g.askWith("출진하시겠습니까?", "예", "아니오", func() {})
		case 210:
			click(func(b button) bool { return b.label == "예" })
		case 270:
			g.modals = []*modal{{kind: "objective"}}
		case 330:
			g.closeModal()
		}
	}
	return nil
}
