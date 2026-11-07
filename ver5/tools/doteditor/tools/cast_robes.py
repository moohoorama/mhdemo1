#!/usr/bin/env python3
"""Scene sheets of named officers without a sheet of their own: copies of the official's scene sheet whose robe (the
faction tags) is moved to theme B and given the officer's own colours, as robe_colors.py does for the court ranks.
The robe then stays the same whatever army's colours the officer stands in.

  python3 cast_robes.py          # units/<id>_noncombat.yaml for every officer in CAST

Then build the sheets (spritebuild/build.py) as usual and point the character's Look.Scene at <id>_noncombat.
"""
import dots

# id: (name, robe light/mid/dark, hair light/mid/dark or None to keep). Most of the robe is the darkest shade.
CAST = {
    'zhugexuan': ('제갈현', ['#c4d0d0', '#9aaeb0', '#6e8488'], None),                  # 낭야 선비, 바랜 옥색
    'mizhu': ('미축', ['#e8c878', '#c8a050', '#97702c'], None),                      # 서주 거부, 황토 금빛
    'mifang': ('미방', ['#d89870', '#b06c44', '#7e4628'], None),                     # 아우, 적갈
    'caobao': ('조표', ['#8ca07c', '#627a56', '#3e5236'], ['#7a7268', '#55504a', '#36322e']),  # 늙은 무장, 짙은 녹·희끗한 머리
    'chendeng': ('진등', ['#86b4b0', '#5a8c8a', '#3a6462'], None),                   # 하비 명문, 청록
    'xutan': ('허탐', ['#c07a6a', '#94503e', '#6a3226'], None),                      # 단양병 장수, 검붉은
    'fanya': ('번아', ['#ece6d2', '#cfc6a6', '#a49a78'], None),                      # 의원, 흰 베옷
}


def main():
    for cid, (name, robe, hair) in CAST.items():
        doc = dots.with_alpha(dots.load('official_noncombat'))
        ids = dots.tag_ids(doc)
        move = {ids.index(f'faction{i}') + 1: ids.index(f'themeB{i}') + 1 for i in (1, 2, 3)}
        for f in doc['frames']:
            f['tags'] = [[move.get(t, t) for t in row] for row in f['tags']]
        for i, c in enumerate(robe, 1):
            doc['tag_colors'][ids.index(f'themeB{i}')] = c + 'ff'
        for i, c in enumerate(hair or [], 1):
            doc['tag_colors'][ids.index(f'hair{i}')] = c + 'ff'
        doc['unit'].update(id=f'{cid}_noncombat', name=f'{name}(장면)')
        print(dots.save(doc))


if __name__ == '__main__':
    main()
