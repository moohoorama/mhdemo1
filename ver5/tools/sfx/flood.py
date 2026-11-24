"""Recombines the game's own water sounds (assets/sounds/water.ogg, water-big.ogg and boat.ogg) into the sounds of the
suigong flood (internal/gui/uidemo_suigong.go): the flood's rush, the front breaking, the spray.

    python3 tools/sfx/flood.py [--out DIR] [--report]

The three sources are each a single swell (water, water-big: 0.4 s up, a plateau of 0.3-0.4 s at -15 to -19 dB, 1.2 s down;
boat: 0.3 s up, a steady wash of 1.7 s at -37 dB, ripple 3 dB, and 0.5 s down), so a cue played again and again
comes in pulses a second long. The flood is one continuous sound instead, built from only their steady parts:

  low wash   boat's wash (0.35-2.0 s), granulated: 0.45 s grains, 0.25 s equal-power crossfades, each grain from a
             random start of the steady part, resampled by up to +-9 %, every other one reversed, left and right
             drawn separately (boat's own channels correlate 0.27; the water cues' 0.97, which sounds like one
             point), bandpassed 50-1000 Hz.
  mid roar   water and water-big plateaus (0.3-0.72 s), 0.22 s grains with 0.1 s crossfades the same way,
             highpassed at 350 Hz.
  spray      water's and water-big's plateaus highpassed at 1.5 kHz.

An envelope in dB for each layer makes the approach: the low wash swells first, the mid roar and the spray
come up later (the water is heard nearer), the loudest 20 ms at the peak of the white. The cues' lengths and
where each one's peak lies are the offsets uidemo_suigong.go plays them at.
"""
import argparse
import subprocess
from pathlib import Path

import numpy as np
from scipy.signal import butter, sosfilt

SR = 44100
ROOT = Path(__file__).resolve().parents[2]
SOUNDS = ROOT / 'assets/sounds'


def decode(path):
    raw = subprocess.run(['ffmpeg', '-v', 'error', '-i', str(path), '-ac', '2', '-ar', str(SR), '-f', 's16le', '-'], capture_output=True, check=True).stdout
    return np.frombuffer(raw, np.int16).reshape(-1, 2).astype(float) / 32768


def encode(x, path):
    pcm = (np.clip(x, -.97, .97) * 32767).astype('<i2').tobytes()
    subprocess.run(['ffmpeg', '-v', 'error', '-y', '-f', 's16le', '-ar', str(SR), '-ac', '2', '-i', '-', '-c:a', 'libvorbis', '-q:a', '5', str(path)], input=pcm, check=True)


def band(x, lo, hi):
    if lo and hi:
        sos = butter(2, [lo / (SR / 2), hi / (SR / 2)], 'band', output='sos')
    elif lo:
        sos = butter(2, lo / (SR / 2), 'high', output='sos')
    else:
        sos = butter(2, hi / (SR / 2), 'low', output='sos')
    return sosfilt(sos, x, axis=0)


def granulate(pool, seconds, grain, fade, rng):
    """One channel of steady noise `seconds` long from random grains of the mono segments in pool."""
    n, g, f = int(seconds * SR), int(grain * SR), int(fade * SR)
    out = np.zeros(n + 3 * g)
    pos, prev = 0, -1
    win = np.sin(np.linspace(0, np.pi / 2, f)) ** 1  # equal power: sin up against cos down
    while pos < n + g:
        k = rng.integers(len(pool))
        while len(pool) > 1 and k == prev:
            k = rng.integers(len(pool))
        prev = k
        seg = pool[k]
        rate = 1 + rng.uniform(-.09, .09)
        m = int((g + f) * rate) + 2
        s0 = rng.integers(0, len(seg) - m)
        y = np.interp(np.arange(g + f) * rate, np.arange(m), seg[s0:s0 + m])
        if rng.random() < .5:
            y = y[::-1]
        y = y.copy()
        y[:f] *= win
        y[-f:] *= win[::-1]
        out[pos:pos + g + f] += y
        pos += g
    return out[:n]


def envelope(points, n):
    t = np.arange(n) / SR
    xs = [p[0] for p in points]
    ys = [p[1] for p in points]
    return 10 ** (np.interp(t, xs, ys) / 20)


