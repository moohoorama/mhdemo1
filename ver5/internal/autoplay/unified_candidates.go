package autoplay

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

// BattleCommands lists the legal commands a battle policy chooses among at this point, in the order it scores them.
func BattleCommands(d *content.Data, m *BattleModel, e *core.Engine) []core.Command {
	return m.inputs(d, e).Commands
}
