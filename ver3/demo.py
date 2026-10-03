#!/usr/bin/env python3
"""ver3 battle demo (pygame): two factions take turns on the demo map.

Each turn the active side reinforces from its map edge, then its units, starting a
moment apart, walk toward the nearest enemy and attack when in reach. The blow lands on the attack's third
frame: the target plays its hit motion and shakes sideways, away from the attacker
first. At half health or less a unit idles in its exhausted motion; at zero it
blinks and disappears.

  python3 ver3/demo.py                      # window
  python3 ver3/demo.py --wave 8 --fps 30    # start with 8 reinforcements per turn at 30 fps
  python3 ver3/demo.py --shots out 3 6 12   # headless: save frames at 3s, 6s, 12s
The bottom bar picks the reinforcement size (1-16 per turn) and the frame rate.
Keys: Space pause, F fast forward, R restart, 1-5 reinforcement size, [ ] frame rate, Esc quit.
"""
import argparse
import math
import os
import random
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT))

SCALE = 2
FPS_CHOICES = [15, 30, 60, 120]   # frames per second; motion is timed in ms, so this only sets smoothness
WAVES = [1, 2, 4, 8, 16]          # reinforcements per turn
STEP = .28            # seconds per tile while walking
STAGGER = .25         # start delay between units acting in the same turn
SIDES = [
    dict(faction='shu', units=['guanyu', 'zhangfei', 'infantry', 'spearman', 'archer', 'cavalry'], edge='west'),
    dict(faction='wei', units=['infantry', 'spearman', 'archer', 'cavalry'], edge='east'),
]
HEROES = {'guanyu', 'zhangfei'}
# hp, attack range (min, max), tiles per turn, reach (Chebyshev)
STATS = {
    'infantry': (100, (26, 34), 3, 1), 'bandit': (90, (24, 32), 3, 1), 'spearman': (100, (28, 36), 3, 1),
    'archer': (80, (22, 30), 3, 3), 'cavalry': (120, (30, 40), 5, 1),
    'guanyu': (220, (44, 56), 5, 1), 'zhangfei': (200, (42, 54), 5, 1),
}
DIRECTIONS = ['E', 'SE', 'S', 'SW', 'W', 'NW', 'N', 'NE']  # by screen angle, 45 degree steps from +x
NEIGHBORS = [(du, dv) for du in (-1, 0, 1) for dv in (-1, 0, 1) if du or dv]
SHAKE_TIME, SHAKE_PX, SHAKE_HZ = .45, 3, 11
BLINK_TIME = 1.2


