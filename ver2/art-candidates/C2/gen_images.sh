#!/bin/bash
# Generate C2 source artwork with Codex's built-in image generation.
# Usage: gen_images.sh <name> <prompt>   (writes C2/src/<name>.png and <name>.prompt.txt)
set -euo pipefail
cd "$(dirname "$0")"
name=$1; prompt=$2
echo "$prompt" > "src/$name.prompt.txt"
codex exec --skip-git-repo-check -s workspace-write \
  "Use your built-in image generation tool (do not write code or draw with scripts) to generate exactly ONE image. $prompt
After generating, copy the generated PNG to $(pwd)/src/$name.png and reply only with that path." \
  > "src/$name.log" 2>&1
test -f "src/$name.png" && echo "ok $name" || echo "FAIL $name"
