# Sprite source artwork

These PNGs are source artwork for the runtime object atlas. Run `make objects`
from the repository root after changing them.

Animated artwork contains four equal horizontal frames in playback order. The
grass files are deliberately stored at logical game resolution; their roots are
already fixed across all four authored poses. The tree files retain their larger
source artwork and are sampled by the Makefile recipe. `tree_0.png` retains two
blank trailing columns from the original; the recipe crops those columns before
slicing its four equal frames.
