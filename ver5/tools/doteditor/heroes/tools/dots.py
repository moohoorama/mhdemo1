"""Shared reading and writing of dot editor unit YAML (mhdemo-unit-pixels v3) for the hero tools."""
import re
import sys
from pathlib import Path

import yaml

DOTEDITOR = Path(__file__).resolve().parents[2]
UNITS = DOTEDITOR / 'units'
HEROES = DOTEDITOR / 'heroes'

GROUPS = [('iron', '철'), ('skin', '피부색'), ('hair', '털색'), ('faction', '진영색'), ('themeA', '테마색 A'),
          ('themeB', '테마색 B'), ('themeC', '테마색 C'), ('horseSkin', '말 피부색'), ('horseMane', '말 갈기색'),
          ('saddle', '말 안장색'), ('outline', '외곽선'), ('effect', '공격 효과')]
# One grid character per opaque tag, in tag index order (1..34). Translucent pixels (effects) use TRANSLUCENT_CHARS.
TAG_CHARS = 'abcdefghijklmnopqrstuvwxyzABCD#EFG'
TRANSLUCENT_CHARS = '123456789*+=%&'


def tag_ids(doc):
    return [t['id'] for t in doc['tag_definitions']]


def load(unit_id):
    path = UNITS / f'{unit_id}.yaml'
    return yaml.load(path.read_text(), Loader=getattr(yaml, 'CSafeLoader', yaml.SafeLoader))


def save(doc):
    """Rebuilds the palette from tags (tag RGB + pixel alpha) and writes units/<id>.yaml."""
    rgb = [c[1:7] for c in doc['tag_colors']]
    palette, lookup = ['#00000000'], {'#00000000': 0}
    for f in doc['frames']:
        f['pixels'] = [[_index(palette, lookup, f'#{rgb[t - 1]}{a}' if t else '#00000000')
                        for t, a in zip(trow, arow)] for trow, arow in zip(f['tags'], f.pop('alpha'))]
    doc['palette'] = palette
    path = UNITS / f"{doc['unit']['id']}.yaml"
    text = yaml.dump(doc, Dumper=getattr(yaml, 'CSafeDumper', yaml.SafeDumper), allow_unicode=True,
                     sort_keys=False, default_flow_style=None, width=100000)
    path.write_text(text)
    return path


def _index(palette, lookup, color):
    if color not in lookup:
        lookup[color] = len(palette)
        palette.append(color)
    return lookup[color]


def with_alpha(doc):
    """Adds f['alpha'] (hex alpha per pixel) so frames can be edited by tag and written back with save()."""
    for f in doc['frames']:
        f['alpha'] = [[doc['palette'][p][7:9] for p in row] for row in f['pixels']]
    return doc


def frame(doc, anim, direction, n):
    for f in doc['frames']:
        if f['animation'] == anim and f['direction'] == direction and f['frame'] == n:
            return f
    raise SystemExit(f'no frame {anim}/{direction}/{n} in {doc["unit"]["id"]}')


def rgba(doc, f, tag_colors=None):
    """Resolved RGBA tuples of a frame, row by row (tag color RGB + the pixel's own alpha)."""
    colors = tag_colors or doc['tag_colors']
    alpha = f.get('alpha') or [[doc['palette'][p][7:9] for p in row] for row in f['pixels']]
    out = []
    for trow, arow in zip(f['tags'], alpha):
        row = []
        for t, a in zip(trow, arow):
            c = colors[t - 1] if t else '#000000'
            row.append((int(c[1:3], 16), int(c[3:5], 16), int(c[5:7], 16), int(a, 16) if t else 0))
        out.append(row)
    return out


def grid_files(unit_id):
    return sorted((HEROES / unit_id / 'grid').glob('*/*.txt'))


def apply_grids(doc):
    """Overlays heroes/<id>/grid/*/*.txt (the working copy) onto a with_alpha() doc; returns the file count."""
    files = grid_files(doc['unit']['id'])
    for path in files:
        apply_grid(doc, path)
    return len(files)


def apply_grid(doc, path):
    """Overlays one grid file; raises SystemExit (frame untouched) when the file is malformed."""
    w, h = doc['unit']['cell']
    ids = tag_ids(doc)
    anim, (d, n) = path.parent.name, path.stem.split('-')
    f = frame(doc, anim, d, int(n))
    text = path.read_text().splitlines()
    table = {'.': (0, '00')}
    for line in text:
        m = re.match(r'; (\S) = (\S+) .*?(?:alpha ([0-9a-f]{2}))?$', line)
        if m and m.group(1) != '.':
            table[m.group(1)] = (ids.index(m.group(2)) + 1, m.group(3) or 'ff')
    rows = [line.split('|', 1)[1] for line in text if not line.startswith(';') and '|' in line]
    if len(rows) != h or any(len(r) != w for r in rows):
        raise SystemExit(f'{path}: expected {h} rows of {w} chars, got {len(rows)} rows '
                         f'(widths {sorted({len(r) for r in rows})})')
    bad = {c for r in rows for c in r} - set(table)
    if bad:
        raise SystemExit(f'{path}: unknown characters {sorted(bad)}')
    f['tags'] = [[table[c][0] for c in r] for r in rows]
    f['alpha'] = [[table[c][1] for c in r] for r in rows]


def load_working(unit_id):
    """units/<id>.yaml with the grid working copy applied, without writing anything. A grid file that is
    malformed (another artist mid-edit) is skipped with a warning; import stays strict."""
    doc = with_alpha(load(unit_id))
    for path in grid_files(unit_id):
        try:
            apply_grid(doc, path)
        except SystemExit as e:
            print(f'skipped {path.relative_to(HEROES)}: {e}', file=sys.stderr)
    return doc
