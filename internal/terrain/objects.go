package terrain

// Object identifies a sprite; every subtile has at most one object.
type Object uint8

const (
	Empty Object = iota
	SmallRock
	MediumRock
	MediumRock2
	LargeRock
	GrassTuft1
	GrassTuft2
	GrassTuft3
	GrassTuft4
	LargeTree
	SmallTree
)

// Coordinate hashing makes decoration stable through repaint, undo and loading.
// Separate random streams keep object choice independent of occupancy.
func decorationHash(x, y, slot, stream int) uint64 {
	n := uint64(uint32(x))<<32 | uint64(uint32(y))
	n ^= uint64(slot+1) * 0x9e3779b97f4a7c15
	n ^= uint64(stream+1) * 0xd1b54a32d192ed03
	n = (n ^ (n >> 30)) * 0xbf58476d1ce4e5b9
	n = (n ^ (n >> 27)) * 0x94d049bb133111eb
	return n ^ (n >> 31)
}
func roll(x, y, slot, stream, n int) int { return int(decorationHash(x, y, slot, stream) % uint64(n)) }
func tuft(x, y, slot int) Object         { return GrassTuft1 + Object(roll(x, y, slot, 1, 4)) }
func tree(x, y, slot int) Object         { return LargeTree + Object(roll(x, y, slot, 2, 2)) }

func (w *World) Objects(x, y int) [4]Object {
	var result [4]Object
	if !w.Inside(x, y) {
		return result
	}
	cell := w.Cell(x, y)
	if cell.Decoration == Trees {
		// Forest is a tile-scale decoration: one broad tree per large tile,
		// anchored at its center rather than one tree per subtile.
		result[0] = tree(x, y, 0)
		return result
	}
	for slot := range result {
		switch {
		case cell.Decoration == NoDecoration && cell.Ground == Grass:
			if roll(x, y, slot, 0, 4) == 0 {
				result[slot] = tuft(x, y, slot)
			}
		case cell.Decoration == Rocks:
			pick := roll(x, y, slot, 0, 5)
			if pick < 4 {
				result[slot] = SmallRock + Object(pick)
			}
		}
	}
	return result
}

// Placements are sorted back-to-front by their subtile foot position.
type Placement struct {
	X, Y, Slot int
	Object     Object
}

func (w *World) Placements() []Placement {
	var result []Placement
	for diagonal := 0; diagonal < 2*w.Width+2*w.Height-1; diagonal++ {
		for sy := 0; sy < 2*w.Height; sy++ {
			sx := diagonal - sy
			if sx < 0 || sx >= 2*w.Width {
				continue
			}
			x, y, slot := sx/2, sy/2, (sy%2)*2+sx%2
			if object := w.Objects(x, y)[slot]; object != Empty {
				result = append(result, Placement{x, y, slot, object})
			}
		}
	}
	return result
}
func (p Placement) Position() (float64, float64) {
	if p.Object == LargeTree || p.Object == SmallTree {
		return Project(float64(p.X)+.5, float64(p.Y)+.5)
	}
	return Project(float64(p.X)+float64(p.Slot%2)/2+.25, float64(p.Y)+float64(p.Slot/2)/2+.25)
}

// At 1x one tick is 125ms. Each plant gets an independent offset within the 8s loop,
// including sub-frame timing, so neighboring trees don't all change together.
const TreeFrameTicks = 16
const TreeCycleTicks = 4 * TreeFrameTicks

func (p Placement) TreeFrame(tick int) int {
	return ((tick%TreeCycleTicks + roll(p.X, p.Y, p.Slot, 6, TreeCycleTicks)) / TreeFrameTicks) % 4
}
func (p Placement) Sprite(tick int) int {
	if p.Object >= GrassTuft1 {
		return ObjectSprite(p.Object, p.TreeFrame(tick))
	}
	return ObjectSprite(p.Object, 0)
}
