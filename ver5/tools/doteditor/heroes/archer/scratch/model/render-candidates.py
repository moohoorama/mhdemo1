"""Render manually drawn candidate grids without changing any pixels."""
import sys
from pathlib import Path

from PIL import Image, ImageDraw

ROOT = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(ROOT / 'heroes' / 'tools'))
import dots
import render

doc = dots.load_working('archer')
files = sorted((ROOT / 'heroes' / 'archer' / 'candidates').glob('*/idle/SW-0.txt'))
sheet = Image.new('RGBA', (56 * 8 * len(files), 46 * 8 + 28), (30, 32, 34, 255))
draw = ImageDraw.Draw(sheet)
for i, path in enumerate(files):
    dots.apply_grid(doc, path)
    candidate = dots.frame(doc, 'idle', 'SW', 0)
    pic = render.on_bg(render.image(doc, candidate), 8, doc['unit']['pivot'])
    sheet.paste(pic, (i * 56 * 8, 28))
    draw.text((i * 56 * 8 + 8, 8), path.parents[1].name, fill=(235, 235, 235, 255))
sheet.save(ROOT / 'heroes' / 'archer' / 'preview' / 'candidates-SW.png')
