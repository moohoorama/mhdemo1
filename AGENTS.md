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

To check UI or on-screen output, capture it instead of guessing from code. `ver5/cmd/_uidemo-vm`
plays a named UI demo (`ver5/internal/gui/uidemo_*.go`) headlessly and saves PNG frames and/or a
60 fps MP4; run it from `ver5/` wrapped in `caffeinate -d -i`, e.g.
`go run ./cmd/_uidemo-vm -ui dialogue -frames 300 -at 30,96 -mp4 reports/uidemo/dialogue.mp4`.
`-size 2400x1080 -uiscale 160` plays it on a phone-sized screen, and `-sheet -at 260,380,720` puts the
same frames from the four reference screens (docs/plans/PLAN_ui-scale-ppi.md) on one image.
Add a new demo file there when no existing demo shows the screen. `cmd/_uidemo` is the windowed version.
