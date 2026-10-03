# Knight motion v3

Native 48×48 RGBA cells; each direction has four columns and five rows:
idle, walk, attack, hit, exhausted. Pivot: (24,38).

The immutable v2 pixel sheets are the input. This revision reuses their pixel
clusters, without resampling or generating new helmet/eye patterns:

- Idle: frame 1 of idle, with the chest raised one pixel on frames 2–3; feet fixed.
- Walk: idle-derived left contact, passing, right contact, passing. Arms oppose
  the corresponding feet. Passing frames differ in the lifted foot. Profile
  directions reuse the visible boot for the otherwise hidden far leg.
- Attack: all four v2 frames preserved byte-for-byte at pixel level.
- Hit: two identical standing frames, exaggerated recoil, partial recovery.
  The torso leans with integer row offsets; the helmet remains a rigid cluster.
- Exhausted: frame 1 of exhausted, alternating one-pixel chest lifts, no scaling.

`rig.json` records the direction-specific masks. `frames.json` includes suggested
frame durations in milliseconds. The preview uses these by default, with manual
FPS overrides and frame selection. Moving limbs naturally change screen-space
pixel agreement; shared details retain their original colors and geometry.

Rebuild from the repository root:

```sh
python3 tools/animate_knight_pixels.py
python3 tools/build_knights.py
```

The generic spritetool only crops and packs the already animated frames.
Runtime output is `assets/knights.png` and `assets/knights.json`.
The preview also retains `assets/knights-v2.png` and the earlier v1 atlas.

Before this revision, all repository changes were staged with `git add -A`.
The original index tree is also retained as
`refs/checkpoints/knights-before-motion` (tree
`5725f4c0042227f8cbe50507dd6aea9872a3f15d`). The v3 changes are intentionally
unstaged, so the staged snapshot remains the requested rollback baseline.
To recover a specific pre-revision file without altering the index:

```sh
git restore --source=refs/checkpoints/knights-before-motion --worktree -- path/to/file
```

This restores paths present in the snapshot; new v3 files remain separately.
The checkpoint is a Git tree reference, not a branch commit.
