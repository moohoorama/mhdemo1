package terrain

const ObjectSpriteCount = 32

func ObjectSprite(object Object, frame int) int {
	if object >= LargeTree && frame%4 != 0 {
		return 23 + int(object-LargeTree)*3 + frame%4 - 1
	}
	if object >= LargeTree {
		return 8 + int(object-LargeTree)
	}
	if object >= GrassTuft1 && frame%4 != 0 {
		return 11 + int(object-GrassTuft1)*3 + frame%4 - 1
	}
	return int(object) - 1
}
