package terrain

const TileWidth, TileHeight = 16, 8
const AssetCount = 38
const GrassAsset = 23

func Layers(mask, frame int) []int {
	if mask == 15 {
		return []int{8}
	}
	if mask == 0 {
		return []int{frame % 8}
	}
	return []int{frame % 8, 8 + mask}
}

// The grass mask includes all dry terrain; wasteland is the top overlay.
func SurfaceLayers(ground, land, frame int) []int {
	if land == 15 {
		return []int{8}
	}
	layers := []int{}
	if ground == 15 {
		layers = append(layers, GrassAsset)
	} else {
		layers = append(layers, frame%8)
		if ground != 0 {
			layers = append(layers, GrassAsset+ground)
		}
	}
	if land != 0 {
		layers = append(layers, 8+land)
	}
	return layers
}
