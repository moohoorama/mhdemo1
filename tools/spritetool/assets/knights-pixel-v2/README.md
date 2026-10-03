# Knight pixel artwork v2

Directions: N, NE, E, SE, S, SW, W, NW.
Rows: idle, walk, attack, hit, exhausted. Four frames per row.

- `*-normalized-reference.png`: original artwork scaled using tracked helmet landmarks, with idle head centers aligned.
- `*-redrawn.png`: built-in ImageGen pixel-art redraws using the normalized direction sheet and the south redraw as style reference. Keep the source alpha.
- `*-pixel.png`: final native 192×240 RGBA sheets, 4×5 cells of 48×48. Shared 28-color palette; binary alpha; no resampling at runtime.
- `normalization.json`: source helmet measurements.
- `frames.json`: native frames and redraw helmet measurements. Helmets target a geometric width/height mean of 10 pixels, preserving their directional proportions. Idle uses four identical copies. Other actions keep relative head movement.

The ground pivot is (24,38). Extra side padding preserves sword trails. These sources are packed into `assets/knights.png` (768×480, 16×10 cells) and `assets/knights.json`. V1 is retained for comparison.

Rebuild the runtime atlas from the checked-in native art:

```sh
python3 tools/build_knights.py
```

Regenerate native art from retained redraws (requires Pillow and NumPy), then pack:

```sh
python3 tools/finalize_knight_pixels.py
python3 tools/build_knights.py
```

To regenerate normalized reference sheets from original transparent images:

```sh
python3 tools/prepare_knight_references.py
```

The project-specific landmarks and palette live in these recipes, outside the generic `tools/spritetool` code.

ImageGen prompt summary: redraw the normalized sheet as tiny game pixel art, preserve all 20 original poses and the facing direction, normalize helmet size, make idle copies motionless, use clear connected pixel clusters and a limited palette, preserve sword attacks, label row meanings as idle/walk/attack/hit/exhausted, remove margin debris, and require actual alpha transparency. The S sheet was generated first and supplied as the style reference for the other seven directions. Final palette and frame alignment are deterministic build steps.
