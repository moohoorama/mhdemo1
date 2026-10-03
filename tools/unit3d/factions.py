"""Faction colors for the team-colored parts of ver2 units.

Sprites draw every team-colored part (tunic, hood, sash, saddle, plume, turban band)
with four reserved key colors (TEAM_KEYS, a blue ramp used nowhere else). A faction is
just a hue plus chroma and lightness adjustments applied in OKLCH, so the four
shading steps keep their contrast for every hue. White and navy need the chroma and
lightness terms; similar hues are told apart by each unit's own silhouette.
"""
import math

TEAM_KEYS = ['#1b3358', '#264d80', '#3567a6', '#5a8fd0']  # shadow, base, light, highlight

# hue in degrees (OKLCH), chroma multiplier, lightness offset
FACTIONS = [
    dict(id='wei', name='위', hue=None, chroma=1.0, lightness=0.0),        # the key ramp itself (blue)
    dict(id='shu', name='촉', hue=92, chroma=1.7, lightness=.26),
    dict(id='wu', name='오', hue=27, chroma=1.45, lightness=.06),
    dict(id='turban', name='황건적', hue=74, chroma=1.5, lightness=.19),
    dict(id='dong', name='동탁·여포', hue=268, chroma=.75, lightness=-.08),
    dict(id='gongsun', name='공손찬', hue=250, chroma=.1, lightness=.4),
    dict(id='yuan', name='원소·원술', hue=104, chroma=1.6, lightness=.3),
    dict(id='barbarian', name='이민족', hue=312, chroma=1.05, lightness=.04),
]


def _lin(c):
    c /= 255
    return c/12.92 if c <= .04045 else ((c + .055)/1.055)**2.4


def _gam(c):
    c = c*12.92 if c <= .0031308 else 1.055*c**(1/2.4) - .055
    return c*255


def to_oklch(hex_color):
    r, g, b = (_lin(int(hex_color[i:i+2], 16)) for i in (1, 3, 5))
    l = (.4122214708*r + .5363325363*g + .0514459929*b) ** (1/3)
    m = (.2119034982*r + .6806995451*g + .1073969566*b) ** (1/3)
    s = (.0883024619*r + .2817188376*g + .6299787005*b) ** (1/3)
    L = .2104542553*l + .7936177850*m - .0040720468*s
    a = 1.9779984951*l - 2.4285922050*m + .4505937099*s
    bb = .0259040371*l + .7827717662*m - .8086757660*s
    return L, math.hypot(a, bb), math.degrees(math.atan2(bb, a)) % 360


def _rgb(L, C, h):
    a, b = C*math.cos(math.radians(h)), C*math.sin(math.radians(h))
    l = (L + .3963377774*a + .2158037573*b)**3
    m = (L - .1055613458*a - .0638541728*b)**3
    s = (L - .0894841775*a - 1.2914855480*b)**3
    return (4.0767416621*l - 3.3077115913*m + .2309699292*s,
            -1.2684380046*l + 2.6097574011*m - .3413193965*s,
            -.0041960863*l - .7034186147*m + 1.7076147010*s)


def from_oklch(L, C, h):
    """Nearest in-gamut sRGB: chroma is reduced until the color fits."""
    for _ in range(40):
        rgb = _rgb(L, C, h)
        if all(-1e-4 <= v <= 1 + 1e-4 for v in rgb):
            break
        C *= .92
    return '#' + ''.join(f'{round(min(255, max(0, _gam(min(1, max(0, v)))))):02x}' for v in rgb)


def ramp(faction):
    if faction['hue'] is None and faction['chroma'] == 1 and faction['lightness'] == 0:
        return list(TEAM_KEYS)
    out = []
    for key in TEAM_KEYS:
        L, C, h = to_oklch(key)
        hue = h if faction['hue'] is None else faction['hue']
        out.append(from_oklch(min(.97, max(.05, L + faction['lightness'])), C*faction['chroma'], hue))
    return out


def table():
    """JSON-ready faction table: the key colors and each faction's replacement ramp."""
    return dict(team_keys=TEAM_KEYS,
                factions=[dict(id=f['id'], name=f['name'], hue=f['hue'], chroma=f['chroma'],
                               lightness=f['lightness'], ramp=ramp(f)) for f in FACTIONS])


if __name__ == '__main__':
    for f in table()['factions']:
        print(f['id'], f['name'], f['ramp'])
