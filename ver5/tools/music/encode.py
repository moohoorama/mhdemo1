"""Encode the themes the game plays into assets/music/themes as Ogg Vorbis, loudness-matched to -16 LUFS.

The masters stay in tools/music/sources (not shipped, kept in git even when no longer played). A theme is played
when music.json or a scene's "Music" names it; every other .ogg in assets/music/themes is removed so the APK
carries only what is heard. Loudness is matched with ffmpeg's two-pass loudnorm (linear gain, true peak -1.5 dB).

    python3 tools/music/encode.py [--bitrate 96k] [--force]
"""
import argparse
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SOURCES = Path(__file__).resolve().parent / 'sources'
THEMES = ROOT / 'assets/music/themes'
LOUDNESS = 'I=-16:TP=-1.5:LRA=11'


def measure(src):
    r = subprocess.run(['ffmpeg', '-hide_banner', '-nostats', '-i', str(src), '-vn',
                        '-af', f'loudnorm={LOUDNESS}:print_format=json', '-f', 'null', '-'],
                       capture_output=True, text=True, check=True)
    m = json.loads(re.search(r'\{[^{}]*"input_i"[^{}]*\}', r.stderr).group(0))
    return (f"loudnorm={LOUDNESS}:linear=true:measured_I={m['input_i']}:measured_TP={m['input_tp']}"
            f":measured_LRA={m['input_lra']}:measured_thresh={m['input_thresh']}:offset={m['target_offset']}")


def names(v):
    if isinstance(v, str):
        yield v
    elif isinstance(v, dict):
        for x in v.values():
            yield from names(x)


def scene_music(v):
    if isinstance(v, dict):
        for k, x in v.items():
            if k == 'Music' and isinstance(x, str):
                yield x
            else:
                yield from scene_music(x)
    elif isinstance(v, list):
        for x in v:
            yield from scene_music(x)


def played():
    out = set(names(json.loads((ROOT / 'assets/music/music.json').read_text())))
    for f in (ROOT / 'assets/scenes').glob('*.json'):
        out |= set(scene_music(json.loads(f.read_text())))
    return sorted(t for t in out if t)


def main():
    p = argparse.ArgumentParser()
    p.add_argument('--bitrate', default='96k')
    p.add_argument('--force', action='store_true', help='re-encode themes that are already there')
    a = p.parse_args()
    THEMES.mkdir(parents=True, exist_ok=True)
    want = played()
    for t in want:
        src = next(SOURCES.glob(f'{t}.*'), None)
        if src is None:
            raise SystemExit(f'no source for theme {t} in {SOURCES}')
        out = THEMES / f'{t}.ogg'
        if out.exists() and not a.force:
            continue
        subprocess.run(['ffmpeg', '-v', 'error', '-y', '-i', str(src), '-vn', '-map_metadata', '-1', '-af', measure(src),
                        '-ar', '44100', '-ac', '2', '-c:a', 'libvorbis', '-b:a', a.bitrate, str(out)], check=True)
        print(f'{t}: {src.stat().st_size // 1000}KB -> {out.stat().st_size // 1000}KB')
    for f in THEMES.iterdir():
        if f.stem not in want or f.suffix != '.ogg':
            f.unlink()
            print(f'removed {f.name}')


if __name__ == '__main__':
    main()
