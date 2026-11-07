#!/usr/bin/env python3
"""Bow (읍) and prostrate (엎드림) poses for officers whose scene sheet has none, borrowed from the official's scene
sheet (official_noncombat, which civ3d.py posed from the same body): the official's robe is recoloured into the
officer's own tags, and the officer's own head is put where the official's head went.

  python3 bow_poses.py liubei guanyu zhangfei jianyong      # adds bow and prostrate to units/<id>_noncombat.yaml

How it matches the two sheets:
- Robe: both idle frames stand on the same body, so a pixel of the official's robe and the same pixel of the
  officer's idle frame are the same piece of cloth. Each official tag takes the officer tag it lies under most often
  (both views, the torso rows counted four times: a bowed body shows its back), and every pixel of the borrowed
  pose is recoloured by that table.
- Head: the officer's head is every pixel of their idle frame down to the bottom of the face (the faces of the two
  sheets sit on the same rows), plus any hair below it (a beard). The official's head is found again in the borrowed
  pose by sliding it until its tags line up; the official's head pixels there are cleared and the officer's head is
  laid on at the same shift.

Then build the sheet (spritebuild/build.py) as usual; the poses play as Pose "bow" / "prostrate" in a scene.
"""
import sys
from collections import Counter, defaultdict

import dots

BASE = 'official_noncombat'
POSES = ('bow', 'prostrate')
FACE_BOTTOM = 21  # last row of the face in the idle frames (rows of the 56×46 cell)
TORSO = 9  # rows below the face that are the coat over the shoulders and back


def tag_of(doc, name):
    return dots.tag_ids(doc).index(name) + 1


def head_mask(doc, frame):
    """The head of an idle frame: everything down to the bottom of the face, and hair (a beard) below it."""
    hair = {tag_of(doc, f'hair{i}') for i in (1, 2, 3)}
    outline = tag_of(doc, 'outline')
    mask = set()
    for y, row in enumerate(frame['tags']):
        for x, t in enumerate(row):
            if t and (y <= FACE_BOTTOM or t in hair):
                mask.add((x, y))
    # the beard's outline below the face goes with it
    for y, row in enumerate(frame['tags']):
        for x, t in enumerate(row):
            if t == outline and y > FACE_BOTTOM and any((x + dx, y + dy) in mask and frame['tags'][y + dy][x + dx] in hair
                                                         for dx in (-1, 0, 1) for dy in (-1, 0, 1)):
                mask.add((x, y))
    return mask


def find_shift(base_idle, mask, pose):
    """The (dx, dy) that lays the official's idle head over the same head in the borrowed pose."""
    h, w = len(pose['tags']), len(pose['tags'][0])
    best, shift = -1, (0, 0)
    for dy in range(-4, 20):
        for dx in range(-16, 8):
            score = 0
            for x, y in mask:
                px, py = x + dx, y + dy
                if 0 <= px < w and 0 <= py < h and pose['tags'][py][px] == base_idle['tags'][y][x]:
                    score += 1
            if score > best:
                best, shift = score, (dx, dy)
    return shift, best / max(1, len(mask))


def robe_table(base, officer, base_mask):
    """Official tag -> officer tag, voted pixel by pixel over the idle frames outside the head."""
    votes = defaultdict(Counter)
    for d in ('SW', 'NW'):
        b, o = dots.frame(base, 'idle', d, 0), dots.frame(officer, 'idle', d, 0)
        for y, row in enumerate(b['tags']):
            for x, t in enumerate(row):
                if t and (x, y) not in base_mask[d] and o['tags'][y][x]:
                    # a bowed body shows its back and shoulders: the torso rows count most
                    votes[t][o['tags'][y][x]] += 4 if y <= FACE_BOTTOM + TORSO else 1
    outline = tag_of(base, 'outline')
    table = {t: c.most_common(1)[0][0] for t, c in votes.items()}
    table[outline] = outline
    return table


def pose_frames(base, officer):
    base_masks = {d: head_mask(base, dots.frame(base, 'idle', d, 0)) for d in ('SW', 'NW')}
    table = robe_table(base, officer, base_masks)
    out = []
    for anim in POSES:
        for d in ('NW', 'SW'):
            bf, pose = dots.frame(base, 'idle', d, 0), dots.frame(base, anim, d, 0)
            (dx, dy), fit = find_shift(bf, base_masks[d], pose)
            tags = [[table.get(t, t) if t else 0 for t in row] for row in pose['tags']]
            alpha = [[a for a in row] for row in pose['alpha']]
            h, w = len(tags), len(tags[0])
            for x, y in base_masks[d]:
                if 0 <= x + dx < w and 0 <= y + dy < h:
                    tags[y + dy][x + dx], alpha[y + dy][x + dx] = 0, '00'
            of = dots.frame(officer, 'idle', d, 0)
            for x, y in head_mask(officer, of):
                if 0 <= x + dx < w and 0 <= y + dy < h:
                    tags[y + dy][x + dx], alpha[y + dy][x + dx] = of['tags'][y][x], of['alpha'][y][x]
            out.append({'direction': d, 'animation': anim, 'frame': 0, 'tags': tags, 'alpha': alpha})
            print(f'  {anim} {d}: head shift ({dx}, {dy}), fit {fit:.0%}')
    return out


def main(ids):
    base = dots.with_alpha(dots.load(BASE))
    for cid in ids:
        uid = cid if cid.endswith('_noncombat') else f'{cid}_noncombat'
        officer = dots.with_alpha(dots.load(uid))
        print(uid)
        new = pose_frames(base, officer)
        officer['frames'] = [f for f in officer['frames'] if f['animation'] not in POSES] + new
        anims = officer['unit']['animations']
        col = max(a['first_column'] + a['frames'] for k, a in anims.items() if k not in POSES)
        for anim in POSES:
            anims[anim] = dict(base['unit']['animations'][anim], first_column=col)
            col += 1
        print(' ', dots.save(officer))


if __name__ == '__main__':
    main(sys.argv[1:] or ['liubei', 'guanyu', 'zhangfei', 'jianyong'])
