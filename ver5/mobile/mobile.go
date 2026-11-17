// Package mobile is the Android binding for the desktop game's GUI.
package mobile

import (
	"path/filepath"
	"sync"
	"time"
	_ "time/tzdata" // Android has no zoneinfo where Go looks for it

	ebitenmobile "github.com/hajimehoshi/ebiten/v2/mobile"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/session"
)

var startOnce sync.Once
var startErr error

// Start initializes one game per Android process. Assets and saves are private to the app. zone is the phone's
// time zone (Java's TimeZone ID): Go reads no TZ on Android and would show save times in UTC.
func Start(root, zone string) error {
	startOnce.Do(func() {
		setZone(zone)
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
		gui.DefaultUIScale = 130 // phone screens: larger menus and windows by default
		g, err := gui.New(s, filepath.Join(root, "runtime", "assets", "fonts", "NotoSansKR.ttf"), 1, "")
		if err != nil {
			startErr = err
			return
		}
		_ = g.EnableSound() // the game plays on silent without it
		ebitenmobile.SetGame(g)
	})
	return startErr
}

// setZone makes zone (an IANA ID such as "Asia/Seoul") the local time zone; an unknown one leaves it as it is.
func setZone(zone string) {
	if loc, err := time.LoadLocation(zone); err == nil {
		time.Local = loc
	}
}

// Back queues Escape on the game thread instead of mutating GUI state from Java.
func Back() { gui.RequestBack() }
