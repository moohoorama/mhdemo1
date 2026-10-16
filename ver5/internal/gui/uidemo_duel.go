package gui

func init() { demos["duel"] = demoDuel }

// demoDuel stages the 사수관 duel (관우 against 화웅) and plays into it: the transition,
// then the cut-in over the duel screen, then the duel itself. The scene needs the game
// running, so it starts on the first frame.
func demoDuel(d *Demo) error {
	d.step = func(f int) {
		if f == 0 {
			if err := d.g.PreviewDuelTransition(d.duel); err != nil {
				d.err = err
			}
		}
	}
	return nil
}
