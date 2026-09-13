package terrain

import (
	"demo1/internal/graphics"
	"fmt"
	"image"
)

type GroundTile struct {
	Position image.Point
	Layers   []int // Water uses asset 0 as its animation placeholder.
}

type DecorationPlacement struct {
	Position image.Point
	Object   Object
	Phase    int
}

// MapScene contains only derived data. No masks or placement hashes are evaluated during drawing.
type MapScene struct {
	Ground      []GroundTile
	Decorations []DecorationPlacement
	Bounds      image.Rectangle
}

func BuildMapScene(w *World) *MapScene {
	s := &MapScene{Bounds: image.Rect(-w.Height*16-16, -16, w.Width*16+16, (w.Width+w.Height)*8+16)}
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			ground, soil := w.MasksFor(x, y, Grass), w.Masks(x, y)
			for k, m := range soil {
				px, py := Project(float64(x)+float64(k%2)/2, float64(y)+float64(k/2)/2)
				s.Ground = append(s.Ground, GroundTile{Position: image.Pt(int(px), int(py)), Layers: SurfaceLayers(ground[k], m, 0)})
			}
		}
	}
	for _, p := range w.Placements() {
		x, y := p.Position()
		s.Decorations = append(s.Decorations, DecorationPlacement{Position: image.Pt(int(x), int(y)), Object: p.Object, Phase: roll(p.X, p.Y, p.Slot, 6, TreeCycleTicks)})
	}
	return s
}

// SceneCache detects both edits to the same world and replacement on undo/load.
type SceneCache struct {
	world    *World
	revision uint64
	scene    *MapScene
}

func (c *SceneCache) Ensure(w *World) *MapScene {
	if c.scene == nil || c.world != w || c.revision != w.Revision() {
		c.world, c.revision, c.scene = w, w.Revision(), BuildMapScene(w)
	}
	return c.scene
}

// BuildDrawList resolves animation frames only. extra accepts future moving sprites
// in projected coordinates; layer 1 interleaves them with decorations by foot depth.
// dst may be reused across frames, but must not alias extra.
func (s *MapScene) BuildDrawList(c *graphics.Catalog, waterTick, plantTick int, extra, dst []graphics.DrawItem) ([]graphics.DrawItem, error) {
	dst = dst[:0]
	water, ok := c.Animations["water"]
	if !ok {
		return nil, fmt.Errorf("missing water animation")
	}
	for _, t := range s.Ground {
		for _, id := range t.Layers {
			if id == 0 {
				id = water.Frame(waterTick)
			}
			dst = append(dst, graphics.DrawItem{Sprite: id, Position: t.Position, Layer: 0})
		}
	}
	for _, p := range s.Decorations {
		a, ok := c.Animations[ObjectAnimation(p.Object)]
		if !ok {
			return nil, fmt.Errorf("missing animation for object %d", p.Object)
		}
		dst = append(dst, graphics.DrawItem{Sprite: a.Frame(plantTick + p.Phase), Position: p.Position, Layer: 1, Depth: p.Position.Y})
	}
	// Cached terrain order is already stable. Sort only when moving sprites are supplied.
	if len(extra) > 0 {
		dst = append(dst, extra...)
		graphics.Sort(dst)
	}
	return dst, nil
}

func ObjectAnimation(object Object) string {
	names := [...]string{"", "rock_small", "rock_medium", "rock_medium2", "rock_large", "grass_0", "grass_1", "grass_2", "grass_3", "tree_0", "tree_1"}
	return names[object]
}

// ValidateAssets checks the terrain-specific contract after generic catalog validation.
func ValidateAssets(c *graphics.Catalog) error {
	if len(c.Sprites) < AssetCount {
		return fmt.Errorf("terrain catalog needs %d ground sprites", AssetCount)
	}
	for i := 0; i < AssetCount; i++ {
		if c.Rect(i).Size() != image.Pt(16, 8) || c.Sprites[i].Pivot != image.Pt(8, 0) {
			return fmt.Errorf("invalid ground sprite %d", i)
		}
	}
	water, ok := c.Animations["water"]
	if !ok || len(water.Frames) != 8 || water.FrameTicks != 1 || !water.Loop {
		return fmt.Errorf("invalid water animation")
	}
	for i, id := range water.Frames {
		if i != id {
			return fmt.Errorf("water frames must reference ground sprites 0-7")
		}
	}
	for o := SmallRock; o <= SmallTree; o++ {
		a, ok := c.Animations[ObjectAnimation(o)]
		want := 1
		if o >= GrassTuft1 {
			want = 4
		}
		if !ok || len(a.Frames) != want || a.FrameTicks != TreeFrameTicks || !a.Loop {
			return fmt.Errorf("invalid object animation %s", ObjectAnimation(o))
		}
	}
	return nil
}
