package main

import (
	"demo1/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
)

type Camera struct {
	Zoom       int
	PanX, PanY float64
}

// screenRenderer uploads each sheet once and retains subimage views for its cells.
type screenRenderer struct {
	catalog *graphics.Catalog
	sprites []*ebiten.Image
}

func newScreenRenderer(c *graphics.Catalog) *screenRenderer {
	sheets := make([]*ebiten.Image, len(c.Images))
	for i, im := range c.Images {
		sheets[i] = ebiten.NewImageFromImage(im)
	}
	r := &screenRenderer{catalog: c}
	for i, s := range c.Sprites {
		r.sprites = append(r.sprites, sheets[s.Sheet].SubImage(c.Rect(i)).(*ebiten.Image))
	}
	return r
}
func (r *screenRenderer) Draw(dst *ebiten.Image, items []graphics.DrawItem, camera Camera) {
	for _, item := range items {
		p := item.Position.Sub(r.catalog.Sprites[item.Sprite].Pivot)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(camera.Zoom), float64(camera.Zoom))
		op.GeoM.Translate(camera.PanX+float64(p.X*camera.Zoom), camera.PanY+float64(p.Y*camera.Zoom))
		dst.DrawImage(r.sprites[item.Sprite], op)
	}
}
func (g *game) buildDrawList() []graphics.DrawItem {
	scene := g.sceneCache.Ensure(g.w)
	var err error
	g.drawList, err = scene.BuildDrawList(g.catalog, g.waterFrame, g.treeTick, nil, g.drawList)
	if err != nil {
		panic(err)
	} // Catalog is validated once at startup.
	return g.drawList
}
func (g *game) exportImage(scale int) *image.NRGBA {
	items := g.buildDrawList()
	return graphics.RenderPNG(g.catalog, items, graphics.SceneBounds(g.catalog, items, g.sceneCache.Ensure(g.w).Bounds), scale)
}
