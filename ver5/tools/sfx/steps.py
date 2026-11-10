"""Cuts single footsteps (or hoof beats) out of a walking recording: each from just before its onset for LENGTH
seconds, faded out, and matched so its loudest 20 ms sits at LEVEL. Writes CUE-1.wav, CUE-2.wav, ... into OUT_DIR,
the variants the game picks one of per cell walked.

    python3 tools/sfx/steps.py IN.mp3 OUT_DIR CUE LEVEL LENGTH T1 T2 ... [--pairs N]

T1, T2 ... are the seconds of the beats' peaks (find them by ear or from the loudness peaks). With --pairs N it writes N
two-beat sounds instead (a hoof's 따각): two of the cut beats, picked at random, 0.085-0.1 s apart, the second at 70%."""
import subprocess
import sys
from pathlib import Path

import numpy as np
from scipy.io import wavfile
from scipy.signal import butter, sosfilt

SR = 44100


def main():
    src, out, cue, level, length = sys.argv[1], Path(sys.argv[2]), sys.argv[3], float(sys.argv[4]), float(sys.argv[5])
    args = sys.argv[6:]
    pairs = 0
    if '--pairs' in args:
        k = args.index('--pairs')
        pairs = int(args[k + 1])
        del args[k:k + 2]
    peaks = [float(a) for a in args]
    raw = subprocess.run(['ffmpeg', '-v', 'error', '-i', src, '-ac', '2', '-ar', str(SR), '-f', 's16le', '-'], capture_output=True, check=True).stdout
    x = np.frombuffer(raw, np.int16).reshape(-1, 2).astype(float) / 32768
    x = sosfilt(butter(2, 50 / (SR / 2), 'high', output='sos'), x, axis=0)
    mono = np.abs(x).max(axis=1)
    out.mkdir(parents=True, exist_ok=True)
    beats = []
    for p in peaks:
        c, back = int(p * SR), int(.05 * SR)
        seg = mono[c - back:c + int(.02 * SR)]
        onset = c - back + int(np.argmax(seg > seg.max() * .2)) - int(.004 * SR)
        y = x[onset:onset + int(length * SR)].copy()
        a, f = int(.003 * SR), int(.07 * SR)
        y[:a] *= np.linspace(0, 1, a)[:, None]
        y[-f:] *= np.linspace(1, 0, f)[:, None] ** 2
        w = int(.02 * SR)
        loud = max(np.sqrt(np.mean(y[j:j + w] ** 2)) for j in range(0, len(y) - w, w // 2))
        beats.append(y * level / loud)
    if pairs:
        rng = np.random.default_rng(7)
        made = []
        for _ in range(pairs):
            a, b = rng.choice(len(beats), 2, replace=False)
            gap = int(rng.uniform(.085, .1) * SR)
            y = np.zeros((gap + len(beats[b]), 2))
            y[:len(beats[a])] += beats[a]
            y[gap:] += beats[b] * .7
            made.append(y)
        beats = made
    for i, y in enumerate(beats, 1):
        wavfile.write(out / f'{cue}-{i}.wav', SR, (np.clip(y, -.89, .89) * 32767).astype(np.int16))


if __name__ == '__main__':
    main()
