# AGENTS.md

Read `ver5/README.md` before modifying this repository. The game lives in `ver5/`.

Sprites the game reads are `ver5/assets/sprites/*.spr`, built from the dot editor YAML
(`ver5/tools/doteditor/units/*.yaml` via `ver5/tools/spritebuild/build.py`; see `ver5/docs/sprite-binary.md`).
Do not add project-specific sprite transformations to `tools/spritetool`. Keep runtime
assets under `ver5/assets/` and source artwork under `tools/spritetool/assets/`.

Before creating or modifying any sprite, read
`tools/spritetool/assets/SPRITE_ART_GUIDE.md` and the target asset's settings
or README. Apply the common rules and the relevant asset-type rules to both
candidate previews and final artwork.

For unit sprites (soldiers, cavalry, archers, heroes), also read
`tools/spritetool/assets/UNIT_ART_GUIDE.md`. `tools/unit3d/` is the old 3D cel-shading pipeline, kept as reference;
new dot work goes through the dot editor (`hero-sprite` skill).

Illustrations (officer portraits, event CGs, backgrounds, UI art) are not
sprites: their sources live under `tools/illustrations/`. Read
`tools/illustrations/README.md` first and keep the chosen ink-wash style.
