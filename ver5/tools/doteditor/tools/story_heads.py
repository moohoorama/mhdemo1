"""2D heads of the later story cast (제갈량·사마의·육손·주유 …, 삼국지 14 looks, 2026-10-30), shared by their scene
sheets (civ3d.py, which renders NW/SW only) and their battle sheets (mounted_kits/story_cast.py, every view).

Keys are those of tools/unit3d/heads.py; the caller maps them to tags (civ3d: HEADMAP / KEY_TAGS; battle: KEY2TAG).
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[4] / 'tools/unit3d'))
import heads as Hd  # noqa: E402


def bearded(grids, kind, key):
    """Hd.beard with the beard in a hair key (h light, g mid, f dark), so the sheet's hair tags colour it: in the
    iron keys a palette that gilds iron (노식, 하후돈) would gild the beard too."""
    out = {}
    for d, rows in grids.items():
        new = Hd.beard(rows, 'ij', kind)
        out[d] = [''.join(key if c == '1' and (y >= len(rows) or rows[y][x] != '1') else c for x, c in enumerate(r))
                  for y, r in enumerate(new)]
    return out


# a cap as wide as the head over the hair (no topknot showing), its top rising toward the back; keys x/y cap
# dark/light, B/H band dark/light
CAP_TOPS = {'SW': ['.......0000..', '.....0yyxx0..', '...00yxxxxx0.', '..0yxxxxxxx0.', '..0HHHBBBBB0.'],
            'NW': ['.......0000..', '.....0xyyx0..', '...00xxyyxx0.', '..0xxxxyyxx0.', '..0BBBBHHBB0.'],
            'S': ['....00000....', '...0yyxxx0...', '..0yyxxxxx0..', '.0yxxxxxxxx0.', '.0HHHBBBBBB0.'],
            'W': ['.......0000..', '.....00yxxx0.', '....0yxxxxxx0', '...0yxxxxxxx0', '..0HHHBBBBBB0'],
            'N': ['....00000....', '...0xxyyx0...', '..0xxxyyxx0..', '.0xxxxyyxxx0.', '.0BBBBHHBBB0.']}


def cap(body, band):
    """Hd.HAIR under a head-sized cap: body (dark, light) and band (dark, light) are the render keys."""
    keys = dict(zip('xyBH', body + band))
    return {d: Hd.recolor(CAP_TOPS[d], keys) + Hd.HAIR[d][5:] for d in CAP_TOPS}


def _plume_down_back(rows):
    """The N view: the plume (b c d) falls from the crown down the middle of the dome's back."""
    g = [list(r) for r in rows]
    for y in range(2, 8):
        for x in (5, 6, 7):
            if g[y][x] not in '.0':
                g[y][x] = 'c' if x == 6 else 'b' if (x + y) % 2 else 'd'
    g[0][5:8], g[1][4:9] = list('000'), list('0dcd0')
    return [''.join(r) for r in g]


# the iron helmet's plume rooted on top and falling down the back of the dome (SW, W: down its right edge, two
# columns wider; NW, N: down over the back; S: the tuft on top); keys b, c, d dark to light
PLUMED = {'SW': ['........00.....', '.......0dd0....', '....000bbccd0..', '...034454332cd0', '..0345544321cd0',
                 '.02333333221bc0', '.0iiiiiii021b0.'] + [r + '..' for r in Hd.IRON['SW'][7:]],
          'NW': ['......00.....', '.....0dd000..', '....000bbcdd0', '...03445bcdd0', '..0344543bcd0',
                 '.02333333bc10'] + Hd.IRON['NW'][6:],
          'S': ['.....000.....', '....0dcd0....'] + Hd.IRON['S'][2:],
          'W': ['........00.....', '.......0dd0....', '....000bbccd0..', '...034454332cd0', '..0345544321cd0',
                '023333333221bc0', '.0iiiiii0221b0.'] + [r + '..' for r in Hd.IRON['W'][7:]],
          'N': _plume_down_back(Hd.IRON['N'])}


def helmet(dome, plume):
    """The plumed iron helmet with its dome (front: the rows over the brow band; back: down to the neck guard)
    in the keys dome (dark, mid, light), the brow band, cheek and neck guards left in iron, the plume in the keys
    plume."""
    feather = dict(zip('bcd', plume))
    cut = dict(zip('12345', (dome[0], dome[0], dome[1], dome[2], dome[2])), **feather)
    return {d: [Hd.recolor([r], cut if y < (8 if d in ('NW', 'N') else 5) else feather)[0] for y, r in enumerate(rows)]
            for d, rows in PLUMED.items()}


DARK_TOPKNOT = {d: Hd.recolor(rows, {'k': 'h', 'l': 'g', 'm': 'f'}) for d, rows in Hd.TOPKNOT.items()}  # band in hair


# hair let down, no topknot (the 황건 brothers, 2026-11-28): a ragged mass over the crown with stray hairs above it,
# the cloth band (k l m) across the brow and the hair falling behind the ears; keys f dark, g mid
LOOSE_TOPS = {'SW': ['.............', '....0..0.0...', '...0f00f0f0..', '..0fgffgffg0.'],
              'NW': ['.............', '....0.0..0...', '...0f0f00f0..', '..0fgfgffgf0.'],
              'S': ['.............', '...0..0..0...', '..0f00f00f0..', '.0fgffgffgf0.'],
              'W': ['.............', '......0.0.0..', '.....0f0f0f0.', '...0fgffgfgf0'],
              'N': ['.............', '...0..0..0...', '..0f00f00f0..', '.0fgffgffgf0.']}


def loose():
    """Hd.TOPKNOT with the bun let down, its three-row band narrowed to one row of cloth across the brow (the hair
    showing above and below it) with its tails still flying behind."""
    out = {}
    for d in LOOSE_TOPS:
        rows = Hd.TOPKNOT[d]
        above = Hd.recolor([rows[4]], {'m': 'g', 'l': 'g', 'k': 'f'})[0]
        below = Hd.recolor([rows[6][:11]], {'m': 'f', 'l': 'f', 'k': 'f'})[0] + rows[6][11:]
        out[d] = LOOSE_TOPS[d] + [above, rows[5], below] + rows[7:]
    return out

