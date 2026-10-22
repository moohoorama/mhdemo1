---
name: item-icon
description: Makes or remakes item (equipment) icons of the ver5 UI sheet as dot art by painting them with the codex-image skill on flat magenta and converting the painting with tools/pixelize_icon.py (palette snap, block vote, 1px outline) — never by hand-placing dots. Use for "아이템 아이콘", "장비 아이콘 추가/다시", "새 장비 도트", "아이콘이 각져/어색해", or whenever a new equipment entry needs its Icon.
---

# Item icon by painting → dot conversion

The user's rule (2026-10-22): item dots are made this way. Hand-drawn agent dots came out boxy ("너무 각져") and a
hand retouch was still "별로"; this route was "훨씬 좋아" and then "이제 아이템 도트는 이런 식으로 하자".

Files: sheet source `tools/spritetool/assets/ver5-ui/` (`settings.json`, `pixels.json`, `sources/`, README),
tool `ver5/tools/pixelize_icon.py`, sheet build `ver5/tools/build_effects.py`, data `ver5/assets/data/equipment.json`
(`Icon` → sprite `item_<Icon>`), check `ver5/internal/gui/equipment_window_test.go`.

## Steps

1. **Name and data.** Pick `<icon>` (short English, shared by look-alike items if wanted) and set the item's `"Icon"`
   in `equipment.json`. New sprite in `settings.json`: add `item_<icon>` to `frame_order` (beside the other `item_*`) and
   ```json
   "item_<icon>": {"cell": [144, 144], "anchor": [8, 8], "dot": 4, "source": "sources/item_<icon>.png",
     "frames": 1, "timing": "...", "layers": {"icon": {"palette": {"k": "#1c120c", ...}}}}
   ```
   The palette is the dot colours: `k` the outline, then 2–4 step ramps per material (shadow → highlight) and a few
   accents, about 8–12 colours. Fewer colours look more like dots; a ramp too coarse loses the shading.
2. **Paint with codex** (`codex-image` skill, ~1–3 min each; several icons in parallel with `xargs -P 6`).
   Reference image: the current icon x16 on magenta when remaking one; none for a new item. Prompt template:
   > A 32x32 pixel art RPG inventory icon for an ancient Chinese Three Kingdoms era game, shown enlarged with big crisp
   > square pixels: <object, materials, colours, identity>. <pose>. Rounded organic silhouette, not boxy. Very simple:
   > few large shapes, cel shading with 3 tones per material plus a highlight, no texture, no noise, dark outline.
   > Centred, about 70 percent of the frame. Background: perfectly flat solid magenta #FF00FF, no shadow. No text, no
   > letters, no characters, no watermark.
   - Long weapons: "lying diagonally at 45 degrees, grip at the bottom-left and tip at the top-right".
   - Armour/clothes: "slight three-quarter view"; books/scrolls tilted ~25°. No Hanja or text anywhere (Korean-only UI).
   - A painterly prompt (textures, ornament) shrinks into blotches — keep the "32x32 pixel art, few large shapes" wording.
   - Nothing in the item may be magenta/purple-pink: the key removes it.
3. **Store the source**: resize the PNG to 512×512 (LANCZOS) into `sources/item_<icon>.png`.
4. **Convert**: from `ver5/`, `uv run --with pillow python tools/pixelize_icon.py item_<icon> ...` (writes `pixels.json`).
   Knobs in the sprite's settings: `"solid"` (share of a dot's block that must be covered; default .5, swords/spears .35,
   bow .2 — thin blades and strings vanish otherwise), `"fit"` (dots across, default 30), `"use"` (only these palette
   letters). Changing the palette and re-running is the normal way to tune colours.
5. **Look** at x4 nearest and at x1 on the window colour `#2a201a` (a small PIL sheet in the scratchpad), all icons of
   the batch side by side with their neighbours. Typical faults: a thin part lost (lower `solid`), a dark area filled
   between limbs/strings, colours mapped to the wrong ramp (fix the palette), too busy (repaint with a simpler prompt).
6. **Build and see it in the game**: `uv run --with pillow python tools/build_effects.py`, then
   `caffeinate -d -i go run ./cmd/_uidemo-vm -ui equip -frames 250 -at 100,240 -size 2560x1700` (`EQUIP_SLOT=armor` or
   `accessory` opens that slot's picker). Run `go test ./internal/gui -run TestEveryEquipment`.
7. **Record** in `ver5/docs/history.md` (`파일이름: 수정내역 — 의도`) and report with the capture paths.
