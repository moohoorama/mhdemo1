package terrain

const ObjectSpriteCount = 28

func ObjectSprite(object Object, frame int) int {
	if object >= LargeTree {
		return 8 + int(object-LargeTree)*4 + frame%4
	}
	if object >= GrassTuft1 && frame%4 != 0 {
		return 16 + int(object-GrassTuft1)*3 + frame%4 - 1
	}
	return int(object) - 1
}
