"""Synthesised sound effects for the GUI: the drafts of each cue, written as WAV files.

    python3 tools/sfx/synth.py OUT_DIR [cue ...]

Everything is built from noise, sines and simple physical models (modal bells and wood, koto, taiko, pan flute,
additive brass, piano and strings, formant voices) and fitted to the music (see `finish`): the same five notes,
the same warmth, the same room. Each draft is loudness-matched (loudest 20 ms RMS) and kept under -1 dBFS, and
seeded by its name, so a run makes the same sounds again.
"""
import sys
import zlib
from pathlib import Path

import numpy as np
from scipy.io import wavfile
from scipy.signal import butter, sosfilt, stft, istft

SR = 44100
rng = np.random.default_rng(11)


def t_axis(dur):
    return np.arange(int(dur * SR)) / SR


def env_ad(n, attack, decay):
    t = np.arange(n) / SR
    return np.minimum(t / max(attack, 1e-4), 1) * np.exp(-np.maximum(t - attack, 0) / decay)


def fade(x, out=.02, inn=.002):
    n = len(x)
    a, b = int(inn * SR), int(out * SR)
    if a:
        x[:a] *= np.linspace(0, 1, a)
    if b:
        x[n - b:] *= np.linspace(1, 0, b)
    return x


def band(x, lo, hi, order=2):
    sos = butter(order, [lo / (SR / 2), min(hi, SR / 2 - 100) / (SR / 2)], 'band', output='sos')
    return sosfilt(sos, x)


def lowpass(x, f, order=2):
    return sosfilt(butter(order, f / (SR / 2), 'low', output='sos'), x)


def highpass(x, f, order=2):
    return sosfilt(butter(order, f / (SR / 2), 'high', output='sos'), x)


def glide_noise(dur, f0, f1, width, curve=1.0):
    """Noise whose pass band (a log-gaussian of the given width) glides from f0 to f1."""
    n = int(dur * SR)
    x = rng.standard_normal(n + 2048)
    f, t, Z = stft(x, SR, nperseg=1024, noverlap=896)
    k = np.clip(t / dur, 0, 1)
    fc = f0 * (f1 / f0) ** (k ** curve)
    mask = np.exp(-0.5 * (np.log((f[:, None] + 1) / fc[None, :]) / width) ** 2)
    _, y = istft(Z * mask, SR, nperseg=1024, noverlap=896)
    return y[:n]


def place(total, parts):
    out = np.zeros(int(total * SR))
    for at, p in parts:
        i = int(at * SR)
        if i >= len(out):
            continue
        m = min(len(p), len(out) - i)
        out[i:i + m] += p[:m]
    return out


