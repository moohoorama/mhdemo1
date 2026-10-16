package gui

func init() { demos["damage"] = demoDamage }

// demoDamage follows the mockup: a hit takes the enemy from 78% to 24%, a heal takes the
// ally from 36% to 86%, then a critical hit takes the enemy from 78% to 8%.
func demoDamage(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	ally, enemy, err := d.pair()
	if err != nil {
		return err
	}
	d.setHP(enemy, .78)
	d.setHP(ally, .36)
	d.g.sync(d.g.S.Engine.Observe())
	d.step = func(f int) {
		switch f {
		case 30:
			d.hit(ally, enemy, .78, .24, false)
		case 150:
			d.hit(ally, ally, .36, .86, false)
		case 270:
			d.hit(ally, enemy, .78, .08, true)
		}
	}
	return nil
}
