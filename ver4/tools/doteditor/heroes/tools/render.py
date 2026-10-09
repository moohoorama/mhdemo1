#!/usr/bin/env python3
"""Pictures of hero dots for looking at them: nothing here judges the drawing.

  render.py frame liubei attack SW 2 [--zoom 10]     # one frame, pivot marked, beside the base frame
  render.py sheet liubei [--anims idle,attack] [--dirs SW,NW] [--zoom 4]
                                                     # rows = direction, columns = frames, one block per animation
  render.py recolor liubei [--groups themeA,faction] [--anims idle,attack] [--zoom 4]
                                                     # per tag group: original block beside a hue-shifted block
Frames come from units/<id>.yaml with the heroes/<id>/grid working copy laid over it (nothing is written
back). Output goes to heroes/<id>/preview/ and the path is printed.
"""
import argparse
import colorsys

from PIL import Image, ImageDraw

import dots

BG = (60, 60, 60, 255)
LABEL = (200, 205, 210, 255)
NAMES = {'idle': '대기', 'walk': '걷기', 'attack': '공격', 'hit': '피격', 'exhausted': '탈진', 'block': '막기',
         'action': '액션'}


def image(doc, f, tag_colors=None):
    w, h = doc['unit']['cell']
    im = Image.new('RGBA', (w, h))
    im.putdata([px for row in dots.rgba(doc, f, tag_colors) for px in row])
    return im


def on_bg(im, zoom, pivot=None):
    big = im.resize((im.width * zoom, im.height * zoom), Image.NEAREST)
    out = Image.new('RGBA', big.size, BG)
    out.alpha_composite(big)
    if pivot:
        d = ImageDraw.Draw(out)
        x, y = pivot[0] * zoom, pivot[1] * zoom
        d.rectangle([x, y, x + zoom - 1, y + zoom - 1], outline=(255, 60, 200, 255))
    return out


def grid_lines(im, zoom):
    d = ImageDraw.Draw(im)
    for x in range(0, im.width, zoom):
        d.line([x, 0, x, im.height], fill=(0, 0, 0, 40))
    for y in range(0, im.height, zoom):
        d.line([0, y, im.width, y], fill=(0, 0, 0, 40))
    return im


def block(doc, anims, dirs, zoom, tag_colors=None):
    """Rows = direction, columns = frames of each animation in turn."""
    u = doc['unit']
    w, h = u['cell'][0] * zoom, u['cell'][1] * zoom
    cols = [(a, n) for a in anims for n in range(u['animations'][a]['frames'])]
    pad, top, left = 6, 18, 34
    sheet = Image.new('RGBA', (left + len(cols) * (w + pad), top + len(dirs) * (h + pad)), (30, 32, 34, 255))
    d = ImageDraw.Draw(sheet)
    for c, (a, n) in enumerate(cols):
        d.text((left + c * (w + pad), 3), f'{a} {n}', fill=LABEL)
    for r, dr in enumerate(dirs):
        d.text((3, top + r * (h + pad) + h // 2), dr, fill=LABEL)
        for c, (a, n) in enumerate(cols):
            sheet.paste(on_bg(image(doc, dots.frame(doc, a, dr, n), tag_colors), zoom, u['pivot']),
                        (left + c * (w + pad), top + r * (h + pad)))
    return sheet


def shifted(doc, group, hue):
    """tag_colors with one group's hue rotated (and lightness kept), to see which pixels follow the group."""
    colors = list(doc['tag_colors'])
    for i, tid in enumerate(dots.tag_ids(doc)):
        if tid.rstrip('123') == group:
            c = colors[i]
            r, g, b = (int(c[k:k + 2], 16) / 255 for k in (1, 3, 5))
            hh, l, s = colorsys.rgb_to_hls(r, g, b)
            if s < 0.15 or group == 'outline':  # grey/black ramps: push toward a strong colour instead
                hh, s, l = 0.85, 0.75, min(max(l, 0.25), 0.7)
            r, g, b = colorsys.hls_to_rgb((hh + hue / 360) % 1, l, max(s, 0.5))
            colors[i] = '#%02x%02x%02xff' % tuple(round(v * 255) for v in (r, g, b))
    return colors


def save(im, doc, name):
    out = dots.HEROES / doc['unit']['id'] / 'preview' / name
    out.parent.mkdir(parents=True, exist_ok=True)
    im.save(out)
    print(out)


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest='cmd', required=True)
    f = sub.add_parser('frame')
    f.add_argument('id'), f.add_argument('anim'), f.add_argument('dir'), f.add_argument('n', type=int)
    f.add_argument('--zoom', type=int, default=10)
    for name in ('sheet', 'recolor'):
        s = sub.add_parser(name)
        s.add_argument('id'), s.add_argument('--anims'), s.add_argument('--dirs'), s.add_argument('--zoom', type=int)
        if name == 'recolor':
            s.add_argument('--groups'), s.add_argument('--hue', type=int, default=150)
    a = p.parse_args()
    doc = dots.load_working(a.id)
    u = doc['unit']
    if a.cmd == 'frame':
        hero = on_bg(image(doc, dots.frame(doc, a.anim, a.dir, a.n)), a.zoom, u['pivot'])
        pics = [grid_lines(hero, a.zoom)]
        base = u.get('source_base')
        if base and not a.id.endswith('_noncombat'):
            bdoc = dots.with_alpha(dots.load(base))
            pics.append(grid_lines(on_bg(image(bdoc, dots.frame(bdoc, a.anim, a.dir, a.n)), a.zoom,
                                         bdoc['unit']['pivot']), a.zoom))
        out = Image.new('RGBA', (sum(p.width for p in pics) + 10 * (len(pics) - 1), pics[0].height), (30, 32, 34, 255))
        x = 0
        for pic in pics:
            out.paste(pic, (x, 0))
            x += pic.width + 10
        save(out, doc, f'frame-{a.anim}-{a.dir}-{a.n}.png')
        return
    anims = a.anims.split(',') if a.anims else list(u['animations'])
    dirs = a.dirs.split(',') if a.dirs else u['rows']
    zoom = a.zoom or (4 if u['cell'][0] < 80 else 3)
    if a.cmd == 'sheet':
        save(block(doc, anims, dirs, zoom), doc, f"sheet-{'_'.join(anims)}-{'_'.join(dirs)}.png")
        return
    groups = a.groups.split(',') if a.groups else [g for g, _ in dots.GROUPS if g != 'effect']
    used = {dots.tag_ids(doc)[t - 1].rstrip('123') for f in doc['frames'] for row in f['tags'] for t in row if t}
    for g in groups:
        if g not in used:
            print('skip', g, '(no pixels)')
            continue
        left, right = block(doc, anims, dirs, zoom), block(doc, anims, dirs, zoom, shifted(doc, g, a.hue))
        out = Image.new('RGBA', (left.width * 2 + 16, left.height), (90, 20, 20, 255))
        out.paste(left, (0, 0))
        out.paste(right, (left.width + 16, 0))
        save(out, doc, f"recolor-{g}-{'_'.join(anims)}.png")


if __name__ == '__main__':
    main()
