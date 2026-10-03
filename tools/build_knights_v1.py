#!/usr/bin/env python3
"""Project recipe; image operations are performed by the generic spritetool."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile

p = argparse.ArgumentParser()
p.add_argument('--root', type=Path, default=Path(__file__).resolve().parents[1])
p.add_argument('--manifest', type=Path)
p.add_argument('--output', type=Path)
a = p.parse_args()
root = a.root.resolve()
source = root / 'tools/spritetool/assets'
manifest = json.loads((a.manifest or source / 'knight-frames.json').read_text())
output = a.output or root / 'assets'
output.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='knight-build-') as tmp:
    tmp = Path(tmp)
    binary = tmp / 'spritetool'
    subprocess.run(['go', 'build', '-o', str(binary), './tools/spritetool'], cwd=root, check=True)
    packed = []
    for frame in manifest['frames']:
        i = frame['index']
        raw, aligned = tmp / f'{i:03}-raw.png', tmp / f'{i:03}.png'
        subprocess.run([str(binary), '--crop', ','.join(map(str, frame['crop'])),
                        '--size', '34x34', str(source / (frame['direction'] + '-transparent.png')),
                        str(raw)], check=True)
        subprocess.run([str(binary), '--atlas', '40x48,1x1', '--atlas-pivot', '20,38',
                        '--sprite-pivot', ','.join(map(str, frame['source_pivot'])),
                        str(raw), str(aligned)], check=True)
        packed.append(str(aligned))
    subprocess.run([str(binary), '--atlas', '40x48,16x10', *packed,
                    str(output / 'knights.png')], check=True)
data = {**manifest, 'sheet': 'knights.png', 'columns': 16, 'rows': 10,
        'tile': [32, 16], 'scale': 34 / 280,
        'note': 'Rows interpreted as idle, walk, attack, guard, sit. Shared direction X and animation Y anchors preserve intra-animation motion.'}
(output / 'knights.json').write_text(json.dumps(data, indent=2) + '\n')
print(f'Built {len(packed)} frames: {output / "knights.png"}')
