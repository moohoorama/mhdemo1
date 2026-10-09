#!/usr/bin/env bash
# Six ink-wash portraits on one 3x2 sheet (see tools/illustrations/README.md).
# usage: tools/illustrations/portrait_sheet.sh <sheet name> "<six cells: appearance, pose, background per person>"
cd "$(dirname "$0")/../.."  # repository root
HEAD="One landscape 3:2 image containing a 3 x 2 grid of six separate portraits of Chinese Three Kingdoms era people. Each cell is its own square portrait of exactly one person, from the chest or waist up, with a thin even gap of plain rice paper between cells so the six can be cropped apart. Same painting style as the reference image: East Asian ink wash painting with light watercolor tints, expressive calligraphic brush strokes, rice-paper texture. Give every cell a different pose, gesture and head angle (some facing left, some right, some toward the viewer), and a different background motif painted in light ink wash exactly as described for that person; never repeat the same mountain background. Keep every person clearly different."
TAIL="No text, no names, no numbers, no seals, no watermark, no border lines."
~/.claude/skills/codex-image/codex-image.sh "tools/illustrations/portraits/sheets/$1.png" "$HEAD $2 $TAIL" tools/illustrations/style-samples/4-ink-wash.png 2>&1 | tail -1
