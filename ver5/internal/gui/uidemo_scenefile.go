package gui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/scenes"
)

func init() { demos["scenefile"] = demoSceneFile }

// demoSceneFile opens a scene from a file outside assets/scenes (-stage path.json or path.json#sceneID) and plays its
// Ambient steps, so a new scene map can be seen with a trial cast before any story node is staged on it.
func demoSceneFile(d *Demo) error {
	path, id, _ := strings.Cut(d.stage, "#")
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f struct{ Scenes map[string]*scenes.Scene }
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	if id == "" {
		ids := make([]string, 0, len(f.Scenes))
		for k := range f.Scenes {
			ids = append(ids, k)
		}
		sort.Strings(ids)
		if len(ids) > 0 {
			id = ids[0]
		}
	}
	def := f.Scenes[id]
	if def == nil || d.g.assets.Maps[def.Map] == nil {
		return fmt.Errorf("no scene %q with a known map in %s", id, path)
	}
	def.ID = id
	d.g.openScene(def)
	d.g.zoomFrom, d.g.zoomTo = d.g.zoom, d.g.zoom
	d.replace = true
	d.overlay = func(dst *ebiten.Image) {
		dst.Fill(color.Black)
		d.g.drawScene(dst, nil)
		d.g.blackOut(dst, d.g.scene.fade)
	}
	return nil
}
