package main

import (
	"demo1/internal/terrain"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
	"path/filepath"
)

func rect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	vector.FillRect(dst, x, y, w, h, c, false)
}
func label(dst *ebiten.Image, s string, x, y int) { ebitenutil.DebugPrintAt(dst, s, x, y) }
func (g *game) diamond(dst *ebiten.Image, u, v, r float64, c color.Color) {
	pts := [][2]float64{{u - r, v - r}, {u + r, v - r}, {u + r, v + r}, {u - r, v + r}}
	for i, p := range pts {
		q := pts[(i+1)%4]
		x, y := g.pos(p[0], p[1])
		xx, yy := g.pos(q[0], q[1])
		vector.StrokeLine(dst, x, y, xx, yy, 1, c, false)
	}
}
func (g *game) drawEditorUI(screen *ebiten.Image) {
	if g.grid {
		for axis := 0; axis < 2; axis++ {
			n := g.w.Width
			if axis == 1 {
				n = g.w.Height
			}
			for i := 0; i <= n*2; i++ {
				t := float64(i) / 2
				a, b := g.pos(t, 0)
				c, d := g.pos(t, float64(g.w.Height))
				if axis == 1 {
					a, b = g.pos(0, t)
					c, d = g.pos(float64(g.w.Width), t)
				}
				width := float32(.5)
				alpha := uint8(50)
				if i%2 == 0 {
					width = 1
					alpha = 130
				}
				vector.StrokeLine(screen, a, b, c, d, width, color.NRGBA{232, 229, 197, alpha}, false)
			}
		}
	}
	mx, my := ebiten.CursorPosition()
	vx, vy := g.mouseCell()
	selected := mx >= panelW && g.w.Inside(vx, vy)
	if selected {
		for dy := -g.radius; dy <= g.radius; dy++ {
			for dx := -g.radius; dx <= g.radius; dx++ {
				if dx*dx+dy*dy <= g.radius*g.radius && g.w.Inside(vx+dx, vy+dy) {
					g.diamond(screen, float64(vx+dx)+.5, float64(vy+dy)+.5, .5, accent)
				}
			}
		}
	}
	rect(screen, 0, 0, panelW, screenH, ink)
	rect(screen, panelW-1, 0, 1, screenH, color.NRGBA{59, 80, 87, 255})
	label(screen, "RIVER / SOIL / GRASS", 20, 21)
	label(screen, "ISOMETRIC AUTOTILE LAB", 20, 43)
	rect(screen, 20, 76, 240, 1, muted)
	for _, b := range buttons {
		c := color.NRGBA{38, 51, 58, 255}
		kind, isBrush := brushKind(b.action)
		selected := (isBrush && g.brush == kind) || (b.action == "grid" && g.grid)
		if selected {
			c = color.NRGBA{46, 95, 95, 255}
		}
		if mx >= 18 && mx < 262 && my >= b.y && my < b.y+34 {
			c = color.NRGBA{61, 81, 86, 255}
		}
		rect(screen, 18, float32(b.y), 244, 34, c)
		if selected {
			rect(screen, 18, float32(b.y), 3, 34, accent)
		}
		label(screen, b.label, 29, b.y+9)
	}
	label(screen, fmt.Sprintf("BRUSH %d   [ / ] to resize", g.radius+1), 20, 318)
	label(screen, "TREE / GRASS SWAY SPEED", 20, 602)
	for i, speed := range []int{1, 2, 4, 8} {
		x := 18 + i*62
		c := color.NRGBA{38, 51, 58, 255}
		if g.swaySpeed == speed {
			c = color.NRGBA{46, 95, 95, 255}
		}
		rect(screen, float32(x), 626, 58, 34, c)
		if g.swaySpeed == speed {
			rect(screen, float32(x), 657, 58, 3, accent)
		}
		label(screen, fmt.Sprintf("%dx", speed), x+20, 635)
	}
	label(screen, "LMB paint  /  RMB erase\nWheel zoom / Space-drag pan\nP export PNG / Z-Y history", 20, 690)
	modified := ""
	if g.dirty || g.strokeChanged {
		modified = " *"
	}
	rect(screen, panelW, 0, screenW-panelW, 43, ink)
	label(screen, fmt.Sprintf("%dx%d diamonds  |  zoom %dx  |  %s%s", g.w.Width, g.w.Height, g.camera.Zoom, filepath.Base(g.path), modified), panelW+20, 13)
	rect(screen, panelW, screenH-33, screenW-panelW, 33, ink)
	status := g.status
	if len(status) > 115 {
		status = status[:112] + "..."
	}
	label(screen, status, panelW+20, screenH-24)
	if selected {
		label(screen, fmt.Sprintf("Cell %d,%d %s\nSoil %v\nDry  %v", vx, vy, g.w.Cell(vx, vy), g.w.Masks(vx, vy), g.w.MasksFor(vx, vy, terrain.Grass)), 20, 746)
	}
}
