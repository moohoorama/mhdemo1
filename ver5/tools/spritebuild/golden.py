#!/usr/bin/env python3
"""Regenerate internal/sprite/testdata/<id>.png with the dot editor's PNG exporter (ver4 apply.py, needs Pillow)."""
import importlib.util
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
spec = importlib.util.spec_from_file_location('apply', ROOT/'ver4/tools/doteditor/apply.py')
apply = importlib.util.module_from_spec(spec)
spec.loader.exec_module(apply)
out = HERE.parents[1]/'internal/sprite/testdata'
for p in sorted((HERE.parent/'doteditor/units').glob('*.yaml')):
    (out/'units').mkdir(parents=True, exist_ok=True)
    apply.render(p, out=out)

sys.path.insert(0, str(ROOT/'ver3'))
import shadows
from PIL import Image
sheet, *_ = shadows.unit_sheet(Image.open(out/'units/infantry.png').convert('RGBA'), (56, 46), (28, 38))
sheet.save(out/'units/infantry.shadow.png')
