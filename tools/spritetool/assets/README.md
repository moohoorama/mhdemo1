# Sprite source artwork

These PNGs are source artwork for the runtime object atlas. Run `make objects`
from the repository root after changing them.

Animated artwork contains four equal horizontal frames in playback order. The
grass files are deliberately stored at logical game resolution; their roots are
already fixed across all four authored poses. The tree strips in `tree-sway/`
retain their larger source artwork and are sampled by the Makefile recipe; see
`tree-sway/README.md`. `tree1.png`–`tree3.png` are the still trees they were
drawn from. `tree_0.png` and `tree_1.png` are older, unused tree artwork.
