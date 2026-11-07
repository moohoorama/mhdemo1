"""Layout of map14 (우이 전투). Screen north is cell (0,0): +x runs screen south-east, +y screen south-west.
The 회수 is an arc round the north corner: in from the north-west edge (x=0), round the middle, out at the
north-east edge (y=0). 유비 starts in the north corner inside the arc; the bridge crosses the arc's north-east
leg toward the south-east; beyond it 원술's small castle. 여포's fenced camp is in the south corner, the town
유비 escapes to in the west corner."""
import sys
W, H = 24, 24
BR = 7                      # the bridge's row (y)
g = [['g--'] * W for _ in range(H)]
def f(x, y): return ((x + .5) ** 3 + (y + .5) ** 3) ** (1 / 3)
for y in range(H):
    for x in range(W):
        if 12.3 <= f(x, y) <= 14.4:
            g[y][x] = 'w--'
# the bridge, and the road from the start through it to the castle and on south round the camp to the town
for x in range(W):
    if g[BR][x] == 'w--':
        g[BR][x] = 'wb-'
def road(cells):
    for x, y in cells:
        if g[y][x] == 'g--':
            g[y][x] = 'l--'
road([(x, BR) for x in range(4, 12)] + [(4, y) for y in range(2, BR)])
road([(x, BR) for x in range(14, 19)] + [(18, y) for y in range(BR, 13)] + [(x, 12) for x in range(16, 19)])
road([(16, y) for y in range(12, 17)] + [(x, 17) for x in range(3, 17)] + [(3, y) for y in range(17, 21)])
g[BR][19] = 'go-'           # 원술's small castle beyond the bridge
g[21][2] = 'go-'            # the town in the west corner
g[20][4] = 'gv-'
# 여포's camp in the south corner: a fence ring with a gap toward the road, tents inside
for x in range(17, 24):
    g[18][x] = 'gz-'
for y in range(18, 24):
    g[y][17] = 'gz-'
g[18][19] = g[18][20] = 'g--'   # gap north-west side
g[20][17] = g[21][17] = 'g--'   # gap west side
for x, y in ((20, 20), (22, 20), (21, 22), (19, 22)):
    g[y][x] = 'gk-'
# trees, hills and wasteland for cover along the way
for x, y in ((1, 6), (2, 9), (7, 2), (9, 4), (6, 10), (8, 1), (15, 2), (21, 3), (22, 9), (20, 13), (13, 15),
             (9, 15), (6, 14), (1, 15), (10, 20), (7, 22), (13, 21), (15, 19), (12, 18), (5, 19), (22, 15)):
    if g[y][x] == 'g--':
        g[y][x] = 'gt-'
for x, y in ((9, 19), (10, 19), (9, 20), (14, 13), (15, 13), (6, 15), (7, 15), (1, 3), (2, 3)):
    if g[y][x] in ('g--',):
        g[y][x] = 'u--'
for x, y in ((20, 5), (21, 6), (20, 6), (12, 22), (13, 23), (2, 13), (3, 14)):
    if g[y][x] == 'g--':
        g[y][x] = 'd--'
# shallows on the banks next to the bridge
out = open(sys.argv[1], 'w') if len(sys.argv) > 1 else sys.stdout
out.write('# map14 우이 전투 (made by map14gen.py). Screen north = (0,0).\n')
for r in g:
    out.write(' '.join(r) + '\n')
