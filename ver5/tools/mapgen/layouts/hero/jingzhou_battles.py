#!/usr/bin/env python3
"""Build the Xinye, Jiangling and Xiangyang battle maps from ver5."""

import math
import random

from southern_battles import write_map


def ground(width, height, tile='g--'):
    return [[tile] * width for _ in range(height)]


def paint(rows, x1, y1, x2, y2, tile):
    for y in range(y1, y2 + 1):
        rows[y][x1:x2 + 1] = [tile] * (x2 - x1 + 1)


def ring(rows, x1, y1, x2, y2, wall, floor, gates):
    paint(rows, x1, y1, x2, y2, wall)
    paint(rows, x1 + 1, y1 + 1, x2 - 1, y2 - 1, floor)
    for x, y, tile in gates:
        rows[y][x] = tile


def groves(rows, clusters, seed):
    rng = random.Random(seed)
    for cx, cy, radius in clusters:
        for y in range(max(1, cy-radius), min(len(rows)-1, cy+radius+1)):
            for x in range(max(1, cx-radius), min(len(rows[0])-1, cx+radius+1)):
                distance = math.hypot(x-cx, y-cy)
                if rows[y][x] == 'g--' and rng.random() < 0.68 * (1-distance/(radius+1)):
                    rows[y][x] = 'gt-'


def xinye():
    rows = ground(29, 25)
    for y, row in enumerate(rows):
        center = 30 - max(0, (y - 7) * 2 // 3) + round(1.5 * math.sin(y / 3))
        for x in range(len(row)):
            if abs(x - center) <= 1:
                row[x] = 'w--'
            elif abs(x - center) <= 3:
                row[x] = 's--'
            elif x < 3 and 5 <= y <= 17:
                row[x] = 'u--' if (x + y) % 3 else 'gt-'
            elif (x * 11 + y * 7) % 31 < 3:
                row[x] = 'gt-'

    seepage = {
        11: ((12,14),), 12: ((10,13), (17,18)), 13: ((10,15), (17,20)),
        14: ((11,16), (19,21)), 15: ((12,17), (20,22)), 16: ((12,15), (17,21)),
        17: ((11,14), (16,19)), 18: ((9,12), (15,17)), 19: ((10,11), (14,16)),
        20: ((15,17),),
    }
    shallows = {(12,12),(11,13),(12,13),(15,14),(16,15),(17,15),(13,16),
                (16,17),(17,17),(10,18),(11,18),(14,19),(17,20)}
    for y, spans in seepage.items():
        for left, right in spans:
            for x in range(left, right+1):
                if rows[y][x] in ('g--', 'gt-'):
                    rows[y][x] = 's--' if (x,y) in shallows else 'g-w'

    for y, start, end in ((0,0,3),(1,1,4),(2,3,6),(3,5,8),(4,7,10),
                          (5,9,13),(6,12,16),(7,15,19),(8,17,20),
                          (9,18,20),(10,19,20)):
        paint(rows, start, y, end, y, 'l--')
    for y in range(10, 24):
        x = 6 + (23 - y) // 3
        paint(rows, x, y, x + 1, y, 'l--')
    ring(rows,18,2,24,8,'c--','i--',[(18,6,'ip-'),(22,8,'ip-')])
    for x,y,tile in ((20,4,'ivF'),(22,4,'io-'),(23,6,'ivF'),(19,7,'i-e'),
                     (21,3,'ivF'),(24,7,'c-F'),(17,5,'dvF'),(16,8,'dvF'),
                     (22,9,'d-e'),(21,10,'dvF'),(16,10,'d-e'),(10,7,'d-r'),
                     (14,12,'d-r'),(7,18,'dv-'),(3,22,'dk-')):
        rows[y][x] = tile
    return rows


def camp(rows, x, y):
    ring(rows,x-2,y-2,x+2,y+2,'gz-','l--',
         [(x,y-2,'l--'),(x,y+2,'l--')])
    rows[y][x] = 'lk-'


def jiangling():
    rows = ground(29, 28)
    for y, row in enumerate(rows):
        left = 2 + round(math.sin(y / 3))
        right = 27 - round(math.sin((y - 9) / 3))
        for x in range(len(row)):
            if x <= left or x >= right:
                row[x] = 'w--'
            elif x == left + 1 or x == right - 1:
                row[x] = 's--'
            elif x in (9,10,16,17) and 5 < y < 20:
                row[x] = 's--' if (y + x) % 6 in (0, 1) else 'g-w'
        road = 13 - round(2 * math.sin((y - 8) / 5))
        paint(rows, road, y, road + 1, y, 'l--')

    paint(rows,3,11,4,13,'l--')
    paint(rows,22,11,24,13,'l--')
    paint(rows,25,11,25,13,'w--')
    paint(rows,4,16,24,16,'l--')
    paint(rows,12,14,13,20,'l--')
    for x,y in ((6,12),(13,12),(20,12)):
        camp(rows,x,y)
        paint(rows,x-1,9,x+1,9,'l--')
    ring(rows,8,19,19,26,'c--','i--',[(13,19,'ip-'),(14,19,'ip-')])
    paint(rows,12,20,14,25,'l--')
    for x,y,tile in ((13,23,'lo-'),(10,21,'iv-'),(16,21,'iv-'),(17,24,'iv-'),
                     (3,12,'dk-'),(24,12,'dk-'),(10,17,'d-r'),(18,17,'d-r')):
        rows[y][x] = tile
    groves(rows, ((6,4,4),(19,5,4),(23,8,3),(6,18,3),(23,22,4),(5,24,3)), 26)
    return rows


def xiangyang():
    rows = ground(35, 28)
    for y, row in enumerate(rows):
        bank = 29 + round(math.sin(y / 3.5))
        for x in range(len(row)):
            if y <= 2 + round(math.sin(x / 5)) or x >= bank:
                row[x] = 'w--'
            elif x == bank - 1:
                row[x] = 's--'
            elif x < 5 and (y < 7 or y > 19):
                row[x] = 'u--'
    paint(rows,16,0,19,8,'w--')
    paint(rows,18,4,31,6,'w--')
    paint(rows,15,6,15,8,'s--')
    paint(rows,20,7,24,7,'s--')
    paint(rows,7,13,14,13,'l--')
    paint(rows,17,16,18,25,'l--')
    for x,y in ((6,9),(6,17),(14,23),(21,23)):
        camp(rows,x,y)
    ring(rows,14,9,21,16,'c--','i--',[(17,9,'m-O'),(14,13,'m--'),(17,16,'m--')])
    for x,y,tile in ((17,8,'s--'),(17,10,'l--'),(17,11,'l--'),(17,12,'l--'),
                     (17,13,'io-'),(11,7,'d-r'),(10,20,'d-r')):
        rows[y][x] = tile
    groves(rows, ((6,5,3),(10,10,3),(9,19,3),(25,12,4),(25,19,4),(24,24,3)), 27)
    return rows


def main():
    write_map('map25', xinye(), '신야 화공 · 불타는 성과 범람한 백하')
    write_map('map26v', jiangling(), '강릉 수비 · 장강 나루와 외곽 진채')
    write_map('map27v', xiangyang(), '양양 공방 · 한수 굽이의 성벽과 수문')


if __name__ == '__main__':
    main()
