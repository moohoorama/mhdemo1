package terrain

const ObjectSpriteCount = 23

func ObjectSprite(object Object, frame int) int {
	if object >= LargeTree {
		return 8 + int(object-LargeTree)
	}
	if object >= GrassTuft1 && frame%4 != 0 {
		return 11 + int(object-GrassTuft1)*3 + frame%4 - 1
	}
	return int(object) - 1
}
