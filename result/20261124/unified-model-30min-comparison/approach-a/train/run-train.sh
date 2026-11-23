#!/bin/sh
# Approach A training, 30-minute wall-clock budget counted after setup (see result.md).
# Run from the frozen working-tree snapshot: /private/tmp/claude-501/-Users-younwookim-orca-mhdemo1-ver5/e1c24e1a-75cd-4342-808c-482480821a91/scratchpad/frozen/ver5
cd "/private/tmp/claude-501/-Users-younwookim-orca-mhdemo1-ver5/e1c24e1a-75cd-4342-808c-482480821a91/scratchpad/frozen/ver5" && "/private/tmp/claude-501/-Users-younwookim-orca-mhdemo1-ver5/e1c24e1a-75cd-4342-808c-482480821a91/scratchpad/frozen/unifiedlearn" train -content assets/data -budget 30m \
  -warm 8 -warm-epochs 20 -bc 8 -bc-epochs 2 -collect 8 -par 4 -dev-games 4 \
  -seed 220001 -dev-seed 230001 -bc-seed 225001 -battle-temperature 0.25 -hidden 32 -init-seed 7 \
  -model "/Users/younwookim/orca/mhdemo1/result/20261124/unified-model-30min-comparison/approach-a/train/model" -out "/Users/younwookim/orca/mhdemo1/result/20261124/unified-model-30min-comparison/approach-a/train"
