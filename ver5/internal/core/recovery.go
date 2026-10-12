package core

// RecoveryPreview is the recovery at the next start of the unit's faction,
// assuming units stay at their current positions and no new actions occur.
type RecoveryPreview struct {
	HP, MP    int
	TerrainHP bool
}

// NextTurnRecovery runs turn transitions in a private world, leaving the engine unchanged.
func (e *Engine) NextTurnRecovery(id string) RecoveryPreview {
	u := e.unit(id)
	if e.state.Phase != "battle" || u == nil || u.HP <= 0 {
		return RecoveryPreview{}
	}
	out := RecoveryPreview{TerrainHP: e.tileAt(u.X, u.Y).RestHP > 0}
	t := fromState(e.data, e.Snapshot())
	// At most three phases separate two starts of the same controller.
	for i := 0; i < 3; i++ {
		t.events = nil
		t.endFaction()
		if u.ActsIn(t.state.Turn) {
			break
		}
	}
	for _, event := range t.events {
		if event.Actor != id {
			continue
		}
		switch event.Kind {
		case "recover-hp":
			out.HP += event.Amount
		case "recover-mp":
			out.MP += event.Amount
		}
	}
	return out
}
