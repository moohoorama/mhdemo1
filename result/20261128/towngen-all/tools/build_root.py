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
        items.append((f'{region}_{n}', Image.open(os.path.join(d, n + '.png')), dict(m, src=os.path.join(d, n))))
def mud(im, rise=15):
    """물에 잠겼던 집: 기둥마다 발밑에서 rise px까지 흙빛으로, 맨 위에 짙은 물 자국 선 (gen_pixel_wet의 _mud와 같은 생각)"""
    import numpy as np
    a = np.array(im.convert('RGBA')).astype(float)
    h, w = a.shape[:2]
    for x in range(w):
        ys = np.nonzero(a[:, x, 3] > 0)[0]
        if not len(ys):
            continue
        foot = ys.max()
        top = max(0, foot - rise)
        seg = a[top:foot + 1, x, :3]
        ink = seg.sum(1) < 90
        tint = seg * np.array([.62, .55, .46]) + np.array([18, 12, 4])
        seg[~ink] = tint[~ink]
        a[top:foot + 1, x, :3] = seg
        if a[top, x, 3] > 0 and a[top, x, :3].sum() >= 90:
            a[top, x, :3] = a[top, x, :3] * .45
    return Image.fromarray(np.clip(a, 0, 255).astype('uint8'))


for name, im, m in list(items):                     # 회남 건물은 흙탕물 자국 진 판도 (하비 성안)
    if name.startswith('huainan_') and '_field_' not in name:
        items.append((name + '_mud', mud(im), dict(m)))
if lines_dir:
    for region in sorted(os.listdir(lines_dir)):
        lp = os.path.join(lines_dir, region, 'lines.json')
        if os.path.exists(lp):
            for n, m in json.load(open(lp)).items():
                items.append((n, Image.open(os.path.join(lines_dir, region, n + '.png')), dict(m, line=True, src=os.path.join(lines_dir, region, n))))
# 전투맵 언덕·산(지형 시트)을 장면 맵에서 쓰도록 extra로: sc_knoll_00..15(바닥), sc_mtn_*(물건)
ter = json.load(open(os.path.join(g, idx['terrain'])))
ter_img = Image.open(os.path.join(g, ter['image'])).convert('RGBA')
for src_name, dst in [(f'bt_knoll_{k:02d}', f'sc_knoll_{k:02d}') for k in range(16)] + \
        [('wm_mtn_big_0', 'sc_mtn_big_0'), ('wm_mtn_big_1', 'sc_mtn_big_1'), ('wm_mtn_rock_0', 'sc_mtn_rock_0'),
         ('wm_mtn_rock_1', 'sc_mtn_rock_1'), ('wm_mtn_rock_2', 'sc_mtn_rock_2')]:
    e = ter['sprites'][src_name]
    x, y, w, h = e['rect']
    items.append((dst, ter_img.crop((x, y, x + w, y + h)), {'pivot': e['pivot'], 'footprint': [1, 1], 'src': '', 'terrain': e['layer']}))
for name, im, m in list(items):
    if m.get('src') and os.path.exists(m['src'] + '_shadow.png'):
        items.append((name + '#mask', Image.open(m['src'] + '_shadow.png'), {'mask_of': name}))
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
masks = {m['mask_of']: (x, y, im) for name, im, m, x, y in placed if 'mask_of' in m}
for name, im, m, x, y in placed:
    out.alpha_composite(im, (x, y))
    if 'mask_of' in m:
        continue
    e = {'rect': [x, y, im.width, im.height], 'pivot': list(m['pivot']), 'layer': 'object'}
    if '_field_' in name or m.get('terrain') == 'ground':
        e['layer'] = 'ground'
        e['shadow'] = False
    is_piece = m.get('line') and '_gate_' not in name and name.rsplit('_', 1)[-1].isdigit()
    if is_piece:
        e['shadow'] = False
        fam = name.rsplit('_', 1)[0]
        scene['walls'][fam] = {'piece': fam}
    if name in masks:
        mx, my, mim = masks[name]
        e['shadow_mask'] = {'rect': [mx, my, mim.width, mim.height], 'pivot': list(m['shadow_pivot'])}
        e.pop('shadow', None)
    idx['extra']['sprites'][name] = e
    fu, fv = m['footprint']
    if '_field_' not in name and not is_piece and not m.get('terrain'):
        scene['footprints'][name] = [-fu / 2, -fv / 2, fu, fv]
out.save(sheet_path)
json.dump(idx, open(os.path.join(g, 'index.json'), 'w'), ensure_ascii=False)
json.dump(scene, open(os.path.join(g, 'scene.json'), 'w'), ensure_ascii=False, indent=1)
print(len(placed), 'sprites; sheet', out.size)
