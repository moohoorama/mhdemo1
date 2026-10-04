"""Hand-drawn 2D heads stamped on the rendered bodies, plus the small grid helpers.

A head is a 13-wide grid of palette keys ('.' transparent, ',' clears). Five views are
drawn (S, SW, W, NW, N); SE, E and NE mirror SW, W and NW. Every face shares one
construction: helmet or hood rim, a one-row shadow under it, 2px-tall eyes.
Hero heads are derived by recoloring and adding a beard. Faction-colored pixels
(plume, turban band, hood) are mapped to the team keys W-Z (see factions.py).
"""
from PIL import Image
import render as R

MIRROR = {'SE': 'SW', 'E': 'W', 'NE': 'NW'}
ANCHOR = (6, 7)  # head grid pixel placed on the projected head centre


def grid(*rows, w=13):
    for r in rows:
        assert len(r) == w, (r, len(r))
    return list(rows)


def overlay(g, part, x0, y0):
    """Paint a grid part onto g; '.' keeps the pixel, ',' clears it."""
    for dy, row in enumerate(part):
        for dx, c in enumerate(row):
            if c != '.' and 0 <= y0 + dy < len(g) and 0 <= x0 + dx < len(g[0]):
                g[y0+dy][x0+dx] = '.' if c == ',' else c


def to_img(g):
    im = Image.new('RGBA', (len(g[0]), len(g)))
    for y, row in enumerate(g):
        for x, c in enumerate(row):
            if c != '.':
                im.putpixel((x, y), R.COLOR[c] + (255,))
    return im


def recolor(rows, mapping):
    table = str.maketrans(mapping)
    return [row.translate(table) for row in rows]


g = grid
IRON = {
    'SW': g('.............', '.......0d0...', '......0dc0...', '....000bb000.',
            '...0344543320', '..03455443210', '.023333332210', '.0iiiiiii0210',
            '.0j0jj0ji0210', '.0j0jj0ji0210', '..0jjjjii0210', '...0iii00210.'),
    'S': g('.....0d0.....', '.....0c0.....', '...000b000...', '..034454320..',
           '.03455443210.', '.02333333210.', '.0iiiiiiiii0.', '02jj0jjj0jj20',
           '02jj0jjj0jj20', '02jjjjjjjii20', '.010iiiii010.', '....00000....'),
    'W': g('.......0d0...', '......0dc0...', '....000bb000.', '...0344543320',
           '..03455443210', '0233333332210', '.0iiiiii02210', '.0j0jjji02210',
           '0jj0jjji02210', '.0jjjjii02210', '..0iiii0210..', '...00000.....'),
    'NW': g('......0d0....', '.....0cd0....', '....000bb000.', '...0344543320',
            '..03445433210', '.023333332210', '.0i0233332210', '.0j0233332210',
            '.0i0223322210', '..00222222210', '...011111110.', '....0000000..'),
    'N': g('.....0d0.....', '.....0c0.....', '...000b000...', '..034454320..',
           '.03445543210.', '.02333333210.', '.02334433210.', '.02333333210.',
           '.02233332210.', '.01222222110.', '..011111110..', '...0000000...'),
}
TOPKNOT = {
    'SW': g('......000....', '.....01110...', '....0011100..', '...001111100.',
            '..0mmmmmmml0.', '.0mmmmmllkk0m', '.0lllllkkk0ml', '.0iiiiiii110k',
            '.0j0jj0ij110.', '.0j0jj0ii110.', '..0jjjjii10..', '...0iiii00...'),
    'S': g('.....000.....', '....01110....', '...0011100...', '..001111100..',
           '.0mmmmmmmll0.', '.0lllllllkk0.', '.0iiiiiiiii0.', '.0jj0jjj0jj0.',
           '.0jj0jjj0jj0.', '.0jjjjjjjii0.', '..0iijjjii0..', '...0iiiii0...'),
    'W': g('........000..', '.......01110.', '....00011100.', '...0011111100',
           '..0mmmmmml110', '.0mmmmmllk110', '.0iiiiii01110', '.0j0jjji01110',
           '0jj0jjji0110.', '.0jjjjii0110.', '..0iiii010...', '...00000.....'),
    'NW': g('......000....', '.....01210...', '....0012100..', '...001221100.',
            '..0mmmmmmml0.', '.0mmllllkkk0m', '.0i01222111ml', '.0j01221111k.',
            '.0i011111110.', '..0011111100.', '...0111110...', '....00000....'),
    'N': g('.....000.....', '....01210....', '...0012100...', '..001221100..',
           '.0mmmmmmmll0.', '.0llllllkkkm.', '.01222211110m', '.01221111110l',
           '.01111111110k', '..011111110..', '...0111110...', '....00000....'),
}
TOPKNOT = {d: recolor(rows, {'1': 'f', '2': 'g'}) for d, rows in TOPKNOT.items()}  # dark brown hair
HOOD = {
    'SW': g('.............', '.............', '....000000...', '...08999880..',
            '..089aaa98870', '.089aaa998870', '.077777788870', '.0iiiiii8870.',
            '.0j0jj0i8870.', '.0j0jj0i8870.', '..0jjjii8870.', '...0iii08870.'),
    'S': g('.............', '.............', '...0000000...', '..089aaa980..',
           '.089aaaa9880.', '.089aa998870.', '.07777777770.', '08iiiiiiiii80',
           '08jj0jjj0jj80', '08jj0jjj0jj80', '08jjjjjjjii80', '.080iiiii080.'),
    'W': g('.............', '.............', '.....000000..', '....08999880.',
           '...089aaa9870', '..089aa998870', '.077777788870', '.0iiiiii08870',
           '.0j0jjji08870', '0jj0jjji08870', '.0jjjjii0870.', '..0iiii0870..'),
    'NW': g('.............', '.............', '....000000...', '...08999880..',
            '..089aaa98870', '.089aa9988870', '.077888888870', '.0i0888888870',
            '.0j0888888870', '.0i0788888870', '..00778888870', '...077777770.'),
    'N': g('.............', '.............', '...0000000...', '..089aaa980..',
           '.089aaaa9880.', '.089aa998870.', '.08899888870.', '.08888888870.',
           '.08888888870.', '.07888888770.', '..077777770..', '...0000000...'),
}


