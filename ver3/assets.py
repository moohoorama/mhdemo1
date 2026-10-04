"""Loader for ver3/asset: reads assets.yaml and the map YAML, slices frames out of the
one-image-per-character sheets and swaps team colors per faction (cached). Drop shadows
are rebuilt first when their sources changed (shadows.py)."""
from pathlib import Path

import pygame
import yaml

import shadows

ASSET = Path(__file__).resolve().parent / 'asset'
SHADOW_RGB = (18, 26, 14)  # dark olive; shadow images carry only alpha


def rgb(hex_color):
    return tuple(int(hex_color[i:i+2], 16) for i in (1, 3, 5))


def load_shadow(path):
    image = pygame.image.load(ASSET / path).convert_alpha()
    image.fill(SHADOW_RGB + (0,), special_flags=pygame.BLEND_RGBA_ADD)
    return image


class UnitArt:
    """One character image: a row per facing, animation frames side by side."""

    def __init__(self, key, spec, team_keys, shadow):
        self.key, self.name = key, spec['name']
        self.cell, self.pivot = spec['cell'], spec['pivot']
        self.rows = {d: i for i, d in enumerate(spec['rows'])}
        self.animations = spec['animations']
        self.image = pygame.image.load(ASSET / spec['image']).convert_alpha()
        self.team_keys = [rgb(c) for c in team_keys]
        self._sheets, self._frames = {}, {}
        self.shadow_image = load_shadow(shadow['image'])
        self.shadow_cell, self.shadow_pivot = shadow['cell'], shadow['pivot']
        self._shadows = {}

    def sheet(self, ramp):
        """The image with the team key colors replaced by a faction ramp."""
        key = tuple(ramp)
        if key not in self._sheets:
            sheet = self.image.copy()
            pixels = pygame.PixelArray(sheet)
            for src, dst in zip(self.team_keys, ramp):
                pixels.replace(src, rgb(dst))
            del pixels
            self._sheets[key] = sheet
        return self._sheets[key]

    def frame(self, ramp, direction, animation, index):
        key = (tuple(ramp), direction, animation, index)
        if key not in self._frames:
            w, h = self.cell
            column = self.animations[animation]['first_column'] + index
            rect = pygame.Rect(column*w, self.rows[direction]*h, w, h)
            self._frames[key] = self.sheet(ramp).subsurface(rect)
        return self._frames[key]

    def shadow(self, direction, animation, index):
        """Shadow of a frame and its pivot (where the unit's feet go)."""
        key = (direction, animation, index)
        if key not in self._shadows:
            w, h = self.shadow_cell
            column = self.animations[animation]['first_column'] + index
            self._shadows[key] = self.shadow_image.subsurface(pygame.Rect(column*w, self.rows[direction]*h, w, h))
        return self._shadows[key], self.shadow_pivot

    def frame_index(self, animation, ms):
        """Frame shown `ms` after the animation started; non-looping ones hold the last frame."""
        spec = self.animations[animation]
        durations = spec['ms']
        total = sum(durations)
        if spec['loop']:
            ms %= total
        elif ms >= total:
            return len(durations) - 1
        for i, d in enumerate(durations):
            if ms < d:
                return i
            ms -= d
        return len(durations) - 1

    def length(self, animation):
        return sum(self.animations[animation]['ms'])


class Tileset:
    def __init__(self, spec, shadow):
        self.image = pygame.image.load(ASSET / spec['image']).convert_alpha()
        self.sprites = {int(i): (self.image.subsurface(pygame.Rect(x, y, w, h)), (px, py))
                        for i, (x, y, w, h, px, py) in spec['sprites'].items()}
        image = load_shadow(shadow['image'])
        self.shadows = {int(i): (image.subsurface(pygame.Rect(x, y, w, h)), (px, py))
                        for i, (x, y, w, h, px, py) in shadow['sprites'].items()}
        self.animations = spec['animations']

    def animation_frame(self, name, ms):
        a = self.animations[name]
        return a['frames'][int(ms // a['ms']) % len(a['frames'])]


class GameMap:
    """Map draw data plus the cell grid used for movement."""

    def __init__(self, path):
        data = yaml.safe_load((ASSET / path).read_text())
        self.name = data['name']
        self.width, self.height = data['size']
        self.cells = data['cells']
        self.bounds = data['bounds']
        self.ground = data['ground']
        self.decorations = data['decorations']

    def walkable(self, u, v):
        return 0 <= u < self.width and 0 <= v < self.height and self.cells[v][u] in '.:R'

    @staticmethod
    def project(u, v):
        return (u - v) * 16, (u + v) * 8


class Assets:
    def __init__(self):
        index = yaml.safe_load((ASSET / 'assets.yaml').read_text())
        cast = shadows.refresh(index, ASSET)
        team_keys = index['factions']['team_keys']
        self.factions = {f['id']: f for f in index['factions']['factions']}
        self.units = {k: UnitArt(k, spec, team_keys, cast['units'][k]) for k, spec in index['units'].items()}
        self.tileset = Tileset(index['tileset'], cast['tileset'])
        self.maps = [GameMap(p) for p in index['maps']]
