package gui

import "fmt"

func init() { demos["titlereel"] = demoTitleReel }

// demoTitleReel is the title screen with its reel starting at the shot -node names (the first without one).
func demoTitleReel(d *Demo) error {
	r := d.g.reel
	if r == nil {
		return fmt.Errorf("no title reel")
	}
	if d.node == "" {
		return nil
	}
	for i, s := range r.reel.Shots {
		if s.ID == d.node {
			r.i = i
			return nil
		}
	}
	return fmt.Errorf("no shot %q in the title reel", d.node)
}