def find_eyes(g, skin):
    """Top pixels of the 2px-tall eyes: '0' over '0' with skin on both sides."""
    h, w = len(g), len(g[0])
    return [(y, x) for y in range(h - 1) for x in range(1, w - 1)
            if g[y][x] == '0' and g[y+1][x] == '0' and g[y][x-1] in skin and g[y][x+1] in skin]


def outline_beard(g, start):
    for y in range(max(0, start - 1), len(g)):
        for x in range(len(g[0])):
            if g[y][x] == '.' and any(0 <= y+dy < len(g) and 0 <= x+dx < len(g[0]) and g[y+dy][x+dx] == 'B'
                                      and y+dy >= start for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1))):
                g[y][x] = '0'


def beard(rows, skin, kind):
    """'long': mustache and a long tapering beard (Guan Yu); 'bristle': bushy jaw and spikes;
    'goatee': mustache and a short pointed chin beard (Liu Bei); 'bushy': a full dwarf-like beard from
    the cheeks down, wide and shaggy, dark brown with lighter strands (Zhang Fei).
    Beard pixels are marked 'B' while drawing so they never mix with dark helmet pixels."""
    g = [list(r) for r in rows]
    eyes = find_eyes(g, skin)
    if not eyes:  # back view
        return rows
    w = len(g[0])
    ey, xs = eyes[0][0], sorted(x for _, x in eyes)
    profile = len(xs) == 1
    chin = max(y for y, row in enumerate(g) if any(c in skin for c in row))
    if kind == 'bushy':
        return bushy(g, ey, chin, skin)
    if kind in ('long', 'goatee'):
        x0, x1 = (xs[0] - 2, xs[0] + 1) if profile else (xs[0] - 1, xs[-1] + 1)
        for x in range(max(0, x0), min(w, x1 + 1)):
            if g[ey + 3][x] in skin:
                g[ey + 3][x] = 'B'
        lower = [chin]
    else:
        lower = range(ey + 3, chin + 1)
    for y in lower:
        for x in range(w):
            if g[y][x] in skin:
                g[y][x] = 'B'
    cols = [x for x, c in enumerate(g[chin]) if c == 'B']
    left, right = min(cols), max(cols)
    start = chin + 1  # continue right under the chin, over its old outline row
    g += [['.'] * w for _ in range(max(0, start + {'long': 6, 'goatee': 3}.get(kind, 2) - len(g)))]
    if kind == 'goatee':
        mid = (left + right)//2 - profile
        for i, half in enumerate((1, 0)):
            for x in range(mid - half, mid + half + 1):
                g[start + i][x] = 'B'
    elif kind == 'long':
        for i in range(5):
            l, r = (left - (1 <= i <= 2), right - i) if profile else (left + i//2, right - (i + 1)//2)
            for x in range(max(0, l), min(w, r + 1)):
                g[start + i][x] = 'B'
    else:
        for x in range(left, right + 1, 2):
            g[start][x] = 'B'
    outline_beard(g, start)
    return [''.join(r).replace('B', '1') for r in g]


def bushy(g, ey, chin, skin):
    w = len(g[0])
    for y in range(ey + 2, chin + 1):  # everything below the eyes
        for x in range(w):
            if g[y][x] in skin:
                g[y][x] = 'B'
    cols = [x for x, c in enumerate(g[chin]) if c == 'B']
    left, right = min(cols), max(cols)
    start = chin + 1
    g += [['.'] * w for _ in range(max(0, start + 5 - len(g)))]
    for i, (l, r) in enumerate([(left - 1, right + 1), (left - 1, right + 1), (left, right), (left + 1, right - 1)]):
        for x in range(max(0, l), min(w, r + 1)):
            g[start + i][x] = 'B'
    outline_beard(g, start)
    last = start + 3
    hair = [[c == 'B' for c in row] for row in g]
    for y in range(len(g)):  # light from the upper left: lit edges and a few strands, dark tips
        for x in range(w):
            if not hair[y][x]:
                continue
            edge = y == 0 or not hair[y-1][x] or x == 0 or not hair[y][x-1]
            g[y][x] = '1' if y == last else 'g' if edge or (x + 2*y) % 5 == 0 else 'f'
    return [''.join(r) for r in g]


# Spearman: pointed cone helmet on the iron-helmet face.
CONE_TOPS = {
    'SW': ['.......0.....', '......040....', '.....03430...', '....0345430..'],
    'S': ['......0......', '.....040.....', '....03430....', '...0345430...', '.03455544310.'],
    'W': ['.......0.....', '......040....', '.....03430...', '....034554320'],
    'NW': ['......0......', '.....040.....', '....03430....', '...034554320.'],
    'N': ['......0......', '.....040.....', '....03430....', '...0345430...'],
}
CONE = {d: top + IRON[d][len(top):] for d, top in CONE_TOPS.items()}
# Guan Yu: green head wrap, ruddy face, long black beard.
GUANYU = {d: beard(recolor(rows, {'7': 'G', '8': 'H', '9': 'I', 'a': 'J', 'j': 'L', 'i': 'K'}), 'KL', 'long')
          for d, rows in HOOD.items()}
# Zhang Fei: blackened helmet, swarthy face, bristling beard.
ZHANGFEI = {d: beard(recolor(rows, {'5': '4', '4': '3', '3': '2', '2': '1', 'j': 'i', 'i': 'h'}), 'hi', 'bushy')
            for d, rows in IRON.items()}

# Faction-colored head pixels -> team keys.
PLUME, BAND, HOOD_CLOTH = {'b': 'X', 'c': 'Y', 'd': 'Z'}, {'k': 'X', 'l': 'Y', 'm': 'Z'}, {'7': 'W', '8': 'X', '9': 'Y', 'a': 'Z'}


def heads(grids, team=None, skin='ij', lid='i', crest=0):
    """A head set: grids per view, team key mapping applied, and the face's skin/eyelid keys.
    crest: rows added above the usual 12-row head (feathers), so the face keeps its anchor."""
    if team:
        grids = {d: recolor(rows, team) for d, rows in grids.items()}
    return dict(grids=grids, skin=skin, lid=lid, anchor=(ANCHOR[0], ANCHOR[1] + crest))


def head_for(grids, d):
    if d in MIRROR:
        return [row[::-1] for row in grids[MIRROR[d]]]
    return grids[d]


def expression(head, kind, skin, lid):
    """Rewrite the 2px-tall eyes: 'hurt' squeezes them shut, 'tired' drops the lids."""
    if kind == 'normal':
        return head
    g = [list(r) for r in head]
    h, w = len(g), len(g[0])
    eyes = find_eyes(g, skin)
    if not eyes:  # back view
        return head
    for y, x in eyes:
        g[y][x] = g[y][x-1] if kind == 'hurt' else lid
        if kind == 'hurt' and g[y+1][x+1] in skin:
            g[y+1][x+1] = '0'
    y, x = eyes[0][0] + 3, round(sum(x for _, x in eyes)/len(eyes)) - (len(eyes) == 1)
    if y < h and g[y][x] in skin:
        g[y][x] = 'b'
        if kind == 'hurt' and y + 1 < h and g[y+1][x] in skin:
            g[y+1][x] = 'b'
    if kind == 'tired':
        ty = eyes[0][0] - 1
        tx = max(i for i, c in enumerate(g[ty]) if c != '.') + 1
        if tx < w:
            g[ty][tx], g[ty+1][tx] = 'q', 'p'
    return [''.join(r) for r in g]


def stamp(g, head_set, d, face, cx, cy):
    """Stamp the view's head, with the frame's expression, centred on (cx, cy)."""
    head = expression(head_for(head_set['grids'], d), face, head_set['skin'], head_set['lid'])
    ax, ay = head_set.get('anchor', ANCHOR)
    overlay(g, head, round(cx) - ax, round(cy) - ay)


# ---- ver4 heads (candidates; see look_candidates.py) ---------------------------------------

def topknot_crown(rows):
    """Gold topknot crown: the hair bun above the band becomes a small gold crown."""
    out = []
    for y, row in enumerate(rows):
        if y < 4:
            row = row.translate(str.maketrans({'f': 'l', 'g': 'k'}))
            i = row.find('l')
            if i >= 0:
                row = row[:i] + 'm' + row[i+1:]
        out.append(row)
    return out


def band_to_hair(rows):
    """Turn the topknot band (and its loose tails) into dark hair."""
    return recolor(rows, {'m': 'g', 'l': 'g', 'k': 'f'})


def flat_cap(rows):
    """Iron helmet without the plume: a flat scholar cap once darkened."""
    return [r if y >= 3 else '.' * len(r) for y, r in enumerate(rows)]


def gwanmo(rows, style=1):
    """Scholar's cap (관모) on the topknot: the face, ears and hairline stay open (the old flat_cap kept
    the helmet's cheek guards and read as a hood). style 1 small cap, 2 tall cap, 3 cap with a black
    headband over the hairline, 4 small cap with a gold pin."""
    out = []
    for y, row in enumerate(rows):
        if y < 4 or (style == 3 and y == 4):
            row = row.translate(str.maketrans({'f': '2', 'g': '3'}))
        if style == 4 and y == 2:
            i, j = row.find('0'), row.rfind('0')
            row = row[:i] + 'm' + row[i + 1:j] + 'm' + row[j + 1:]
        out.append(row)
    if style == 2:  # two more rows of cap; heads(..., crest=2) keeps the face anchor
        out = out[:2] + out[1:2] + out[1:]
    return out


DARK_IRON = {'5': '3', '4': '3', '3': '2', '2': '1'}
GOLD_IRON = {'2': 'k', '3': 'l', '4': 'm', '5': 'n'}
WHITE_HOOD = {'7': '3', '8': '4', '9': '5', 'a': '6'}
YELLOW_HOOD = {'7': 'k', '8': 'l', '9': 'm', 'a': 'n'}
HAIR = {d: band_to_hair(rows) for d, rows in TOPKNOT.items()}
CROWN = {d: topknot_crown(rows) for d, rows in HAIR.items()}

# Pheasant tail feathers above Lu Bu's crown (FEATHER_ROWS rows over the 12-row head).
FEATHERS = {
    'S': ['.0.........0.', '.h0.......0h.', '..h0.....0h..', '..ih.....hi..',
          '...ih...hi...', '....h...h....', '....ih.hi....', '.....h.h.....'],
    'W': ['.........0..0', '........h0.h.', '.......ih.ih.', '......ih.ih..',
          '.....ih.ih...', '.....h.ih....', '....ih.h.....', '.....hh......'],
}
FEATHERS.update(SW=FEATHERS['S'], NW=FEATHERS['W'], N=FEATHERS['S'])
FEATHER_ROWS = 8


def crested(grids, crest):
    return {d: crest[d] + rows for d, rows in grids.items()}
