"""The small roadside town of chapter 5 S#2 (낙양 가는 길의 작은 고을), where the brothers stop at dusk and meet Cao Cao.

  python3 ver5/plan/draft_maps/road_town/gen.py      -> ver5/plan/draft_maps/road_town/road_town.json

One dirt road runs through from the east (where the brothers come in) to the west (toward Luoyang); the tavern on its
north side opens into S_tavern2; a few thatched houses, a market corner and a well. Copied into the game as
assets/graphics/maps/S_road_town.json. Uses the kit of ../danyang_village/gen.py."""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / 'danyang_village'))
import gen as kit  # noqa: E402  (writes the village drafts too; they are reproducible)

kit.OUT = Path(__file__).resolve().parent

c = kit.Map('S_draft_road_town', '낙양 가는 길 · 작은 고을', 10, 8, seed=21)
# the road through the town, a lane to the market corner
c.paint(-12, 7, 31, 8)
c.paint(7, 9, 12, 12)
c.paint(9, 5, 10, 6)
# north side: houses and the tavern (door on its SW face, onto the road)
c.put('house_thatch', 3, 3)
c.put('tavern', 10, 3)
c.put('lantern', 8, 5); c.put('lantern', 12, 5)
c.put('wine_jar', 13, 4); c.put('bench', 6, 5)
c.put('house_thatch_x', 16, 3)
c.put('hut', 19, 4)
# south side: market corner, the well, houses
c.put('market_stall', 8, 11); c.put('market_stall_cloth', 11, 11)
c.put('grain_sack', 9, 13); c.put('water_jar', 12, 13)
c.put('well', 15, 11); c.put('water_bucket', 16, 12)
c.put('house_thatch_x', 3, 12)
c.put('house_thatch', 18, 13)
c.put('notice_post', 6, 9)
c.put('milestone', 19, 9)
# fences round two yards
c.fence_line(1, 5, 5, 5)
c.fence_line(16, 15, 20, 15)
c.trees(5, (-10, -10, 21, 0), gap=4)
c.trees(4, (-10, 1, -1, 25), gap=4)
c.trees(3, (22, 14, 31, 25), gap=4)
c.trees(2, (0, 20, 20, 25), gap=5)
c.put('stree_willow', 22, 6)
c.save()
