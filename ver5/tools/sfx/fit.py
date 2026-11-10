"""Fits a downloaded recording to the game's effects: cuts the silence before it and the noise floor after it, takes
out the rumble under 30 Hz, fades the end, and matches its loudest 20 ms to the cue's level. Writes a WAV for ffmpeg
to turn into the Vorbis file the game reads.

    python3 tools/sfx/fit.py IN.mp3 OUT.wav LEVEL [MAX_SECONDS] [LAYERS]

LAYERS > 1 lays copies 0.12 s apart (a bigger blaze from one flame).
"""
import subprocess
import sys

import numpy as np
from scipy.io import wavfile
from scipy.signal import butter, sosfilt

SR = 44100


def read(path):
    raw = subprocess.run(['ffmpeg', '-v', 'error', '-i', path, '-ac', '2', '-ar', str(SR), '-f', 's16le', '-'], capture_output=True, check=True).stdout
    return np.frombuffer(raw, np.int16).reshape(-1, 2).astype(float) / 32768


def envelope_db(x, win):
    mono = np.abs(x).max(axis=1)
    n = len(mono) // win
    rms = np.sqrt(np.mean(mono[:n * win].reshape(n, win) ** 2, axis=1))
    return 20 * np.log10(rms + 1e-9)


def main():
    src, dst, level = sys.argv[1], sys.argv[2], float(sys.argv[3])
    longest = float(sys.argv[4]) if len(sys.argv) > 4 else 0
    layers = int(sys.argv[5]) if len(sys.argv) > 5 else 1
    x = read(src)
    x = sosfilt(butter(2, 30 / (SR / 2), 'high', output='sos'), x, axis=0)
    win = int(.01 * SR)
    db = envelope_db(x, win)
    top = db.max()
    start = max(0, int(np.argmax(db > top - 40)) - 1) * win
    end = (len(db) - int(np.argmax(db[::-1] > top - 50))) * win
    x = x[start:end]
    if longest:
        x = x[:int(longest * SR)]
    if layers > 1:
        gap = int(.12 * SR)
        out = np.zeros((len(x) + gap * (layers - 1), 2))
        for i in range(layers):
            out[i * gap:i * gap + len(x)] += x * (1 - .15 * i)
        x = out
    f = min(len(x), int(.08 * SR))
    x[-f:] *= np.linspace(1, 0, f)[:, None]
    x[:int(.002 * SR)] *= np.linspace(0, 1, int(.002 * SR))[:, None]
    w = int(.02 * SR)
    loud = max(np.sqrt(np.mean(x[i:i + w] ** 2)) for i in range(0, max(1, len(x) - w), w // 2))
    x *= level / loud
    peak = np.abs(x).max()
    if peak > .89:
        x *= .89 / peak
    wavfile.write(dst, SR, (x * 32767).astype(np.int16))
    print(f'{dst}: {len(x) / SR:.2f} s, from {start / SR:.2f} s')


if __name__ == '__main__':
    main()
