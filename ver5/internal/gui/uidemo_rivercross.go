package gui

import (
	"image/color"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"srpg/internal/content"
	"srpg/internal/mapedit"
)

func init() { demos["rivercross"] = demoRiverCross }

// demoRiverCross paints a two-cell river over the stage's map with a bridge and a ford across it, for comparing how
// the crossings meet the sunk water.
func demoRiverCross(d *Demo) error {
	err := d.battleWith(d.stage, func(doc *mapedit.Document) {
		data := d.g.S.Data
		G, S, T := byte(content.LayerGround), byte(content.LayerStructure), byte(content.LayerState)
		for y := 0; y < doc.Play.H; y++ {
			for x := 0; x < doc.Play.W; x++ {
				doc.Paint(data, x, y, S, '-')
				doc.Paint(data, x, y, T, '-')
				c := byte('g')
				if x == 8 || x == 9 {
					c = 'w'
					if y >= 9 && y <= 10 {
						c = 's'
					}
				}
				doc.Paint(data, x, y, G, c)
				if c == 'w' && (y == 4) {
					doc.Paint(data, x, y, S, 'b')
				}
			}
		}
		doc.Stage.Events, doc.Stage.Forced = nil, true
		doc.Stage.Allies = []content.Point{{X: 9, Y: 4}, {X: 8, Y: 10}, {X: 13, Y: 7}}
		var enemies []content.Spawn
		for i, sp := range doc.Stage.Enemies {
			if i == 2 {
				break
			}
			sp.X, sp.Y, sp.Facing, sp.Policy, sp.Goal, sp.GoalArea = 3+i, 7, "SE", "hold", nil, 0
			enemies = append(enemies, sp)
		}
		doc.Stage.Enemies = enemies
	})
	if err != nil {
		return err
	}
	g := d.g
	d.step = func(f int) {
		if f == 1 {
			g.zoom = 2
			g.camX, g.camY = g.field.Center(9, 7)
		}
	}
	return nil
}

func init() { demos["rivercross-b07"] = demoRiverCrossB07 }

// demoRiverCrossB07 lays the Dong Zhuo pursuit's map (B07) under the stage's armies, by day, and looks at its bridge.
func demoRiverCrossB07(d *Demo) error {
	err := d.battleWith(d.stage, func(doc *mapedit.Document) {
		m, err := mapedit.Load(filepath.Join("assets", "scenario", "hero", "stage07.json"))
		if err != nil {
			panic(err)
		}
		st := doc.Stage
		*doc = *m
		doc.Stage = st
		doc.Stage.Events, doc.Stage.Forced = nil, true
		doc.Stage.Allies = []content.Point{{X: 8, Y: 5}, {X: 5, Y: 7}, {X: 10, Y: 3}}
		var enemies []content.Spawn
		for i, sp := range doc.Stage.Enemies {
			if i == 2 {
				break
			}
			sp.X, sp.Y, sp.Facing, sp.Policy, sp.Goal, sp.GoalArea = 12+i, 8, "NW", "hold", nil, 0
			enemies = append(enemies, sp)
		}
		doc.Stage.Enemies = enemies
	})
	if err != nil {
		return err
	}
	g := d.g
	d.step = func(f int) {
		if f == 1 {
			g.zoom = 2
			g.camX, g.camY = g.field.Center(8, 5)
		}
	}
	return nil
}

func init() { demos["rivercross-layers"] = demoRiverCrossLayers }

// riverLayers are Field.Layers' names in drawing order, as demoRiverCrossLayers captures them.
var riverLayers = []string{"background", "ground", "floors", "walls", "decor", "waterfx", "bridge-deck", "bridge-edge",
	"objects"}

// demoRiverCrossLayers plays demoRiverCross close up on the ford and shows the field one debug layer at a time, on
// black and then on white, so the two captures of a layer give its colours and its alpha: layer i is on black at
// frame 62+6i and on white at 65+6i; the whole field is at 62+6n (n = len(riverLayers)).
func demoRiverCrossLayers(d *Demo) error {
	if err := demoRiverCross(d); err != nil {
		return err
	}
	g := d.g
	bg := color.RGBA{A: 255}
	d.step = func(f int) {
		g.clockMS = 5000 // still water, so a layer's two captures match
		if f == 1 {
			g.zoom = 2
			g.camX, g.camY = g.field.Center(8.5, 9.5)
		}
		if f < 60 {
			return
		}
		i, k := (f-60)/6, (f-60)%6
		switch {
		case i >= len(riverLayers):
			g.field.Layers = nil
		case k == 0:
			g.field.Layers, bg = map[string]bool{riverLayers[i]: true}, color.RGBA{A: 255}
		case k == 3:
			bg = color.RGBA{255, 255, 255, 255}
		}
	}
	d.overlay = func(dst *ebiten.Image) {
		if g.field.Layers == nil {
			return
		}
		dst.Fill(bg)
		inWorld(func() { drawScaled(dst, g.field.Canvas, &ebiten.DrawImageOptions{}) })
	}
	return nil
}
