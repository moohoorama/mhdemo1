#!/usr/bin/env python3
"""Validate and pack the configured native knight sheets with spritetool."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import struct
import subprocess
import tempfile

DEFAULT_ROOT = Path(__file__).resolve().parents[1]
PROJECT = Path('tools/knights/project.json')
NOTE = ('Rebuilt connected-body motion from full standing and seated references. '
        'Binary alpha and shared palette. Packed without resizing.')


def read_json(path):
    return json.loads(path.read_text())


def png_size(path):
    with path.open('rb') as source:
        header = source.read(24)
    if len(header) != 24 or header[:8] != b'\x89PNG\r\n\x1a\n' or header[12:16] != b'IHDR':
        raise ValueError(f'Not a PNG: {path}')
    return struct.unpack('>II', header[16:24])


def validate_manifest(manifest, source):
    """Reject missing/ambiguous frame mappings before running the Go tool."""
    cell = manifest['cell']
    if len(cell) != 2 or any(type(v) is not int or v <= 0 for v in cell):
        raise ValueError('cell must contain two positive integers')
    directions, animations = manifest['directions'], manifest['animations']
    if not directions or len(set(directions)) != len(directions):
        raise ValueError('directions must be nonempty and unique')
    if not animations or len(set(animations)) != len(animations):
        raise ValueError('animations must be nonempty and unique')
    frames = sorted(manifest['frames'], key=lambda frame: frame['index'])
    if not frames or [frame['index'] for frame in frames] != list(range(len(frames))):
        raise ValueError('frame indices must be unique and contiguous from zero')
    sizes = {direction: png_size(source / f'{direction}-pixel.png') for direction in directions}
    groups = {(d, a): [] for d in directions for a in animations}
    for frame in frames:
        key = (frame['direction'], frame['animation'])
        if key not in groups:
            raise ValueError(f'Unknown direction/animation: {key}')
        groups[key].append(frame['frame'])
        crop = frame['crop']
        if len(crop) != 4 or any(type(v) is not int for v in crop):
            raise ValueError(f'Invalid crop at frame {frame["index"]}')
        x, y, width, height = crop
        sheet_width, sheet_height = sizes[frame['direction']]
        if [width, height] != cell or min(x, y) < 0 or x + width > sheet_width or y + height > sheet_height:
            raise ValueError(f'Out-of-bounds or wrong-sized crop at frame {frame["index"]}')
    for (direction, animation), numbers in groups.items():
        if not numbers or sorted(numbers) != list(range(len(numbers))):
            raise ValueError(f'Missing or duplicate frames: {direction}/{animation}')
        durations = manifest.get('frame_durations_ms', {}).get(animation)
        if durations is not None and (len(durations) != len(numbers) or any(v <= 0 for v in durations)):
            raise ValueError(f'Invalid durations: {animation}')
    return frames


def build(root, manifest_path, output, project, check_only=False):
    manifest = read_json(manifest_path)
    frames = validate_manifest(manifest, manifest_path.parent)
    columns = project['columns']
    if type(columns) is not int or columns <= 0:
        raise ValueError('columns must be a positive integer')
    rows = math.ceil(len(frames) / columns)
    if check_only:
        print(f'Valid: {len(frames)} frames from {manifest_path}')
        return
    width, height = manifest['cell']
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='knight-build-') as directory:
        temporary = Path(directory)
        binary = temporary / 'spritetool'
        subprocess.run(['go', 'build', '-o', str(binary), './tools/spritetool'], cwd=root, check=True)
        packed = []
        for frame in frames:
            destination = temporary / f'{frame["index"]:03}.png'
            subprocess.run([str(binary), '--crop', ','.join(map(str, frame['crop'])),
                            str(manifest_path.parent / f'{frame["direction"]}-pixel.png'),
                            str(destination)], check=True)
            packed.append(str(destination))
        atlas = temporary / 'knights.png'
        subprocess.run([str(binary), '--atlas', f'{width}x{height},{columns}x{rows}',
                        *packed, str(atlas)], check=True)
        pixels = atlas.read_bytes()
        data = {**manifest, 'frames': frames, 'sheet': 'knights.png', 'columns': columns,
                'rows': rows, 'tile': project['tile'], 'note': NOTE,
                'asset_revision': hashlib.sha256(pixels).hexdigest()[:16]}
        (output / 'knights.png').write_bytes(pixels)
        (output / 'knights.json').write_text(json.dumps(data, indent=2) + '\n')
    print(f'Packed {len(frames)} native frames: {output / "knights.png"}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, default=DEFAULT_ROOT)
    parser.add_argument('--manifest', type=Path, help='Source manifest, relative to --root or absolute')
    parser.add_argument('--output', type=Path, help='Output directory, relative to --root or absolute')
    parser.add_argument('--check', action='store_true', help='Validate sources without generating files')
    args = parser.parse_args()
    root = args.root.resolve()
    try:
        project = read_json(root / PROJECT)
        manifest = root / (args.manifest or project['manifest'])
        output = root / (args.output or 'assets')
        build(root, manifest, output, project, args.check)
    except (ValueError, KeyError, OSError) as error:
        parser.error(str(error))


if __name__ == '__main__':
    main()
