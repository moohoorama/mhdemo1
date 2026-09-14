ASSET_TEMP ?= .asset_temp
ASSET_DIR ?= assets
SPRITE_SOURCE ?= tools/spritetool/assets
SPRITETOOL := go run ./tools/spritetool
TILEGENERATOR := go run ./tools/tilegenerator

.PHONY: assets terrain objects clean-assets

assets: terrain objects

terrain:
	$(TILEGENERATOR) $(ASSET_DIR)/terrain.png

objects:
	rm -rf -- "$(ASSET_TEMP)"
	mkdir -p "$(ASSET_TEMP)"
	$(SPRITETOOL) --trim --size 8x6  --outline 1 $(SPRITE_SOURCE)/rock_small.png   $(ASSET_TEMP)/00.png
	$(SPRITETOOL) --trim --size 11x8 --outline 1 $(SPRITE_SOURCE)/rock_medium.png  $(ASSET_TEMP)/01.png
	$(SPRITETOOL) --trim --size 10x9 --outline 1 $(SPRITE_SOURCE)/rock_medium2.png $(ASSET_TEMP)/02.png
	$(SPRITETOOL) --trim --size 14x12 --outline 1 $(SPRITE_SOURCE)/rock_large.png   $(ASSET_TEMP)/03.png
	$(SPRITETOOL) --frames 4 --frame 0 --outline 1 $(SPRITE_SOURCE)/grass_0.png $(ASSET_TEMP)/04.png
	$(SPRITETOOL) --frames 4 --frame 0 --outline 1 $(SPRITE_SOURCE)/grass_1.png $(ASSET_TEMP)/05.png
	$(SPRITETOOL) --frames 4 --frame 0 --outline 1 $(SPRITE_SOURCE)/grass_2.png $(ASSET_TEMP)/06.png
	$(SPRITETOOL) --frames 4 --frame 0 --outline 1 $(SPRITE_SOURCE)/grass_3.png $(ASSET_TEMP)/07.png
	$(SPRITETOOL) --crop 0,0,1772,887 --frames 4 --frame 0 --trim --size 22x20 --outline 1 $(SPRITE_SOURCE)/tree_0.png $(ASSET_TEMP)/08.png
	$(SPRITETOOL) --crop 0,0,1772,887 --frames 4 --frame 1 --trim --size 22x20 --outline 1 $(SPRITE_SOURCE)/tree_0.png $(ASSET_TEMP)/09.png
	$(SPRITETOOL) --crop 0,0,1772,887 --frames 4 --frame 2 --trim --size 22x20 --outline 1 $(SPRITE_SOURCE)/tree_0.png $(ASSET_TEMP)/10.png
	$(SPRITETOOL) --crop 0,0,1772,887 --frames 4 --frame 3 --trim --size 22x20 --outline 1 $(SPRITE_SOURCE)/tree_0.png $(ASSET_TEMP)/11.png
	$(SPRITETOOL) --frames 4 --frame 0 --trim --size 20x18 --outline 1 $(SPRITE_SOURCE)/tree_1.png $(ASSET_TEMP)/12.png
	$(SPRITETOOL) --frames 4 --frame 1 --trim --size 20x18 --outline 1 $(SPRITE_SOURCE)/tree_1.png $(ASSET_TEMP)/13.png
	$(SPRITETOOL) --frames 4 --frame 2 --trim --size 20x18 --outline 1 $(SPRITE_SOURCE)/tree_1.png $(ASSET_TEMP)/14.png
	$(SPRITETOOL) --frames 4 --frame 3 --trim --size 20x18 --outline 1 $(SPRITE_SOURCE)/tree_1.png $(ASSET_TEMP)/15.png
	$(SPRITETOOL) --frames 4 --frame 1 --outline 1 $(SPRITE_SOURCE)/grass_0.png $(ASSET_TEMP)/16.png
	$(SPRITETOOL) --frames 4 --frame 2 --outline 1 $(SPRITE_SOURCE)/grass_0.png $(ASSET_TEMP)/17.png
	$(SPRITETOOL) --frames 4 --frame 3 --outline 1 $(SPRITE_SOURCE)/grass_0.png $(ASSET_TEMP)/18.png
	$(SPRITETOOL) --frames 4 --frame 1 --outline 1 $(SPRITE_SOURCE)/grass_1.png $(ASSET_TEMP)/19.png
	$(SPRITETOOL) --frames 4 --frame 2 --outline 1 $(SPRITE_SOURCE)/grass_1.png $(ASSET_TEMP)/20.png
	$(SPRITETOOL) --frames 4 --frame 3 --outline 1 $(SPRITE_SOURCE)/grass_1.png $(ASSET_TEMP)/21.png
	$(SPRITETOOL) --frames 4 --frame 1 --outline 1 $(SPRITE_SOURCE)/grass_2.png $(ASSET_TEMP)/22.png
	$(SPRITETOOL) --frames 4 --frame 2 --outline 1 $(SPRITE_SOURCE)/grass_2.png $(ASSET_TEMP)/23.png
	$(SPRITETOOL) --frames 4 --frame 3 --outline 1 $(SPRITE_SOURCE)/grass_2.png $(ASSET_TEMP)/24.png
	$(SPRITETOOL) --frames 4 --frame 1 --outline 1 $(SPRITE_SOURCE)/grass_3.png $(ASSET_TEMP)/25.png
	$(SPRITETOOL) --frames 4 --frame 2 --outline 1 $(SPRITE_SOURCE)/grass_3.png $(ASSET_TEMP)/26.png
	$(SPRITETOOL) --frames 4 --frame 3 --outline 1 $(SPRITE_SOURCE)/grass_3.png $(ASSET_TEMP)/27.png
	$(SPRITETOOL) --atlas 24x32,8x4 --atlas-pivot 12,28 --sprite-pivot center,bottom-2 $(ASSET_TEMP)/*.png $(ASSET_DIR)/objects.png
	rm -rf -- "$(ASSET_TEMP)"

clean-assets:
	rm -f -- "$(ASSET_DIR)/terrain.png" "$(ASSET_DIR)/objects.png"
	rm -rf -- "$(ASSET_TEMP)"
