#!/usr/bin/env python3
"""Court robes by rank: copies of the official's scene sheet whose robe (the team key, faction tags) is moved to
theme B and given the rank's colour, so a court reads by colour how near each man stands to the throne.

  python3 robe_colors.py          # units/official_<rank>_noncombat.yaml for every rank in RANKS

Then build the sheets (spritebuild/build.py) as usual. The faction-coloured official_noncombat stays for a lord's
own officials.
"""
import dots

# highest first: 삼공(진홍), 구경(자주), 대부(청), 낭관(녹). Most of the robe is the darkest of the three tags, so the
# third shade is the colour the robe reads as.
RANKS = {
    'crimson': ('삼공', ['#e07078', '#c4485a', '#982e3c']),
    'purple': ('구경', ['#b08cd0', '#8c64aa', '#6a4486']),
    'blue': ('대부', ['#84acdc', '#5e8cc8', '#3c6aa2']),
    'green': ('낭관', ['#98c87c', '#74aa5e', '#4e8440']),
}


def main():
    for rank, (title, colors) in RANKS.items():
        doc = dots.with_alpha(dots.load('official_noncombat'))
        ids = dots.tag_ids(doc)
        move = {ids.index(f'faction{i}') + 1: ids.index(f'themeB{i}') + 1 for i in (1, 2, 3)}
        for f in doc['frames']:
            f['tags'] = [[move.get(t, t) for t in row] for row in f['tags']]
        for i, c in enumerate(colors, 1):
            doc['tag_colors'][ids.index(f'themeB{i}')] = c + 'ff'
        doc['unit'].update(id=f'official_{rank}_noncombat', name=f'문관({title})')
        print(dots.save(doc))


if __name__ == '__main__':
    main()
