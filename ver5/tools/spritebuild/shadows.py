"""Drop shadow of a sprite: its silhouette above the pivot (the feet), laid flat on the ground and sheared
down-right, away from the light at the screen's upper left. The silhouette is grown by GROW px and filled with
CORE alpha, inside a blurred halo of up to HALO alpha. Images are black with the shadow in alpha.
internal/sprite/render.go (makeShadow) is the runtime version; sprite_test.go holds it to this one. Needs Pillow.
"""
import math

SLOPE = (.6, .3)   # ground travel per pixel of height: right, down (2:1, along the map grid)
GROW = 1           # px added around the silhouette
CORE, HALO = 130, 110
BLUR = 3           # box blur radius, applied twice
PAD = GROW + 2*BLUR + 1


def shadow(sprite, pivot):
    """Shadow image of one RGBA sprite and the sprite's pivot inside it."""
    from PIL import Image, ImageChops, ImageFilter
    a, b = SLOPE
    w = sprite.width
    px, py = pivot
    silhouette = sprite.getchannel('A').point(lambda v: 255 if v else 0)
    depth = math.ceil(py*b)
    flat = Image.new('L', (w + math.ceil(py*a) + 2*PAD, depth + 1 + 2*PAD))
    for d in range(depth + 1):
        src = round(py - d/b)
        if src < 0:
            break
        x, y = PAD + round(d/b*a), PAD + d
        flat.paste(255, (x, y, x + w, y + 1), mask=silhouette.crop((0, src, w, src + 1)))
    grown = flat.filter(ImageFilter.MaxFilter(2*GROW + 1)) if GROW else flat
    halo = grown.filter(ImageFilter.BoxBlur(BLUR)).filter(ImageFilter.BoxBlur(BLUR))
    alpha = ImageChops.lighter(grown.point(lambda v: CORE if v else 0), halo.point(lambda v: v*HALO//255))
    black = Image.new('L', alpha.size)
    return Image.merge('RGBA', (black, black, black, alpha)), (px + PAD, PAD)


def unit_sheet(image, cell, pivot):
    """Shadow sheet with the same rows and columns as a unit image."""
    from PIL import Image
    w, h = cell
    cols, rows = image.width//w, image.height//h
    sheet = None
    for row in range(rows):
        for col in range(cols):
            cast, spivot = shadow(image.crop((col*w, row*h, (col + 1)*w, (row + 1)*h)), pivot)
            if sheet is None:
                sheet = Image.new('RGBA', (cast.width*cols, cast.height*rows))
            sheet.paste(cast, (col*cast.width, row*cast.height))
    return sheet, list(cast.size), list(spivot)
