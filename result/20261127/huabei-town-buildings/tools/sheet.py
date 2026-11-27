import json, sys, os
from PIL import Image
d, out = sys.argv[1], sys.argv[2]
meta = json.load(open(os.path.join(d, 'meta.json')))
ims = [(Image.open(os.path.join(d, n + '.png')), meta[n]) for n in meta]
W = sum(i.width for i, _ in ims) + 16 * (len(ims) + 1); H = max(i.height for i, _ in ims) + 24
o = Image.new('RGBA', (W, H), (182, 150, 104, 255)); x = 16
for i, m in ims:
    o.alpha_composite(i, (x, H - 8 - i.height)); x += i.width + 16
o.resize((W * 3, H * 3), Image.NEAREST).save(out)
