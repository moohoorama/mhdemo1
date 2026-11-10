"""Brings recorded effects toward the music's tone: measures the music's long-term spectrum in 1/3-octave bands, then
pulls each effect's bands STRENGTH of the way toward it (each band by at most 9 dB, bands where the effect is silent
left alone) and keeps its loudest 20 ms where it was. Writes OUT_DIR/<name>.wav; run it once on the fitted recordings,
not on its own output.

    python3 tools/sfx/match.py MUSIC1.mp3,MUSIC2.mp3,... OUT_DIR STRENGTH EFFECT.ogg ...
"""
import subprocess
import sys
from pathlib import Path

import numpy as np
from scipy.io import wavfile

SR = 44100
EDGES = 1000 * 2 ** (np.arange(-14, 15) / 3)  # 1/3 octaves, about 40 Hz to 20 kHz
CENTRES = np.sqrt(EDGES[:-1] * EDGES[1:])


def read(path, seconds=None):
    cmd = ['ffmpeg', '-v', 'error', '-i', path] + (['-t', str(seconds)] if seconds else []) + ['-ac', '2', '-ar', str(SR), '-f', 's16le', '-']
    return np.frombuffer(subprocess.run(cmd, capture_output=True, check=True).stdout, np.int16).reshape(-1, 2).astype(float) / 32768


def bands(x):
    """Band energies in dB under the loudest band."""
    m, n = x.mean(axis=1), 4096
    power = np.mean([np.abs(np.fft.rfft(m[i:i + n] * np.hanning(n))) ** 2 for i in range(0, max(1, len(m) - n), n // 2)], axis=0)
    f = np.fft.rfftfreq(n, 1 / SR)
    db = 10 * np.log10([power[(f >= lo) & (f < hi)].sum() + 1e-12 for lo, hi in zip(EDGES[:-1], EDGES[1:])])
    return db - db.max()


def loudest(x):
    w = SR // 50
    return max(np.sqrt(np.mean(x[i:i + w] ** 2)) for i in range(0, max(1, len(x) - w), w // 2))


def main():
    music, out, strength = sys.argv[1].split(','), Path(sys.argv[2]), float(sys.argv[3])
    out.mkdir(parents=True, exist_ok=True)
    target = np.mean([bands(read(p, 90)) for p in music], axis=0)
    for path in sys.argv[4:]:
        x = read(path)
        padded = np.vstack([x, np.zeros((max(0, 4096 - len(x)), 2))])
        have = bands(padded)
        gain = np.clip(strength * (target - have), -9, 9)
        gain[have < -50] = 0
        gain = np.convolve(np.pad(gain, 1, mode='edge'), np.ones(3) / 3, mode='valid')
        f = np.fft.rfftfreq(len(padded), 1 / SR)
        curve = 10 ** (np.interp(np.log(np.maximum(f, 20)), np.log(CENTRES), gain) / 20)
        y = np.fft.irfft(np.fft.rfft(padded, axis=0) * curve[:, None], n=len(padded), axis=0)[:len(x)]
        y = np.clip(y * loudest(x) / loudest(y), -.89, .89)
        wavfile.write(out / (Path(path).stem + '.wav'), SR, (y * 32767).astype(np.int16))
        print(Path(path).stem)


if __name__ == '__main__':
    main()
