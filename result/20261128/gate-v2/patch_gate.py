"""미리보기 루트의 성문 조각(sgate2_*)을 gate2/ 의 새 그림으로 바꾼다: extra 시트 아래에 붙이고 index.json 항목을 덮어씀"""
import json, os, sys
from PIL import Image
root, src = sys.argv[1], sys.argv[2]
g = os.path.join(root, 'assets/graphics')
idx = json.load(open(os.path.join(g, 'index.json')))
sheet_path = os.path.join(g, idx['extra']['image'])
sheet = Image.open(sheet_path).convert('RGBA')
meta = json.load(open(os.path.join(src, 'meta.json')))
ims = [(n, Image.open(os.path.join(src, n + '.png'))) for n in meta]
W = sheet.width; x, y, rowh = 0, sheet.height + 2, 0; placed = []
for n, im in ims:
    if x + im.width > W:
        x, y, rowh = 0, y + rowh + 2, 0
    placed.append((n, im, x, y)); x += im.width + 2; rowh = max(rowh, im.height)
out = Image.new('RGBA', (W, y + rowh + 2)); out.alpha_composite(sheet, (0, 0))
for n, im, x, y in placed:
    out.alpha_composite(im, (x, y))
    e = dict(idx['extra']['sprites'].get(n) or idx['props']['sprites'].get(n) or {})
    e.update({'rect': [x, y, im.width, im.height], 'pivot': meta[n]['pivot'], 'layer': 'object', 'gate_offset': meta[n]['gate_offset']})
    idx['extra']['sprites'][n] = e
    idx['props']['sprites'].pop(n, None)
out.save(sheet_path)
json.dump(idx, open(os.path.join(g, 'index.json'), 'w'), ensure_ascii=False)
print('patched', len(placed))
