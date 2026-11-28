"""미리보기 루트: ver5의 assets를 링크하고 graphics만 복사해 지역 건물을 extra 시트에 붙인다.
  python3 build_root.py REGDIR ROOT
"""
import json, os, shutil, sys
from PIL import Image
VER5 = '/Users/younwookim/orca/mhdemo1/ver5'
reg, root = sys.argv[1], sys.argv[2]
lines_dir = sys.argv[3] if len(sys.argv) > 3 else None
if os.path.exists(root):
    shutil.rmtree(root)
os.makedirs(os.path.join(root, 'assets'))
for e in os.listdir(os.path.join(VER5, 'assets')):
    src = os.path.join(VER5, 'assets', e)
    if e in ('music', 'fonts'):
        os.symlink(src, os.path.join(root, 'assets', e))
    elif e != 'graphics' and os.path.isdir(src):
        shutil.copytree(src, os.path.join(root, 'assets', e), symlinks=True)
g = os.path.join(root, 'assets', 'graphics')
shutil.copytree(os.path.join(VER5, 'assets', 'graphics'), g, symlinks=True)
idx = json.load(open(os.path.join(g, 'index.json')))
sheet_path = os.path.join(g, idx['extra']['image'])
sheet = Image.open(sheet_path).convert('RGBA')
items = []
for region in sorted(os.listdir(reg)):
    d = os.path.join(reg, region)
    if not os.path.isdir(d):
        continue
    meta = json.load(open(os.path.join(d, 'meta.json')))
    for n, m in meta.items():
        items.append((f'{region}_{n}', Image.open(os.path.join(d, n + '.png')), m))
if lines_dir:
    for region in sorted(os.listdir(lines_dir)):
        lp = os.path.join(lines_dir, region, 'lines.json')
        if os.path.exists(lp):
            for n, m in json.load(open(lp)).items():
                items.append((n, Image.open(os.path.join(lines_dir, region, n + '.png')), dict(m, line=True)))
# 시트 아래에 줄지어 붙인다
W = sheet.width
x, y, rowh, placed = 0, sheet.height + 2, 0, []
for name, im, m in items:
    if x + im.width > W:
        x, y, rowh = 0, y + rowh + 2, 0
    placed.append((name, im, m, x, y))
    x += im.width + 2
    rowh = max(rowh, im.height)
out = Image.new('RGBA', (W, y + rowh + 2))
out.alpha_composite(sheet, (0, 0))
scene = json.load(open(os.path.join(g, 'scene.json')))
for name, im, m, x, y in placed:
    out.alpha_composite(im, (x, y))
    e = {'rect': [x, y, im.width, im.height], 'pivot': list(m['pivot']), 'layer': 'object'}
    if '_field_' in name:
        e['layer'] = 'ground'
        e['shadow'] = False
    is_piece = m.get('line') and '_gate_' not in name and name.rsplit('_', 1)[-1].isdigit()
    if is_piece:
        e['shadow'] = False
        fam = name.rsplit('_', 1)[0]
        scene['walls'][fam] = {'piece': fam}
    idx['extra']['sprites'][name] = e
    fu, fv = m['footprint']
    if '_field_' not in name and not is_piece:
        scene['footprints'][name] = [-fu / 2, -fv / 2, fu, fv]
out.save(sheet_path)
json.dump(idx, open(os.path.join(g, 'index.json'), 'w'), ensure_ascii=False)
json.dump(scene, open(os.path.join(g, 'scene.json'), 'w'), ensure_ascii=False, indent=1)
print(len(placed), 'sprites; sheet', out.size)
