"""Gives a silent capture its sound: the cues and themes the game asked for (an SRPG_SFX_TRACE file written by
cmd/_uidemo-vm, one "frame cue", "frame cue volume" or "frame music theme" per line, 60 frames a second) are laid on the video's
timeline at the volumes the game plays them (internal/gui/sound.go).

    python3 tools/sfx/dub.py TRACE VIDEO.mp4 OUT.mp4 [--sounds DIR ...] [--no-music] [--crop W:H:X:Y]

Each --sounds DIR is looked in before assets/sounds, so other versions of some cues can be heard in place.
"""
import argparse
import random
import re
import subprocess
from pathlib import Path

import numpy as np
from scipy.io import wavfile

SR = 44100
ROOT = Path(__file__).resolve().parents[2]
SFX_VOLUME, MUSIC_VOLUME = .6, .35
FADE_OUT, FADE_IN = .8, 1.5
GAP = 3  # frames: the same cue again within this is not doubled, as in the game


def decode(path):
    raw = subprocess.run(['ffmpeg', '-v', 'error', '-i', str(path), '-ac', '2', '-ar', str(SR), '-f', 's16le', '-'], capture_output=True, check=True).stdout
    return np.frombuffer(raw, np.int16).reshape(-1, 2).astype(float) / 32768


def variants(cue, dirs):
    for d in dirs:
        found = sorted(p for p in d.glob('*.ogg') if re.fullmatch(re.escape(cue) + r'(-\d+)?', p.stem))
        if found:
            return found
    return []


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('trace')
    ap.add_argument('video')
    ap.add_argument('out')
    ap.add_argument('--sounds', action='append', default=[])
    ap.add_argument('--no-music', action='store_true')
    ap.add_argument('--crop', help='w:h:x:y of the capture to keep, before scaling to 960 wide')
    a = ap.parse_args()
    dirs = [Path(d) for d in a.sounds] + [ROOT / 'assets/sounds']
    dur = float(subprocess.run(['ffprobe', '-v', 'error', '-show_entries', 'format=duration', '-of', 'csv=p=0', a.video], capture_output=True, text=True).stdout)
    n = int(dur * SR)
    sfx, music = np.zeros((n, 2)), np.zeros((n, 2))
    last, themes, cache = {}, [], {}
    rnd = random.Random(1)
    for line in Path(a.trace).read_text().splitlines():
        parts = line.split()
        frame = int(parts[0])
        if parts[1] == 'music':
            themes.append((frame, parts[2] if len(parts) > 2 else ''))
            continue
        cue = parts[1]
        vol = float(parts[2]) if len(parts) > 2 else 1.0
        if cue in last and frame - last[cue] < GAP:
            continue
        last[cue] = frame
        files = variants(cue, dirs)
        if not files:
            continue
        f = rnd.choice(files)
        x = cache.setdefault(f, decode(f))
        i = int(frame / 60 * SR)
        k = min(len(x), n - i)
        if k > 0:
            sfx[i:i + k] += x[:k] * SFX_VOLUME * vol
    if not a.no_music:
        for j, (frame, theme) in enumerate(themes):
            if not theme:
                continue
            start = int(frame / 60 * SR)
            stop = int(themes[j + 1][0] / 60 * SR) if j + 1 < len(themes) else n
            x = decode(ROOT / 'assets/music/themes' / f'{theme}.mp3')
            m = min(stop + int(FADE_OUT * SR), n) - start
            x = np.resize(x, (m, 2))
            t = np.arange(m) / SR
            env = np.minimum(t / FADE_IN, 1) * np.clip((stop / SR - start / SR + FADE_OUT - t) / FADE_OUT, 0, 1)
            music[start:start + m] += x * env[:, None] * MUSIC_VOLUME
    mix = sfx + music
    peak = np.abs(mix).max()
    if peak > .98:
        mix *= .98 / peak
    wav = Path(a.out).with_suffix('.wav')
    wavfile.write(wav, SR, (mix * 32767).astype(np.int16))
    subprocess.run(['ffmpeg', '-v', 'error', '-y', '-i', a.video, '-i', str(wav), '-map', '0:v', '-map', '1:a',
                    '-c:v', 'libx264', '-crf', '30', '-preset', 'slow', '-vf', (f'crop={a.crop},' if a.crop else '') + 'scale=960:-2', '-pix_fmt', 'yuv420p',
                    '-c:a', 'aac', '-b:a', '128k', '-shortest', '-movflags', '+faststart', a.out], check=True)
    wav.unlink()
    print(a.out)


if __name__ == '__main__':
    main()
