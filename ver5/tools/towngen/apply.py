"""towngen 결과를 게임에 반영한다 (사용자 승낙 뒤).

1. 지역 건물(work/reg)·줄 타일(work/lines)·회남 흙탕물 판(_mud)을 assets_candidate/tiles에 넣고 tiles.json에 올린다.
2. tools/build_extra.py로 extra 시트를 다시 굽는다.
3. scene.json에 줄 조각 집안(walls)과 건물 바닥(footprints)을 넣는다.
4. 미리보기 루트(work/root)에서 생성한 맵과 towngen이 고친 탐험·장면 파일을 ver5/assets로 옮긴다.

  python3 tools/towngen/apply.py [WORK]
"""
import json
import re
import os
import shutil
import subprocess
import sys

from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
VER5 = os.path.dirname(os.path.dirname(HERE))
CAND = os.path.join(os.path.dirname(VER5), 'assets_candidate')


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


ROOMS = {'S_tool_shop', 'S_tavern', 'S_zhuo_office', 'S_xiaopei_office', 'S_council', 'S_liu_house', 'S_mansion'}
MADE = 'regional town building / wall line, pixel art on the small-tile grid (tools/towngen)'


def items(work):
    """(이름, 그림 경로, 그림자 마스크 경로 또는 None, 메타, 줄 조각인가)"""
    out = []
    reg = os.path.join(work, 'reg')
    for region in sorted(os.listdir(reg)):
        d = os.path.join(reg, region)
        if not os.path.isdir(d):
            continue
        for n, m in json.load(open(os.path.join(d, 'meta.json'))).items():
            mask = os.path.join(d, n + '_shadow.png')
            out.append((f'{region}_{n}', os.path.join(d, n + '.png'), mask if os.path.exists(mask) else None, m, False))
            if region == 'huainan' and not n.startswith('field_'):
                out.append((f'{region}_{n}_mud', os.path.join(d, n + '.png'), mask if os.path.exists(mask) else None, m, 'mud'))
    lines = os.path.join(work, 'lines')
    for region in sorted(os.listdir(lines)):
        lp = os.path.join(lines, region, 'lines.json')
        if not os.path.exists(lp):
            continue
        for n, m in json.load(open(lp)).items():
            mask = os.path.join(lines, region, n + '_shadow.png')
            out.append((n, os.path.join(lines, region, n + '.png'), mask if os.path.exists(mask) else None, m, 'line'))
    props = os.path.join(work, 'props')
    if os.path.exists(os.path.join(props, 'meta.json')):
        for n, m in json.load(open(os.path.join(props, 'meta.json'))).items():
            out.append((n, os.path.join(props, n + '.png'), None, m, 'prop'))
    return out


def tiles(work):
    tj = os.path.join(CAND, 'tiles/tiles.json')
    doc = json.load(open(tj))
    sprites = doc['sprites']
    scene_path = os.path.join(VER5, 'assets/graphics/scene.json')
    scene = json.load(open(scene_path))
    walls, feet = {}, {}
    n_new = 0
    for name, png, mask, m, kind in items(work):
        im = Image.open(png).convert('RGBA')
        if kind == 'mud':
            im = mud(im)
        dst = f'tiles/{name}.png'
        im.save(os.path.join(CAND, dst))
        e = {'file': dst, 'size': list(im.size), 'pivot': list(m['pivot']), 'layer': 'object', 'scale': 'scene', 'made': MADE}
        if kind == 'prop':                      # 문 햇빛·이정표: 바닥 칸을 차지하지 않는다
            e['layer'], e['shadow'] = m['layer'], False
            n_new += name not in sprites
            sprites[name] = e
            continue
        field = '_field_' in name
        piece = kind == 'line' and '_gate_' not in name and name.rsplit('_', 1)[-1].isdigit()
        if field:
            e['layer'] = 'ground'
            e['shadow'] = False
        if mask:
            mdst = f'tiles/{name}_shadow.png'
            shutil.copyfile(mask, os.path.join(CAND, mdst))
            e['shadow_mask'] = {'file': mdst, 'pivot': list(m['shadow_pivot'])}
        elif piece:
            e['shadow'] = False
        n_new += name not in sprites
        sprites[name] = e
        if piece:
            fam = name.rsplit('_', 1)[0]
            if fam not in scene['walls']:
                walls[fam] = {'piece': fam}
        elif not field:
            fu, fv = m['footprint']
            if name not in scene['footprints']:
                feet[name] = [-fu / 2, -fv / 2, fu, fv]
    json.dump(doc, open(tj, 'w'), ensure_ascii=False, indent=1)
    open(tj, 'a').write('\n')
    scene_text(scene_path, walls, feet)
    return n_new


