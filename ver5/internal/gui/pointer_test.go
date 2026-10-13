package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTouchPointer(t *testing.T) {
	t.Run("tap and release retain position", func(t *testing.T) {
		var p touchPointer
		p.update([]touchPoint{{id: 1, x: 100, y: 200}})
		assert.True(t, p.justPressed)
		assert.True(t, p.held)
		p.update([]touchPoint{{id: 1, x: 120, y: 210}})
		assert.False(t, p.justPressed)
		assert.True(t, p.held)
		p.update(nil)
		assert.False(t, p.held)
		assert.Equal(t, 120, p.x)
		assert.Equal(t, 210, p.y)
	})
	t.Run("pinch suppresses clicks until all fingers release", func(t *testing.T) {
		var p touchPointer
		p.update([]touchPoint{{id: 1, x: 100, y: 200}})
		p.update([]touchPoint{{id: 1, x: 100, y: 200}, {id: 2, x: 200, y: 200}})
		assert.False(t, p.held)
		assert.False(t, p.justPressed)
		assert.Equal(t, 1.0, p.ratio)
		p.update([]touchPoint{{id: 1, x: 100, y: 210}, {id: 2, x: 300, y: 210}})
		assert.Equal(t, 2.0, p.ratio)
		assert.Equal(t, 50.0, p.dx)
		assert.Equal(t, 10.0, p.dy)
		p.update([]touchPoint{{id: 2, x: 300, y: 210}})
		assert.False(t, p.held)
		assert.False(t, p.justPressed)
		p.update(nil)
		p.update([]touchPoint{{id: 3, x: 50, y: 50}})
		assert.True(t, p.justPressed)
	})
	t.Run("coincident fingers do not divide by zero", func(t *testing.T) {
		var p touchPointer
		p.update([]touchPoint{{id: 1, x: 10, y: 10}, {id: 2, x: 10, y: 10}})
		p.update([]touchPoint{{id: 1, x: 20, y: 20}, {id: 2, x: 30, y: 30}})
		assert.Equal(t, 1.0, p.ratio)
	})
}
