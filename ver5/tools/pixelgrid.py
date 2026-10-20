"""ASCII pixel grids for the hand-pixel sheets' seed scripts (tools/spritetool/assets/ver5-*/seed.py):
shapes are rasterised into a grid of palette letters ("." clear) that is then retouched by hand."""
import math


class Grid:
    def __init__(self, w, h):
        self.w, self.h = w, h
        self.rows = [["."] * w for _ in range(h)]

    def set(self, x, y, c):
        x, y = int(math.floor(x)), int(math.floor(y))
        if 0 <= x < self.w and 0 <= y < self.h:
            self.rows[y][x] = c

    def ellipse(self, cx, cy, rx, thick, c):
        """An iso ground ring (half as tall as wide), thick pixels across."""
        if rx <= 0:
            return
        for y in range(self.h):
            for x in range(self.w):
                d = math.hypot((x + .5 - cx) / rx, (y + .5 - cy) / (rx / 2)) * rx
                if abs(d - rx) <= thick / 2:
                    self.set(x, y, c)

    def trapezoid(self, cx, base, top, half0, half1, c):
        """Rows from base (exclusive) up to top, half width going from half0 to half1."""
        for y in range(int(top), int(base)):
            t = (base - y) / max(1, base - top)
            half = half0 + (half1 - half0) * t
            for x in range(self.w):
                if abs(x + .5 - cx) <= half:
                    self.set(x, y, c)

    def disc(self, x, y, r, c):
        for py in range(int(y - r) - 1, int(y + r) + 2):
            for px in range(int(x - r) - 1, int(x + r) + 2):
                if math.hypot(px + .5 - x, py + .5 - y) <= r:
                    self.set(px, py, c)

    def line(self, x0, y0, x1, y1, thick, c, square=False):
        """A stroke from (x0, y0) to (x1, y1): one pixel wide when thick <= 1, else round-capped, or
        drawn with a thick x thick square brush, which keeps narrow strokes free of bumps."""
        n = max(1, int(math.hypot(x1 - x0, y1 - y0) * 4))
        for j in range(n + 1):
            x, y = x0 + (x1 - x0) * j / n, y0 + (y1 - y0) * j / n
            if thick <= 1:
                self.set(x, y, c)
            elif square:
                k = int(round(thick))
                for dy in range(k):
                    for dx in range(k):
                        self.set(x - k / 2 + .5 + dx, y - k / 2 + .5 + dy, c)
            else:
                self.disc(x, y, thick / 2, c)

    def polygon(self, pts, c):
        """A filled polygon (even-odd, pixel centres)."""
        for y in range(self.h):
            for x in range(self.w):
                px, py, inside = x + .5, y + .5, False
                for (ax, ay), (bx, by) in zip(pts, pts[1:] + pts[:1]):
                    if (ay > py) != (by > py) and px < ax + (py - ay) * (bx - ax) / (by - ay):
                        inside = not inside
                if inside:
                    self.set(x, y, c)

    def oval(self, cx, cy, rx, ry, thick, c, fill=None):
        """An upright ellipse outline thick pixels across, optionally filled with fill."""
        for y in range(self.h):
            for x in range(self.w):
                d = math.hypot((x + .5 - cx) / rx, (y + .5 - cy) / ry)
                if fill and d < 1:
                    self.set(x, y, fill)
                if abs(d - 1) * min(rx, ry) <= thick / 2:
                    self.set(x, y, c)

    def lines(self):
        return ["".join(r) for r in self.rows]


def ks(n):
    return [(i + .5) / n for i in range(n)]