def facing(du, dv):
    """Screen direction of a map step: the iso projection turns (du, dv) into screen x/y."""
    sx, sy = (du - dv)*16, (du + dv)*8
    angle = math.degrees(math.atan2(sy, sx)) % 360
    return DIRECTIONS[int((angle + 22.5) // 45) % 8]


class Unit:
    def __init__(self, art, side, ramp, cell):
        self.art, self.side, self.ramp = art, side, ramp
        hp, self.attack, self.move, self.reach = STATS[art.key]
        self.hp = self.max_hp = hp
        self.cell = cell
        self.pos = [float(cell[0]), float(cell[1])]
        self.direction = 'SE' if side == 0 else 'NW'
        self.anim, self.anim_ms = 'idle', 0.0
        self.walk = None          # (from, to, elapsed)
        self.shake = None         # (sign, elapsed)
        self.dying = None         # blink elapsed
        self.appear = .5          # spawn blink
        self.gone = False
        self.top = art.frame(ramp, 'S', 'idle', 0).get_bounding_rect().top

    @property
    def alive(self):
        return self.hp > 0

    def rest_anim(self):
        return 'exhausted' if self.hp <= self.max_hp/2 else 'idle'

    def play(self, anim):
        self.anim, self.anim_ms = anim, 0.0

    def face(self, other_cell):
        self.direction = facing(other_cell[0] - self.cell[0], other_cell[1] - self.cell[1])

    def step_to(self, cell):
        self.face(cell)
        self.walk = (self.cell, cell, 0.0)
        self.cell = cell
        if self.anim != 'walk':
            self.play('walk')

    def update(self, dt):
        self.anim_ms += dt*1000
        if self.walk:
            a, b, t = self.walk
            t = min(STEP, t + dt)
            k = t/STEP
            self.pos = [a[0] + (b[0] - a[0])*k, a[1] + (b[1] - a[1])*k]
            self.walk = None if t >= STEP else (a, b, t)
        anim = self.art.animations[self.anim]
        if not anim['loop'] and self.anim_ms >= self.art.length(self.anim) and self.dying is None:
            self.play(self.rest_anim())
        if self.shake:
            sign, t = self.shake
            self.shake = (sign, t + dt) if t + dt < SHAKE_TIME else None
        if self.dying is not None:
            self.dying += dt
            self.gone = self.dying >= BLINK_TIME
        self.appear = max(0.0, self.appear - dt)

    def visible(self):
        if self.dying is not None:
            return int(self.dying*12) % 2 == 0
        return int(self.appear*16) % 2 == 0

    def foot(self, origin):
        x, y = (self.pos[0] - self.pos[1])*16, (self.pos[0] + self.pos[1] + 1)*8
        dx = 0
        if self.shake:
            sign, t = self.shake
            dx = sign*SHAKE_PX*math.sin(2*math.pi*SHAKE_HZ*t)*(1 - t/SHAKE_TIME)
        return origin[0] + x + dx, origin[1] + y


class Battle:
    def __init__(self, assets, rng, wave=1):
        self.assets, self.rng, self.wave = assets, rng, wave
        self.map = assets.maps[0]
        self.units, self.popups = [], []
        self.side, self.banner = 0, ''
        self.script = self.run()
        self.wait = 0.0

    # ---- board -------------------------------------------------------------
    def occupied(self, cell, ignore=None):
        return any(u.cell == cell and u is not ignore and not u.gone for u in self.units)

    def enemies(self, unit):
        return [u for u in self.units if u.side != unit.side and u.alive]

    def in_reach(self, unit, cell, enemy):
        return max(abs(cell[0] - enemy.cell[0]), abs(cell[1] - enemy.cell[1])) <= unit.reach

    def plan(self, unit):
        """BFS to the nearest cell from which an enemy is within reach; returns (path, target)."""
        enemies = self.enemies(unit)
        if not enemies:
            return [], None
        start = unit.cell
        prev, queue, head = {start: None}, [start], 0
        while head < len(queue):
            cell = queue[head]
            head += 1
            near = [e for e in enemies if self.in_reach(unit, cell, e)]
            if near:
                path = []
                while cell != start:
                    path.append(cell)
                    cell = prev[cell]
                return path[::-1], min(near, key=lambda e: e.hp)
            for du, dv in NEIGHBORS:
                nxt = (cell[0] + du, cell[1] + dv)
                if nxt not in prev and self.map.walkable(*nxt) and not self.occupied(nxt, unit):
                    prev[nxt] = cell
                    queue.append(nxt)
        return [], None

    def cap(self):
        """Units per side on the field."""
        return max(6, self.wave*3)

    def spawn(self, side):
        """Reinforce from the side's map edge; when the edge column is full, use the next ones inward."""
        spec = SIDES[side]
        mine = [u for u in self.units if u.side == side and u.alive]
        count = min(self.wave*(1 if mine else 2), self.cap() - len(mine))
        ramp = self.assets.factions[spec['faction']]['ramp']
        for depth in range(4):
            u = depth if spec['edge'] == 'west' else self.map.width - 1 - depth
            cells = [(u, v) for v in range(self.map.height) if self.map.walkable(u, v) and not self.occupied((u, v))]
            self.rng.shuffle(cells)
            for cell in cells[:max(0, count)]:
                present = {x.art.key for x in self.units if x.side == side and x.alive}
                keys = [k for k in spec['units'] if k not in HEROES or k not in present]
                self.units.append(Unit(self.assets.units[self.rng.choice(keys)], side, ramp, cell))
                count -= 1

    def strike(self, attacker, target, ox):
        if not target.alive:  # another unit finished it during the wind-up
            return
        damage = self.rng.randint(*attacker.attack)
        target.hp = max(0, target.hp - damage)
        target.play('hit')
        # shake away from the attacker first (hit from the left -> pushed right)
        sign = 1 if attacker.foot(ox)[0] <= target.foot(ox)[0] else -1
        target.shake = (sign, 0.0)
        if not target.alive:
            target.dying = 0.0
        self.popups.append([target, f'-{damage}', 0.0])

    # ---- turn script (a generator yielding seconds to wait) -----------------
    def run(self):
        while True:
            spec = SIDES[self.side]
            self.banner = f"{self.assets.factions[spec['faction']]['name']} 진영의 턴"
            self.spawn(self.side)
            yield .9
            order = sorted((u for u in self.units if u.side == self.side and u.alive),
                           key=lambda u: len(self.plan(u)[0]))
            yield from self.together([self.act(u) for u in order])
            self.side ^= 1
            yield .4

    @staticmethod
    def together(scripts):
        """Run unit scripts side by side, each starting STAGGER seconds after the previous one."""
        timers = [[script, i*STAGGER] for i, script in enumerate(scripts)]
        while timers:
            wait = min(t for _, t in timers)
            if wait > 0:
                yield wait
            for entry in timers:
                entry[1] -= wait
            for entry in [e for e in timers if e[1] <= 1e-9]:
                try:
                    entry[1] = next(entry[0])
                except StopIteration:
                    timers.remove(entry)

    def act(self, unit):
        if not unit.alive:
            return
        path, target = self.plan(unit)
        for cell in path[:unit.move]:
            if not unit.alive or self.occupied(cell, unit):  # another unit took the cell meanwhile
                break
            unit.step_to(cell)
            yield STEP
        if unit.anim == 'walk':
            unit.play(unit.rest_anim())
        target = min((e for e in self.enemies(unit) if self.in_reach(unit, unit.cell, e)),
                     key=lambda e: e.hp, default=None)
        if not target:
            yield .1
            return
        unit.face(target.cell)
        unit.play('attack')
        ms = unit.art.animations['attack']['ms']
        yield sum(ms[:2])/1000          # gather power
        self.strike(unit, target, (0, 0))
        yield sum(ms[2:])/1000 + .25     # follow through

    def update(self, dt):
        self.wait -= dt
        while self.wait <= 0:
            self.wait += next(self.script)
        for u in self.units:
            u.update(dt)
        self.units = [u for u in self.units if not u.gone]
        for p in self.popups:
            p[2] += dt
        self.popups = [p for p in self.popups if p[2] < .9]


class View:
    def __init__(self, assets, font_path):
        import pygame
        self.pg = pygame
        self.assets = assets
        m = assets.maps[0]
        x0, y0, x1, y1 = m.bounds
        self.origin = (-x0 + 12, -y0 + 64)
        self.size = (x1 - x0 + 24, y1 - y0 + 84)
        self.surface = pygame.Surface(self.size, pygame.SRCALPHA)
        self.font = pygame.font.Font(font_path, 13)
        self.big = pygame.font.Font(font_path, 18)
        self.grounds = [self.bake_ground(m, frame) for frame in range(8)]

    def bake_ground(self, m, water_frame):
        pg = self.pg
        surf = pg.Surface(self.size, pg.SRCALPHA)
        for x, y, layers in m.ground:
            for sid in layers:
                image, (px, py) = self.assets.tileset.sprites[water_frame if sid == 0 else sid]
                surf.blit(image, (self.origin[0] + x - px, self.origin[1] + y - py))
        return surf

    def draw(self, battle, clock_ms):
        pg = self.pg
        s = self.surface
        s.fill((30, 38, 46, 255))
        tiles = self.assets.tileset
        s.blit(self.grounds[tiles.animation_frame('water', clock_ms)], (0, 0))
        items = []
        for x, y, anim, phase in battle.map.decorations:
            sid = tiles.animation_frame(anim, clock_ms + phase*125)
            image, (px, py) = tiles.sprites[sid]
            items.append((self.origin[1] + y, 0, image, (self.origin[0] + x - px, self.origin[1] + y - py)))
        for u in battle.units:
            fx, fy = u.foot(self.origin)
            items.append((fy, 1, u, (fx, fy)))
        for depth, kind, obj, pos in sorted(items, key=lambda i: (i[0], i[1])):
            if kind == 0:
                s.blit(obj, pos)
            else:
                self.draw_unit(obj, pos)
        for unit, text, t in battle.popups:
            fx, fy = unit.foot(self.origin)
            img = self.font.render(text, True, (255, 236, 160))
            s.blit(img, (fx - img.get_width()/2, fy - unit.art.pivot[1] + unit.top - 12 - t*18))
        banner = self.big.render(battle.banner, True, (245, 240, 225))
        pg.draw.rect(s, (0, 0, 0, 150), (8, 8, banner.get_width() + 16, banner.get_height() + 8), border_radius=6)
        s.blit(banner, (16, 12))
        return s

    def draw_unit(self, u, foot):
        pg = self.pg
        fx, fy = foot
        ring = pg.Surface((22, 9), pg.SRCALPHA)
        r, g, b = (int(u.ramp[2][i:i+2], 16) for i in (1, 3, 5))
        pg.draw.ellipse(ring, (r, g, b, 120), ring.get_rect())
        self.surface.blit(ring, (fx - 11, fy - 5))
        if not u.visible():
            return
        art = u.art
        frame = art.frame(u.ramp, u.direction, u.anim, art.frame_index(u.anim, u.anim_ms))
        self.surface.blit(frame, (round(fx - art.pivot[0]), round(fy - art.pivot[1])))
        if u.alive:
            top = fy - art.pivot[1] + u.top - 4
            pg.draw.rect(self.surface, (20, 20, 24), (fx - 9, top, 18, 3))
            k = u.hp/u.max_hp
            color = (96, 214, 110) if k > .5 else (236, 184, 64) if k > .25 else (226, 72, 60)
            pg.draw.rect(self.surface, color, (fx - 8, top + 1, max(1, round(16*k)), 1))


FONT = '/System/Library/Fonts/AppleSDGothicNeo.ttc'
PANEL_H = 52


class Controls:
    """Bottom bar: reinforcement size and frame rate buttons, live counters."""

    def __init__(self, pygame, font_path, width, top):
        self.pg, self.top, self.width = pygame, top, width
        self.font = pygame.font.Font(font_path, 18)
        self.small = pygame.font.Font(font_path, 15)
        self.buttons = []  # (rect, kind, value)
        x = 16
        for kind, label, values in (('wave', '증원', WAVES), ('fps', 'FPS', FPS_CHOICES)):
            x += self.font.size(label)[0] + 10
            for v in values:
                w = self.font.size(str(v))[0] + 18
                self.buttons.append((pygame.Rect(x, top + 10, w, 32), kind, v))
                x += w + 6
            x += 24
        self.labels_x = x

    def click(self, pos):
        for rect, kind, value in self.buttons:
            if rect.collidepoint(pos):
                return kind, value
        return None

    def draw(self, screen, battle, fps, measured):
        pg = self.pg
        pg.draw.rect(screen, (22, 28, 36), (0, self.top, self.width, PANEL_H))
        x = 16
        for kind, label in (('wave', '증원'), ('fps', 'FPS')):
            first = next(r for r, k, _ in self.buttons if k == kind)
            screen.blit(self.font.render(label, True, (190, 205, 220)), (first.x - self.font.size(label)[0] - 10, self.top + 14))
        for rect, kind, value in self.buttons:
            active = value == (battle.wave if kind == 'wave' else fps)
            pg.draw.rect(screen, (56, 87, 115) if active else (38, 49, 62), rect, border_radius=6)
            pg.draw.rect(screen, (131, 200, 255) if active else (65, 80, 100), rect, 1, border_radius=6)
            text = self.font.render(str(value), True, (237, 243, 250))
            screen.blit(text, text.get_rect(center=rect.center))
        counts = [sum(1 for u in battle.units if u.side == i and u.alive) for i in range(2)]
        names = [battle.assets.factions[SIDES[i]['faction']]['name'] for i in range(2)]
        info = f'실제 {measured:.0f} fps · {names[0]} {counts[0]} : {counts[1]} {names[1]} · 최대 {battle.cap()}명/진영'
        screen.blit(self.small.render(info, True, (152, 170, 188)), (self.labels_x, self.top + 17))
        keys = 'Space 정지 · F 빨리 · R 재시작 · 1–5 증원 · [ ] FPS'
        screen.blit(self.small.render(keys, True, (120, 136, 152)), (self.width - self.small.size(keys)[0] - 16, self.top + 17))


def compose(screen, view, controls, battle, clock_ms, fps, measured):
    surf = view.draw(battle, clock_ms)
    screen.blit(view.pg.transform.scale(surf, (surf.get_width()*SCALE, surf.get_height()*SCALE)), (0, 0))
    controls.draw(screen, battle, fps, measured)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--shots', nargs='+', help='headless: OUT_PREFIX then seconds to capture')
    ap.add_argument('--seed', type=int, default=7)
    ap.add_argument('--wave', type=int, choices=WAVES, default=1, help='reinforcements per turn')
    ap.add_argument('--fps', type=int, choices=FPS_CHOICES, default=60)
    args = ap.parse_args()
    if args.shots:
        os.environ.setdefault('SDL_VIDEODRIVER', 'dummy')
    import pygame
    import assets as A
    pygame.init()
    font = FONT if os.path.exists(FONT) else None
    pygame.display.set_mode((1, 1))  # convert_alpha needs a display
    assets = A.Assets()
    view = View(assets, font)
    battle = Battle(assets, random.Random(args.seed), args.wave)
    size = (view.size[0]*SCALE, view.size[1]*SCALE + PANEL_H)
    controls = Controls(pygame, font, size[0], size[1] - PANEL_H)
    fps = args.fps
    if args.shots:
        screen = pygame.Surface(size)
        prefix, times = args.shots[0], sorted(float(t) for t in args.shots[1:])
        clock_ms, dt = 0.0, 1/fps
        for t in times:
            while clock_ms/1000 < t:
                battle.update(dt)
                clock_ms += dt*1000
            compose(screen, view, controls, battle, clock_ms, fps, fps)
            pygame.image.save(screen, f'{prefix}-{t:g}s.png')
        return
    screen = pygame.display.set_mode(size)
    pygame.display.set_caption('ver3 · 전투 데모')
    clock, clock_ms, paused, speed = pygame.time.Clock(), 0.0, False, 1
    while True:
        dt = clock.tick(fps)/1000
        for e in pygame.event.get():
            if e.type == pygame.QUIT or (e.type == pygame.KEYDOWN and e.key == pygame.K_ESCAPE):
                return
            if e.type == pygame.MOUSEBUTTONDOWN and e.button == 1:
                hit = controls.click(e.pos)
                if hit and hit[0] == 'wave':
                    battle.wave = hit[1]
                elif hit:
                    fps = hit[1]
            if e.type == pygame.KEYDOWN:
                if e.key == pygame.K_SPACE:
                    paused = not paused
                elif e.key == pygame.K_f:
                    speed = 3 if speed == 1 else 1
                elif e.key == pygame.K_r:
                    battle = Battle(assets, random.Random(), battle.wave)
                elif pygame.K_1 <= e.key <= pygame.K_5:
                    battle.wave = WAVES[e.key - pygame.K_1]
                elif e.key in (pygame.K_LEFTBRACKET, pygame.K_RIGHTBRACKET):
                    i = FPS_CHOICES.index(fps) + (1 if e.key == pygame.K_RIGHTBRACKET else -1)
                    fps = FPS_CHOICES[max(0, min(len(FPS_CHOICES) - 1, i))]
        if not paused:
            for _ in range(speed):
                battle.update(min(dt, .1))
                clock_ms += min(dt, .1)*1000
        compose(screen, view, controls, battle, clock_ms, fps, clock.get_fps())
        pygame.display.flip()


if __name__ == '__main__':
    main()
