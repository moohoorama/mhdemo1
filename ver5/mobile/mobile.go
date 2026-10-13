// Package mobile is the Android binding for the desktop game's GUI.
package mobile

import (
	"path/filepath"
	"sync"

	ebitenmobile "github.com/hajimehoshi/ebiten/v2/mobile"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/session"
)

var startOnce sync.Once
var startErr error

// Start initializes one game per Android process. Assets and saves are private to the app.
func Start(root string) error {
	startOnce.Do(func() {
		data, err := content.Load(filepath.Join(root, "runtime", "assets", "data"))
		if err != nil {
			startErr = err
			return
		}
		s, err := session.New(data, filepath.Join(root, "saves"))
		if err != nil {
			startErr = err
			return
		}
		g, err := gui.New(s, filepath.Join(root, "runtime", "assets", "fonts", "NotoSansKR.ttf"), 1, "")
		if err != nil {
			startErr = err
			return
		}
		ebitenmobile.SetGame(g)
	})
	return startErr
}

// Back queues Escape on the game thread instead of mutating GUI state from Java.
func Back() { gui.RequestBack() }