def note(name):
    names = {'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
    semi = names[name[0]]
    rest = name[1:]
    if rest.startswith('#'):
        semi, rest = semi + 1, rest[1:]
    elif rest.startswith('b'):
        semi, rest = semi - 1, rest[1:]
    return 440 * 2 ** ((semi + 12 * (int(rest) + 1) - 69) / 12)


# ---- instruments -------------------------------------------------------------------------------------------

def modal(freqs, amps, decays, dur):
    t = t_axis(dur)
    return sum(a * np.sin(2 * np.pi * f * t + rng.uniform(0, 6.28)) * np.exp(-t / d) for f, a, d in zip(freqs, amps, decays))


def bell(f0, dur=1.6, bright=1.0):
    ratios = [.5, 1, 1.183, 1.506, 2, 2.514, 2.662, 3.011, 4.166]
    amps = [.35, 1, .55, .45, .5, .25, .2, .15, .1]
    decays = [1.6, 1.1, .8, .6, .5, .3, .28, .2, .12]
    x = modal([f0 * r for r in ratios], [a * (bright if i > 3 else 1) for i, a in enumerate(amps)], [d * dur / 1.6 for d in decays], dur)
    return x * np.minimum(t_axis(dur) / .002, 1)


def wood(f0, dur=.12, ring=.04):
    ratios, amps = [1, 2.57, 4.2, 6.1], [1, .6, .35, .2]
    x = modal([f0 * r for r in ratios], amps, [ring, ring * .6, ring * .4, ring * .25], dur)
    click = highpass(rng.standard_normal(int(.004 * SR)), 2500) * .6
    x[:len(click)] += click * np.linspace(1, 0, len(click))
    return x


def brass(f0, dur, loud=1.0, attack=.035, scoop=.03, vib=5.5, detune=0.0):
    """Additive brass: brightness follows loudness, a small pitch scoop into the note, a late vibrato."""
    t = t_axis(dur)
    env = np.minimum(t / attack, 1) * np.where(t > dur - .06, np.clip((dur - t) / .06, 0, 1), 1)
    env *= 1 - .15 * np.minimum(t / (dur + .01), 1)
    cents = -scoop * 1200 * np.exp(-t / .03) + 8 * np.sin(2 * np.pi * vib * t) * np.clip((t - .15) / .2, 0, 1) + detune
    f = f0 * 2 ** (cents / 1200)
    ph = 2 * np.pi * np.cumsum(f) / SR
    tilt = 2.4 - 1.5 * env * loud
    x = np.zeros_like(t)
    for k in range(1, 16):
        if f0 * k > 12000:
            break
        x += k ** (-tilt) * np.sin(k * ph)
    breath = band(rng.standard_normal(len(t)), 1500, 6000) * .02 * np.exp(-t / .05)
    return (x + breath) * env * loud


def brass_section(freqs, dur, loud=1.0, voices=3):
    return sum(brass(f, dur, loud, detune=rng.uniform(-9, 9), scoop=rng.uniform(.02, .045)) for f in freqs for _ in range(voices)) / voices


def piano(f0, dur=.9, vel=1.0):
    t = t_axis(dur)
    B = 1.5e-4
    x = np.zeros_like(t)
    for k in range(1, 14):
        fk = k * f0 * np.sqrt(1 + B * k * k)
        if fk > 14000:
            break
        a = vel ** (1 + k * .08) / k ** 1.1
        d = .9 / (1 + .35 * k) * (440 / f0) ** .3
        x += a * (np.sin(2 * np.pi * fk * t) + .3 * np.sin(2 * np.pi * fk * 1.0007 * t)) * np.exp(-t / d)
    hammer = band(rng.standard_normal(int(.01 * SR)), 800, 5000) * .15 * vel
    x[:len(hammer)] += hammer
    return x * np.minimum(t / .002, 1)


def strings(f0, dur, attack=.12, vib=5.0, voices=3):
    """A soft string section: saw-like partials, slow bow attack, vibrato, gentle lowpass."""
    t = t_axis(dur)
    env = np.minimum(t / attack, 1) * np.clip((dur - t) / .25, 0, 1)
    out = np.zeros_like(t)
    for v in range(voices):
        cents = rng.uniform(-7, 7) + 12 * np.sin(2 * np.pi * (vib + rng.uniform(-.4, .4)) * t + rng.uniform(0, 6)) * np.clip(t / .25, 0, 1)
        ph = 2 * np.pi * np.cumsum(f0 * 2 ** (cents / 1200)) / SR
        for k in range(1, 12):
            if f0 * k > 9000:
                break
            out += (1 / k) * np.sin(k * ph + rng.uniform(0, 6))
    return lowpass(out / voices, 3200) * env


def voice(dur, f0, vowels, rate, level):
    """One murmuring voice: a glottal pulse train through formants that change every syllable."""
    form = {'a': (800, 1200, 2500), 'e': (500, 1900, 2500), 'i': (320, 2300, 3000), 'o': (500, 900, 2400), 'u': (350, 800, 2300), 'eo': (600, 1000, 2500)}
    n = int(dur * SR)
    out = np.zeros(n)
    t0 = rng.uniform(0, .15)
    while t0 < dur:
        syl = rng.uniform(.6, 1.4) / rate
        m = int(syl * SR)
        tt = np.arange(m) / SR
        f = f0 * (1 + .12 * np.sin(np.pi * tt / syl) + rng.uniform(-.08, .08))
        ph = np.cumsum(f) / SR
        src = ((ph % 1) < .35).astype(float) - .35
        if rng.random() < .5:  # a consonant
            src[:int(.03 * SR)] += band(rng.standard_normal(int(.03 * SR)), 2000, 7000) * 2
        vw = form[rng.choice(list(form))]
        y = np.zeros(m)
        for fc, g in zip(vw, (1, .6, .3)):
            y += g * band(src, fc * .85, fc * 1.15)
        e = np.sin(np.pi * np.clip(tt / syl, 0, 1)) ** 1.5
        i = int(t0 * SR)
        k = min(m, n - i)
        out[i:i + k] += (y * e)[:k]
        t0 += syl * rng.uniform(1, 1.3) + (rng.uniform(.05, .25) if rng.random() < .2 else 0)
    return out * level


def crackles(dur, rate, lo=2500, hi=9000, amp=.6):
    out = np.zeros(int(dur * SR))
    t = 0
    while True:
        t += rng.exponential(1 / rate)
        if t >= dur:
            break
        m = int(rng.uniform(.002, .008) * SR)
        c = band(rng.standard_normal(m + 64), lo, hi)[:m] * np.exp(-np.arange(m) / (m / 3))
        i = int(t * SR)
        k = min(m, len(out) - i)
        out[i:i + k] += c[:k] * amp * rng.uniform(.3, 1)
    return out


def roar(dur, lo=150, hi=1400, attack=.12, flicker=14):
    t = t_axis(dur)
    x = band(rng.standard_normal(len(t)), lo, hi)
    am = 1 + .5 * lowpass(rng.standard_normal(len(t)), flicker) * 25
    e = np.minimum(t / attack, 1) * np.exp(-np.maximum(t - attack, 0) / (dur * .45))
    return x * np.clip(am, .2, 2) * e


def rock_hit(f0, amp, dur=.09):
    ratios = [1, 1.7, 2.6, 3.9, 5.3]
    x = modal([f0 * r * rng.uniform(.95, 1.05) for r in ratios], [1, .7, .5, .35, .2], [.025, .018, .012, .008, .006], dur)
    click = band(rng.standard_normal(int(.003 * SR)), 1500, 8000)
    x[:len(click)] += click * .8
    return x * amp


def thump(f0, f1, dur, decay):
    t = t_axis(dur)
    f = f1 + (f0 - f1) * np.exp(-t / (decay * .4))
    return np.sin(2 * np.pi * np.cumsum(f) / SR) * np.exp(-t / decay) * np.minimum(t / .002, 1)


def koto(f0, dur=1.2, bright=.5):
    """A plucked koto string (Karplus-Strong): a noise burst in a delay line that loses its highs each pass."""
    n, period = int(dur * SR), int(SR / f0)
    buf = lowpass(rng.uniform(-1, 1, period), 2000 + 6000 * bright)
    out = np.empty(n)
    for i in range(n):
        j = i % period
        out[i] = buf[j]
        buf[j] = .5 * (buf[j] + buf[(j + 1) % period]) * .996
    return out * np.minimum(t_axis(dur) / .001, 1)


def taiko(f0, dur=.5, decay=.18, stick=.4):
    """A big drum: membrane modes with a pitch drop as the skin settles, and a soft stick."""
    t = t_axis(dur)
    drop = 1 + .25 * np.exp(-t / .02)
    out = np.zeros_like(t)
    for r, a, d in zip([1, 1.59, 2.14, 2.3, 2.65], [1, .5, .3, .25, .15], [1, .6, .4, .35, .25]):
        out += a * np.sin(2 * np.pi * np.cumsum(f0 * r * drop) / SR) * np.exp(-t / (decay * d))
    hit = lowpass(rng.standard_normal(int(.012 * SR)), 1800) * stick
    out[:len(hit)] += hit * np.linspace(1, 0, len(hit))
    return out * np.minimum(t / .001, 1)


def flute(f0, dur, attack=.08, glide=1.0):
    """A pan flute: a near-sine tone with breath, sliding from f0 to f0*glide."""
    t = t_axis(dur)
    f = f0 * glide ** (t / dur) * (1 + .004 * np.sin(2 * np.pi * 5 * t))
    ph = 2 * np.pi * np.cumsum(f) / SR
    tone = np.sin(ph) + .15 * np.sin(2 * ph) + .05 * np.sin(3 * ph)
    breath = band(rng.standard_normal(len(t)), f0 * .8, f0 * 3) * .25
    env = np.minimum(t / attack, 1) * np.clip((dur - t) / .12, 0, 1)
    return (tone + breath) * env


def gravel(dur, rate, amp, lo=400, hi=4000):
    """Untuned grit: short noise ticks, no ringing partials (a tuned rock_hit reads as a xylophone)."""
    out = np.zeros(int(dur * SR))
    t = 0
    while True:
        t += rng.exponential(1 / rate)
        if t >= dur:
            break
        m = int(rng.uniform(.004, .015) * SR)
        g = band(rng.standard_normal(m + 64), lo, hi)[:m] * np.exp(-np.arange(m) / (m / 4))
        i = int(t * SR)
        k = min(m, len(out) - i)
        out[i:i + k] += g[:k] * amp * rng.uniform(.3, 1) * np.exp(-t / (dur * .3))
    return out


# ---- the shared finish ---------------------------------------------------------------------------------------
# The music uses only C D E G A, is warm (99% of its energy under 1.2-4 kHz) and sits at about -21 dB RMS. Every
# effect keeps to the same five notes, loses its highs above `bright` and goes through the same room.

ROOM = None


def room():
    global ROOM
    if ROOM is None:
        r = np.random.default_rng(1)
        n = int(1.1 * SR)
        t = np.arange(n) / SR
        ir = r.standard_normal(n) * np.exp(-t / .16)  # about 1.1 s to -60 dB
        ir = lowpass(ir, 3000)
        ir[:int(.012 * SR)] = 0  # a little pre-delay
        ROOM = ir / np.sqrt(np.sum(ir ** 2))
    return ROOM


def finish(x, bright=4500, wet=.12):
    x = lowpass(lowpass(x, bright), bright * 1.6)
    if wet:
        w = np.convolve(x, room())
        x = np.pad(x, (0, len(w) - len(x))) + wet * w
        x = x[:len(x) - np.argmax(np.abs(x[::-1]) > 1e-3 * np.max(np.abs(x)))]  # drop the tail below -60 dB
    return x


# ---- cues --------------------------------------------------------------------------------------------------
# Each cue maps its drafts to (sound, bright, wet). Notes stay in C D E G A.

def swing(v):  # 슈욱: a heavy, low whoosh
    x = glide_noise(.28, 3200, 900, .4) + .5 * glide_noise(.28, 800, 300, .5)
    return x * env_ad(len(x), .07, .1), 3500, .06


def shoot(v):  # an arrow: a whoosh moving away
    x = glide_noise(.46, 4000, 800, .45, curve=.7)
    t = t_axis(.46)
    return x * np.minimum(t / .08, 1) * np.exp(-np.maximum(t - .08, 0) / .16), 3800, .06


def hit(v):  # 퍽
    if v == 'A':  # a drum-like body and a cloth slap
        body = taiko(95, .25, .06, .2)
        slap = band(rng.standard_normal(int(.08 * SR)), 250, 1800) * env_ad(int(.08 * SR), .0008, .012) * 1.4
        return place(.25, [(0, body), (0, slap)]), 3000, .05
    if v == 'B':  # leather and flesh: a short low thud, no ring
        body = thump(130, 50, .18, .045) * 1.2
        slap = band(rng.standard_normal(int(.1 * SR)), 180, 1200) * env_ad(int(.1 * SR), .001, .02) * 1.3
        crunch = gravel(.06, 300, .25, 600, 2500)
        return place(.18, [(0, body), (0, slap), (.002, crunch)]), 2800, .05
    body = taiko(75, .3, .09, .35) * 1.1  # heavier: a drum and a grunt of grit
    slap = lowpass(rng.standard_normal(int(.05 * SR)), 1400) * env_ad(int(.05 * SR), .0005, .01)
    return place(.3, [(0, body), (0, slap), (.004, gravel(.08, 200, .3, 500, 2000))]), 2600, .06


def crit(v):  # a heavier 퍽 with a crack
    body = taiko(70, .45, .12, .5) * 1.3
    crack = band(rng.standard_normal(int(.03 * SR)), 900, 4000) * env_ad(int(.03 * SR), .0005, .006) * 1.5
    slap = band(rng.standard_normal(int(.1 * SR)), 200, 1500) * env_ad(int(.1 * SR), .001, .02) * 1.2
    return place(.45, [(0, body), (0, crack), (0, slap)]), 3500, .08


def block(v):  # the blow taken on a shield: a dull thud
    body = thump(120, 55, .2, .07)
    noise = band(rng.standard_normal(int(.2 * SR)), 200, 1100) * env_ad(int(.2 * SR), .002, .035) * 1.2
    return body + noise, 3000, .06


def fall(v):  # a body going down: a low thud, cloth, a rattle of armour
    d = .7
    body = thump(85, 38, .4, .12) * 1.2
    cloth = band(rng.standard_normal(int(.25 * SR)), 200, 1500) * env_ad(int(.25 * SR), .01, .06)
    rattle = gravel(.35, 120, .35, 900, 3500)
    return place(d, [(0, cloth), (.05, body), (.06, rattle)]), 3200, .1


def duel(v):  # blades meet: a muted clang
    clang = modal([1150, 1700, 2420, 3300, 4100], [1, .7, .5, .3, .2], [.12, .09, .07, .05, .03], .4)
    scrape = band(rng.standard_normal(int(.12 * SR)), 1500, 4000) * env_ad(int(.12 * SR), .001, .04) * .5
    return place(.4, [(0, clang), (0, scrape)]), 4200, .15


def turn(v):  # 따라란~: bells, E G C rising
    if v == 'A':
        x = place(1.8, [(0, bell(note('E5'), 1.2, .5)), (.12, bell(note('G5'), 1.2, .5)), (.24, bell(note('C6'), 1.6, .6))])
    else:  # a koto plucks the same notes under softer bells
        x = place(1.8, [(i * .12, koto(note(n), 1.4) * .9 + np.pad(bell(note(n), 1.2, .3) * .4, (0, int(.2 * SR) + 1))[:int(1.4 * SR)]) for i, n in enumerate(['E5', 'G5', 'C6'])])
    return x, 4000, .22


def levelup(v):  # 따다다단: C C C G
    ns = [('C5', .08), ('C5', .08), ('C5', .08), ('G5', .55)]
    parts, at = [], 0
    for i, (n, d) in enumerate(ns):
        last = i == len(ns) - 1
        f = note(n)
        p = brass_section([f], d + (0 if last else .02), 1.0 if last else .8, voices=2)
        if v == 'B':  # a horn an octave below
            p = p + .6 * brass_section([f / 2], d + (0 if last else .02), .8, voices=1)
        parts.append((at, p))
        at += d + .025
    return place(at + .1, parts), 4000, .18


def promote(v):  # 빰빠바밤빠빰: a section in chords from C D E G A
    C = ['C4', 'E4', 'G4', 'C5']
    Am = ['A3', 'C4', 'E4', 'A4']
    Gs = ['G3', 'D4', 'G4', 'A4']
    C6 = ['C5', 'E5', 'G5', 'A5', 'C6']
    seq = [(C, .22), (C, .1), (C, .1), (Am, .26), (Gs, .12), (C6, .85)]
    parts, at = [], 0
    for i, (ch, d) in enumerate(seq):
        last = i == len(seq) - 1
        parts.append((at, brass_section([note(n) for n in ch], d + .02, 1.0 if (last or d > .15) else .75, voices=2)))
        if v == 'B' and (d > .15 or last):
            parts.append((at, taiko(70, .6, .2, .3) * 1.5))
        at += d + .03
    return place(at + .2, parts), 4000, .2


def ui_select(v):  # 딸깍
    return place(.06, [(0, wood(1800, .03, .008) * .6), (.022, wood(2600, .03, .006))]), 6000, 0


def ui_back(v):  # a lower, softer click
    return place(.06, [(0, wood(1300, .03, .008) * .6), (.02, wood(1000, .03, .008))]), 5000, 0


def goals(v):  # 슈우웅, falling
    if v == 'A':  # a breath that sinks with a tone under it
        x = glide_noise(.5, 2500, 500, .4) * env_ad(int(.5 * SR), .05, .18)
        return x + .5 * flute(note('E5'), .5, .05, note('A4') / note('E5')), 3500, .15
    return place(.6, [(0, flute(note('E5'), .25, .03)), (.18, flute(note('A4'), .4, .03))]), 3500, .2  # two flute notes down


def coins(v):  # small bells: G6 E6
    return place(.5, [(0, bell(note('G6'), .4, .4) * .7), (.07, bell(note('E6'), .45, .4))]), 5000, .1


def arrival(v):  # war drums: 둥 둥 둥
    return place(1.0, [(0, taiko(65, .6, .22)), (.22, taiko(65, .6, .22) * .8), (.44, taiko(58, .7, .28) * 1.1)]), 2500, .2


def cast(v):  # a koto runs up the pentatonic over a breath
    ns = ['C4', 'D4', 'E4', 'G4', 'A4', 'C5', 'D5', 'E5']
    breath = glide_noise(.7, 600, 2000, .5) * env_ad(int(.7 * SR), .3, .2) * .3
    return place(1.0, [(i * .035, koto(note(n), .8) * .5) for i, n in enumerate(ns)] + [(0, breath)]), 4000, .2


def fire(v):  # 화르륵
    d = .9
    x = roar(d, 150, 1500, .1) + crackles(d, 40, 1500, 5000) + .5 * glide_noise(d, 400, 2000, .5)
    return x * env_ad(int(d * SR), .08, .4), 4000, .1


def fire_big(v):  # three flames over a rumble
    d = 1.6
    parts = [(i * .12, fire('A')[0] * rng.uniform(.7, 1)) for i in range(3)]
    rumble = lowpass(rng.standard_normal(int(d * SR)), 120) * env_ad(int(d * SR), .2, .6) * 4
    return place(d, parts) + rumble + crackles(d, 90, 1500, 5000, .7) * env_ad(int(d * SR), .2, .7), 4000, .12


def earth(v):  # 때굴때굴: two big stones
    d = 1.1
    parts = []
    for r in range(2):
        at, gap, f0, amp = rng.uniform(0, .15), rng.uniform(.06, .09), rng.uniform(350, 770), 1
        while at < d - .1 and amp > .08:
            parts.append((at, rock_hit(f0 * rng.uniform(.9, 1.1), amp)))
            at += gap * rng.uniform(.7, 1.3)
            gap *= .93
            amp *= .86
    rumble = lowpass(rng.standard_normal(int(d * SR)), 200) * env_ad(int(d * SR), .1, .5) * 1.5
    return place(d, parts) + rumble, 3500, .1


def boulder(v):  # 빡: a crack and a low blow, grit and dust after
    d = 1.0
    crack = highpass(rng.standard_normal(int(.03 * SR)), 1200) * env_ad(int(.03 * SR), .0005, .008) * 2.5
    body = thump(90, 40, .5, .14) * 2
    crunch = band(rng.standard_normal(int(.25 * SR)), 300, 2500) * env_ad(int(.25 * SR), .001, .05) * 1.2
    dust = lowpass(rng.standard_normal(int(d * SR)), 600) * env_ad(int(d * SR), .02, .25) * .8
    return place(d, [(0, crack), (0, body), (0, crunch), (0, dust), (.03, gravel(.7, 60, .5))]), 4000, .1


def water(v):  # wind and a wave breaking quickly
    d, peak = 1.4, .3
    t = t_axis(d)
    e = np.where(t < peak, (t / peak) ** 2, np.exp(-(t - peak) / (d * .3)))
    f, tt, Z = stft(rng.standard_normal(len(t) + 2048), SR, nperseg=1024, noverlap=896)
    cut = 300 + 3000 * np.interp(tt, t, e)
    _, wave = istft(Z / (1 + (f[:, None] / cut[None, :]) ** 4), SR, nperseg=1024, noverlap=896)
    wind = glide_noise(d, 700, 1300, .2, curve=.6) * (.4 + .6 * np.sin(np.pi * t / d)) * .5
    return wave[:len(t)] * e * 1.4 + wind, 3500, .1


def heal(v):  # 따단다단~: soft strings
    ns = ['C4', 'E4', 'G4', 'C5'] if v == 'A' else ['A3', 'C4', 'E4', 'A4']
    parts = [(i * .13, strings(note(n), 1.3 - i * .13, .08)) for i, n in enumerate(ns)]
    pad = strings(note(ns[0]) / 2, 1.3, .3) * .5
    return place(1.4, parts + [(0, pad)]), 3500, .25


def scheme(v):  # 웅성웅성: fourteen murmuring voices
    d = 1.4
    out = np.zeros(int(d * SR))
    for _ in range(14):
        near = rng.uniform(.3, 1)
        out += lowpass(voice(d, rng.uniform(95, 230), None, rng.uniform(4, 6.5), near), 1500 + 2000 * near)
    t = t_axis(d)
    return out * np.minimum(t / .25, 1) * np.clip((d - t) / .3, 0, 1), 3000, .2


def buff(v):  # piano, low to high: C E G
    notes = ['C3', 'E3', 'G3', 'C4', 'E4', 'G4', 'C5', 'E5', 'G5', 'C6']
    return place(len(notes) * .045 + .8, [(i * .045, piano(note(n), .9, .6 + .4 * i / len(notes))) for i, n in enumerate(notes)]), 4500, .18


def debuff(v):  # piano, five heavy steps down: E C A E A
    notes, step, dur = ['E5', 'C5', 'A4', 'E4', 'A3'], .1, .45
    return place(len(notes) * step + dur, [(i * step, piano(note(n), dur, 1 - .3 * i / len(notes)) * np.exp(-t_axis(dur) / (dur * .4))) for i, n in enumerate(notes)]), 3500, .15


CUES = {  # cue: (function, drafts, loudness)
    'swing': (swing, 'A', .15), 'shoot': (shoot, 'A', .14), 'hit': (hit, 'ABC', .17), 'crit': (crit, 'A', .2),
    'block': (block, 'A', .15), 'fall': (fall, 'A', .15), 'duel': (duel, 'A', .14),
    'turn': (turn, 'AB', .12), 'levelup': (levelup, 'AB', .14), 'promote': (promote, 'AB', .15),
    'ui-select': (ui_select, 'A', .09), 'ui-back': (ui_back, 'A', .08), 'goals': (goals, 'AB', .11),
    'coins': (coins, 'A', .1), 'arrival': (arrival, 'A', .16), 'cast': (cast, 'A', .12),
    'fire': (fire, 'A', .14), 'fire-big': (fire_big, 'A', .17), 'earth': (earth, 'A', .14), 'boulder': (boulder, 'A', .17),
    'water': (water, 'A', .15), 'heal': (heal, 'AB', .11), 'scheme': (scheme, 'A', .12), 'buff': (buff, 'A', .12),
    'debuff': (debuff, 'A', .12),
}


def loudness(x, target):
    win = int(.02 * SR)
    r = max(np.sqrt(np.mean(x[i:i + win] ** 2)) for i in range(0, max(1, len(x) - win), win // 2))
    x = x * target / (r + 1e-9)
    p = np.max(np.abs(x))
    return x * (.89 / p) if p > .89 else x


def main():
    global rng
    out = Path(sys.argv[1])
    out.mkdir(parents=True, exist_ok=True)
    for name in sys.argv[2:] or list(CUES):
        fn, drafts, target = CUES[name]
        for v in drafts:
            rng = np.random.default_rng(zlib.crc32(f'{name}-{v}'.encode()))
            x, bright, wet = fn(v)
            x = fade(loudness(finish(np.asarray(x, float), bright, wet), target))
            wavfile.write(out / f'{name}-{v}.wav', SR, (x * 32767).astype(np.int16))
        print(name)


if __name__ == '__main__':
    main()
