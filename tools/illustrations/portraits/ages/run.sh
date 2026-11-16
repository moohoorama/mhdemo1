#!/usr/bin/env bash
# usage: run.sh <key> <ref-portrait-key> <prompt>
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
tail=" Keep him recognisably the same man as in the reference portrait. Waist-up portrait, square 1:1 image. This is a portrait for a serious Three Kingdoms strategy game: he must look impressive and dignified, a mature man's face that is clearly his own, never goofy, never fat, never sickly, no grin or laughing. No large anime eyes. No text, no names, no seals, no watermark, no border. Style: East Asian ink wash painting with light watercolor tints, expressive calligraphic brush strokes, soft rice-paper texture background. Match the brushwork, palette and paper of the reference image."
~/.claude/skills/codex-image/codex-image.sh "$here/$1.png" "$3$tail" "$here/../$2.png" > "$here/$1.log" 2>&1 && echo "done $1" || echo "FAIL $1"
