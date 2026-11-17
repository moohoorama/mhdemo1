package gui

import "fmt"

func init() { demos["scenenode4"] = demoSceneNode4 }

// demoSceneNode4 is scenenode with the walk's zoom at 4, the closest the player can set, printing "LINE node line
// frame" as each line comes up: for checking that the camera keeps whoever acts or speaks in view.
func demoSceneNode4(d *Demo) error {
	if err := demoSceneNode(d); err != nil {
		return err
	}
	g := d.g
	g.S.Settings.ExploreZoom = 4
	step, last := d.step, ""
	d.step = func(f int) {
		step(f)
		if s := g.scene; s != nil {
			if k := fmt.Sprintf("%s %d", s.node, g.lineIndex); k != last {
				last = k
				fmt.Println("LINE", k, f)
			}
		}
	}
	return nil
}
