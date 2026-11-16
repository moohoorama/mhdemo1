#!/usr/bin/env bash
# usage: run2.sh <key> <new|edit> <prompt> <ref...>
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
k=$1 mode=$2 p=$3; shift 3
style=" Waist-up portrait, square 1:1 image. This is a portrait for a serious Three Kingdoms strategy game: he must look impressive and dignified, a mature man's face that is clearly his own, never goofy, never fat, never sickly, no grin or laughing. No large anime eyes. No text, no names, no seals, no watermark, no border. Style: East Asian ink wash painting with light watercolor tints, expressive calligraphic brush strokes, soft rice-paper texture background. Match the brushwork, palette and paper of the reference image."
[[ $mode == edit ]] && style=" Keep the same brushwork, palette, rice paper and framing. No text, no seals, no watermark, no border."
~/.claude/skills/codex-image/codex-image.sh "$here/$k.png" "$p$style" "$@" > "$here/$k.log" 2>&1 && echo "done $k" || echo "FAIL $k"