def loud20(x):
    """RMS of the loudest 20 ms over both channels' power (the channels of the made cues are not alike, so their mean is no measure)."""
    pw = (x ** 2).mean(axis=1) if x.ndim == 2 else x ** 2
    w = int(.02 * SR)
    return np.sqrt(max(pw[i:i + w].mean() for i in range(0, len(pw) - w, w // 2)))


def stereo(pool, seconds, grain, fade, seed, lo, hi):
    rng = np.random.default_rng(seed)
    return np.stack([band(granulate(pool, seconds, grain, fade, rng), lo, hi) for _ in range(2)], axis=1)


def sources():
    boat, water, big = (decode(SOUNDS / f'{n}.ogg') for n in ('boat', 'water', 'water-big'))
    cut = lambda x, a, b: x[int(a * SR):int(b * SR)]
    wash = [cut(boat, .35, 2.0)[:, c] for c in (0, 1)]
    plateau = [cut(water, .3, .72)[:, c] for c in (0, 1)] + [cut(big, .3, .72)[:, c] for c in (0, 1)]
    return boat, water, big, wash, plateau


def make_surge(wash, plateau):
    """The flood's rush, 4.4 s: in under the burst's own sound within 0.35 s, swelling for 3.2 s to the peak of the white, dying away within 1 s after."""
    n = 4.4
    low = stereo(wash, n, .45, .25, 11, 50, 1000)
    mid = stereo(plateau, n, .22, .1, 12, 350, None)
    low /= loud20(low)
    mid /= loud20(mid)
    N = len(low)
    low *= envelope([(0, -70), (.15, -26), (.35, -14), (1.0, -10), (1.5, -9), (2.1, -8), (2.8, -6), (3.2, -3), (3.6, -3), (4.0, -16), (4.4, -60)], N)[:, None]
    mid *= envelope([(0, -80), (.15, -44), (.35, -28), (1.0, -18), (1.5, -15), (2.1, -12), (2.8, -8), (3.2, -4), (3.6, -4), (4.0, -20), (4.4, -70)], N)[:, None]
    x = low + mid
    return x / loud20(x) * .26


def make_crash(boat, big, wash, plateau):
    """The front breaking on the soldiers, 1.5 s: a quick wall of mid roar over a low thud."""
    n = 1.5
    mid = stereo(plateau, n, .22, .1, 22, 300, None)
    mid /= loud20(mid)
    thud = stereo(wash, n, .45, .25, 23, 40, 220)
    thud /= loud20(thud)
    N = len(mid)
    mid *= envelope([(0, -80), (.03, -30), (.12, -9), (.35, -10), (.9, -24), (1.5, -60)], N)[:, None]
    thud *= envelope([(0, -80), (.02, -20), (.1, -6), (.45, -12), (1.1, -30), (1.5, -70)], N)[:, None]
    x = mid + thud
    return x / loud20(x) * .19


def make_spray(plateau):
    """The spray flying, 2.4 s: the highs of the water's plateau, in at once and hissing out."""
    n = 2.4
    x = stereo(plateau, n, .22, .1, 31, 1500, None)
    x /= loud20(x)
    x *= envelope([(0, -80), (.04, -25), (.25, -8), (.8, -10), (1.6, -22), (2.4, -60)], len(x))[:, None]
    return x / loud20(x) * .085


def report(x, name):
    m = x.mean(axis=1)
    pw = (x ** 2).mean(axis=1)
    d = len(m) / SR
    w = int(.1 * SR)
    env = [10 * np.log10(pw[i:i + w].mean() + 1e-18) for i in range(0, len(pw) - w, w)]
    S = np.abs(np.fft.rfft(m)) ** 2
    f = np.fft.rfftfreq(len(m), 1 / SR)
    bands = [(0, 200), (200, 500), (500, 1000), (1000, 2000), (2000, 4000), (4000, 22050)]
    # repetition: the autocorrelation of the 10 ms level at lags of 0.15-2.0 s, after the swell is taken out
    h = int(.01 * SR)
    lv = np.log10(np.sqrt(np.convolve(pw, np.ones(h) / h, 'valid'))[::h] + 1e-6)
    lv = lv - np.convolve(lv, np.ones(31) / 31, 'same')
    ac = np.correlate(lv, lv, 'full')[len(lv) - 1:] / (lv @ lv)
    lag = ac[15:200]
    # seams: the largest 20 ms level jump (dB) between neighbours in the body of the sound (within 12 dB of its loudest)
    l20 = 10 * np.log10(np.convolve(pw, np.ones(h * 2) / (h * 2), 'valid')[::h * 2] + 1e-18)
    body = np.where(l20 > l20.max() - 12)[0]
    jump = np.abs(np.diff(l20[body[0]:body[-1] + 1])).max() if len(body) > 1 else 0
    print(f'{name}: {d:.2f} s, peak {np.abs(x).max():.3f}, loudest 20 ms {20 * np.log10(loud20(x)):.1f} dB, L-R corr {np.corrcoef(x[:, 0], x[:, 1])[0, 1]:.2f}')
    print('  100 ms level dB:', ' '.join(f'{e:.0f}' for e in env))
    print('  bands %:', ' '.join(f'{a}-{b}:{100 * S[(f >= a) & (f < b)].sum() / S.sum():.0f}' for a, b in bands),
          f'| centroid {(f * S).sum() / S.sum():.0f} Hz')
    print(f'  repetition: highest level autocorrelation at lag 0.15-2.0 s {lag.max():.2f} (at {(lag.argmax() + 15) / 100:.2f} s); largest 20 ms jump in the body {jump:.1f} dB')


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--out', default=str(SOUNDS))
    ap.add_argument('--report', action='store_true', help='only print the analysis of the three sources and of the made cues')
    a = ap.parse_args()
    boat, water, big, wash, plateau = sources()
    made = {'flood-surge': make_surge(wash, plateau), 'flood-crash': make_crash(boat, big, wash, plateau), 'flood-spray': make_spray(plateau)}
    for name, x in (('boat', boat), ('water', water), ('water-big', big)):
        report(x, name + ' (source)')
    for name, x in made.items():
        report(x, name)
        if not a.report:
            encode(x, Path(a.out) / f'{name}.ogg')


if __name__ == '__main__':
    main()