def scene_text(path, walls, feet):
    """scene.json은 손으로 다듬은 모양이라 다시 쓰지 않고 줄을 끼워 넣는다: walls 끝에 줄 조각 집안, footprints 첫머리에 건물"""
    t = open(path).read()
    c = lambda x: json.dumps(x, ensure_ascii=False, separators=(', ', ': '))
    if walls:
        a = t.index('\n "walls": {')
        b = t.index('\n', a + 1)
        line = t[a:b]
        assert line.endswith('}},'), line
        add = ''.join(f',\n  {c(k)}: {c(v)}' for k, v in walls.items())
        t = t[:a] + line[:-2] + add + '},' + t[b:]
    if feet:
        a = t.index('\n "footprints": {') + len('\n "footprints": {')
        t = t[:a] + ''.join(f'\n  {c(k)}: {c(v)},' for k, v in feet.items()) + t[a:]
    json.loads(t)
    open(path, 'w').write(t)


def map_text(m):
    """저장소 맵 파일 모양: 위 단계 키 한 줄씩, 바닥 줄·물건·줄은 한 줄에 하나"""
    c = lambda x: json.dumps(x, ensure_ascii=False, separators=(',', ':'))
    out = []
    for k, v in m.items():
        if isinstance(v, list) and v and isinstance(v[0], (str, list)) and k in ('ground', 'height', 'objects', 'walls'):
            body = ',\n'.join('  ' + c(x) for x in v)
            out.append(f' {c(k)}: [\n{body}\n ]')
        else:
            out.append(f' {c(k)}: {c(v)}')
    return '{\n' + ',\n'.join(out) + '\n}\n'


def smith_scene(root):
    """2장 대장간 장면은 저장소 글 모양을 살려 맵·걸음·방향만 바꾼다"""
    sc = json.load(open(os.path.join(root, 'scenes/ch02.json')))['Scenes']['c02-e-smith']
    k1 = sc['Beats']['c02-k1'][0]
    move, faces = k1[0][0]['Move'], {st['Actor']: st['Face'] for st in k1[1]}
    p = os.path.join(VER5, 'assets/scenes/ch02.json')
    t = open(p).read()
    a = t.index('"c02-e-smith": {')
    b = t.index('"c02-k2": [', a)
    seg = t[a:b]
    seg = re.sub(r'"Map": "[^"]*"', f'"Map": "{sc["Map"]}"', seg, count=1)
    seg = re.sub(r'\{"Actor": "유비", "Move": \[[^\]]*\]\}', f'{{"Actor": "유비", "Move": [{move[0]}, {move[1]}]}}', seg, count=1)
    seg = re.sub(r'\{"Actor": "유비", "Face": "[A-Z]+"\}, \{"Actor": "대장장이", "Face": "[A-Z]+"\}',
                 f'{{"Actor": "유비", "Face": "{faces["유비"]}"}}, {{"Actor": "대장장이", "Face": "{faces["대장장이"]}"}}', seg, count=1)
    open(p, 'w').write(t[:a] + seg + t[b:])
    return 'scenes/ch02.json (c02-e-smith)'


def data(work):
    """맵(roles/에 남은 것)과 미리보기 루트에서 바뀐 탐험 파일, 2장 대장간 장면"""
    root = os.path.join(work, 'root/assets')
    moved = []
    for f in sorted(os.listdir(os.path.join(work, 'root/roles'))):
        m = f[:-5]
        art = json.load(open(os.path.join(root, 'graphics/maps', m + '.json')))
        open(os.path.join(VER5, 'assets/graphics/maps', m + '.json'), 'w').write(map_text(art))
        moved.append(f'maps/{m}.json')
    moved.append(smith_scene(root))
    towns = {f[:-5] for f in os.listdir(os.path.join(work, 'root/roles'))}
    gen = os.path.join(root, 'scenario/hero')
    for f in sorted(os.listdir(gen)):
        if not f.startswith('explore') or not f.endswith('.json'):
            continue
        dst = os.path.join(VER5, 'assets/scenario/hero', f)
        new, cur = json.load(open(os.path.join(gen, f))), json.load(open(dst))
        before = json.dumps(cur, ensure_ascii=False)
        for k in list(cur.get('Areas') or {}):          # towngen이 고치는 것만: 성 맵 지역의 출구·사람·표지, 뺀 실내 대장간
            na = (new.get('Areas') or {}).get(k)
            if na is None:
                if cur['Areas'][k].get('Map') == 'S_weapon_shop':
                    del cur['Areas'][k]
                continue
            if na.get('Map') in towns or na.get('Map') in ROOMS:      # 실내는 출구 자리·방향만 바뀐다
                for field in (('Exits', 'People', 'Signs') if na.get('Map') in towns else ('Exits',)):
                    if field in na:
                        cur['Areas'][k][field] = na[field]
        if json.dumps(cur, ensure_ascii=False) != before:
            json.dump(cur, open(dst, 'w'), ensure_ascii=False, indent=1)
            moved.append(f'scenario/hero/{f}')
    return moved


if __name__ == '__main__':
    work = sys.argv[1] if len(sys.argv) > 1 else os.path.join(HERE, 'work')
    print('tiles: new', tiles(work))
    subprocess.run([sys.executable, os.path.join(VER5, 'tools/build_extra.py')], check=True)
    changed = data(work)
    maps = [os.path.join(VER5, 'assets/graphics', m) for m in changed if m.startswith('maps/')]
    subprocess.run(['go', 'run', './cmd/_mapfmt', *maps], cwd=VER5, check=True)     # scenemap.Encode 모양으로
    for m in changed:
        print(' ', m)
