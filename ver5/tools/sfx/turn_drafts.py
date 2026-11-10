"""Drafts for the turn's start: ten in the levelup's brass (trumpet, a horn an octave below, the music's five notes, its
room), draft 3 on six Chinese instruments (the turn is chinese 2, the dizi), the levelup on the dizi, and the retreat's
falling E C A E A on Chinese instruments, and four victory fanfares rising through C E G to the top C.

    python3 tools/sfx/turn_drafts.py OUT_DIR [brass|chinese|levelup|fall|victory]"""
import sys
import zlib
from pathlib import Path

import numpy as np
from scipy.io import wavfile

sys.path.insert(0, str(Path(__file__).parent))
import synth  # noqa: E402
from synth import SR, brass_section, fade, finish, loudness, note, place, taiko  # noqa: E402


def call(notes, horn=True, loud=.9):
    """Notes as (name, seconds, gap after); a horn doubles each an octave below."""
    parts, at = [], 0
    for i, (n, d, gap) in enumerate(notes):
        last = i == len(notes) - 1
        f = note(n)
        p = brass_section([f], d, 1.0 if last else loud, voices=2)
        if horn:
            p = p + .6 * brass_section([f / 2], d, .8, voices=1)
        parts.append((at, p))
        at += d + gap
    return parts, at


def chord(names, d, loud=1.0):
    return brass_section([note(n) for n in names], d, loud, voices=2)


def draft(k):
    if k == 1:  # 빠-밤: a fourth up
        p, t = call([('G4', .12, .03), ('C5', .45, 0)])
    elif k == 2:  # a slow horn call rising C G C
        p, t = call([('C4', .18, .04), ('G4', .18, .04), ('C5', .5, 0)], loud=.75)
    elif k == 3:  # 따-란: up a fifth, short
        p, t = call([('C5', .1, .02), ('G5', .4, 0)])
    elif k == 4:  # three quick notes up and a held top: E G C
        p, t = call([('E5', .07, .02), ('G5', .07, .02), ('C6', .45, 0)])
    elif k == 5:  # a chord that swells in
        x = chord(['C4', 'G4', 'C5', 'E5'], .8)
        x *= np.minimum(np.arange(len(x)) / SR / .35, 1)
        p, t = [(0, x)], .8
    elif k == 6:  # a military signal on the horn: G D G
        p, t = call([('G3', .14, .05), ('D4', .14, .05), ('G4', .5, 0)], horn=False)
        p = [(a, x * 1.0) for a, x in p]
    elif k == 7:  # a call and its softer echo
        p1, t1 = call([('G5', .1, .02), ('C6', .3, 0)])
        p2, t2 = call([('G5', .1, .02), ('C6', .3, 0)], loud=.6)
        p, t = p1 + [(t1 + .12 + a, x * .45) for a, x in p2], t1 + .12 + t2
    elif k == 8:  # a chord stab over a drum
        p, t = [(0, chord(['C4', 'G4', 'C5', 'E5'], .35)), (0, taiko(70, .6, .2) * 1.4)], .6
    elif k == 9:  # da-da-DAN on one note, the levelup without its leap
        p, t = call([('G4', .08, .025), ('G4', .08, .025), ('C5', .5, 0)])
    else:  # a calmer horn in A: A E A
        p, t = call([('A3', .16, .04), ('E4', .16, .04), ('A4', .55, 0)], loud=.7)
    return place(t + .1, p)


# ---- draft 3 (따-란: C5 up to G5) on Chinese instruments ------------------------------------------------------

def t_of(d):
    return np.arange(int(d * SR)) / SR


def suona(f0, d, scoop=.06):
    """A double reed: a buzzy pulse through a nasal resonance, scooping up into the note, with a wide late vibrato."""
    t = t_of(d)
    cents = -scoop * 1200 * np.exp(-t / .04) + 25 * np.sin(2 * np.pi * 6 * t) * np.clip((t - .12) / .15, 0, 1)
    ph = np.cumsum(f0 * 2 ** (cents / 1200)) / SR
    src = ((ph % 1) < .22).astype(float) - .22
    y = synth.band(src, 900, 1500) * 1.2 + synth.band(src, 2200, 3200) * .8 + synth.lowpass(src, 900) * .5
    env = np.minimum(t / .02, 1) * np.clip((d - t) / .05, 0, 1)
    return y * env


def dizi(f0, d):
    """A bamboo flute whose membrane buzzes: a breathy tone with a rasp riding its upper partials."""
    t = t_of(d)
    f = f0 * (1 + .006 * np.sin(2 * np.pi * 5.5 * t) * np.clip((t - .1) / .1, 0, 1))
    ph = 2 * np.pi * np.cumsum(f) / SR
    tone = np.sin(ph) + .35 * np.sin(2 * ph) + .2 * np.sin(3 * ph) + .1 * np.sin(4 * ph)
    rasp = synth.band(synth.rng.standard_normal(len(t)), 2500, 5000) * (.5 + .5 * np.sin(2 * ph)) * .5
    breath = synth.band(synth.rng.standard_normal(len(t)), f0, f0 * 4) * .15
    env = np.minimum(t / .03, 1) * np.clip((d - t) / .08, 0, 1)
    return (tone + rasp + breath) * env


def zheng(f0, d, bend=0.0):
    """A guzheng string plucked, its pitch pressed up by `bend` semitones after the pluck."""
    x = synth.koto(f0, d, .7)
    if bend:
        t = t_of(d)
        r = 2 ** (bend * np.clip((t - .08) / .15, 0, 1) / 12)
        idx = np.cumsum(r)
        idx = idx[idx < len(x) - 1]
        x = np.interp(idx, np.arange(len(x)), x)
    return x


def erhu(f0, d, slide_from=None):
    """A two-string fiddle: a nasal bowed tone, sliding into the note, with a deep vibrato."""
    t = t_of(d)
    start = slide_from or f0
    f = f0 + (start - f0) * np.exp(-t / .05)
    f = f * 2 ** (30 * np.sin(2 * np.pi * 5.5 * t) * np.clip((t - .1) / .15, 0, 1) / 1200)
    ph = 2 * np.pi * np.cumsum(f) / SR
    saw = sum(np.sin(k * ph) / k for k in range(1, 14))
    y = synth.band(saw, 700, 1300) * 1.5 + synth.band(saw, 2000, 3000) * .6 + synth.lowpass(saw, 600) * .4
    env = np.minimum(t / .05, 1) * np.clip((d - t) / .08, 0, 1)
    return y * env


def bianzhong(f0, d=1.2):
    """A bronze chime bell: its strike tone, a minor third above it, and bright short partials."""
    ratios, amps, decays = [1, 1.19, 2.0, 2.76, 3.9, 5.4], [1, .55, .4, .3, .15, .08], [.9, .7, .4, .25, .12, .08]
    return synth.modal([f0 * r for r in ratios], amps, [x * d / 1.2 for x in decays], d) * np.minimum(t_of(d) / .001, 1)


def small_gong(d=.6):
    """The opera's small gong (小锣): a bright ring whose pitch rises after the strike."""
    t = t_of(d)
    f = 900 * (1 + .12 * (1 - np.exp(-t / .08)))
    ph = 2 * np.pi * np.cumsum(f) / SR
    x = (np.sin(ph) + .5 * np.sin(2.4 * ph) + .3 * np.sin(3.7 * ph)) * np.exp(-t / .22)
    return x + synth.band(synth.rng.standard_normal(len(t)), 2000, 6000) * np.exp(-t / .02) * .4


C5, G5 = note('C5'), note('G5')


def chinese(k):
    if k == 1:  # suona
        return place(.7, [(0, suona(C5, .12)), (.14, suona(G5, .45, .03))])
    if k == 2:  # dizi
        return place(.7, [(0, dizi(C5, .11)), (.13, dizi(G5, .45))])
    if k == 3:  # guzheng: the top string is F, pressed up a whole tone to G as zheng players do
        return place(1.2, [(0, zheng(C5, 1.0)), (.12, zheng(note('F5'), 1.1, 2))])
    if k == 4:  # erhu sliding up into G
        return place(.8, [(0, erhu(C5, .14)), (.15, erhu(G5, .5, C5 * 1.3))])
    if k == 5:  # bianzhong bells
        return place(1.4, [(0, bianzhong(C5, 1.0)), (.14, bianzhong(G5, 1.2))])
    # suona over the small gong and a drum: the opera's entrance
    return place(.9, [(0, small_gong()), (0, taiko(80, .4, .12) * .8), (.12, suona(C5, .12)), (.26, suona(G5, .45, .03))])


def levelup_dizi(k):
    """The levelup (C C C, then up to G) on the dizi."""
    notes = [(C5, .07), (C5, .07), (C5, .07)]
    if k == 1:  # the dizi alone
        p = [(i * .095, dizi(f, d)) for i, (f, d) in enumerate(notes)] + [(.285, dizi(G5, .6))]
        return place(1.0, p)
    if k == 2:  # a grace note flicked in front of the top (A to G), as dizi players ornament
        p = [(i * .095, dizi(f, d)) for i, (f, d) in enumerate(notes)] + [(.285, dizi(note('A5'), .04)), (.32, dizi(G5, .6))]
        return place(1.0, p)
    if k == 3:  # the dizi over the guzheng plucking the same notes an octave below
        p = [(i * .095, dizi(f, d)) for i, (f, d) in enumerate(notes)] + [(.285, dizi(G5, .6))]
        p += [(i * .095, zheng(f / 2, .6) * .5) for i, (f, d) in enumerate(notes)] + [(.285, zheng(G5 / 2, 1.0) * .6)]
        return place(1.3, p)
    # the dizi with the small gong on the top note
    p = [(i * .095, dizi(f, d)) for i, (f, d) in enumerate(notes)] + [(.285, dizi(G5, .6)), (.285, small_gong() * .6)]
    return place(1.0, p)


def fall_chinese(k):
    """The retreat (a fall from E5 down to A3 in five steps, about 0.8 s) on Chinese instruments."""
    names = ['E5', 'C5', 'A4', 'E4', 'A3']
    fs = [note(n) for n in names]
    step = .1
    if k == 1:  # guzheng plucking each step down, the last left to ring
        return place(.9, [(i * step, zheng(f, .9 if i == 4 else .5) * (1 - .1 * i)) for i, f in enumerate(fs)])
    if k == 2:  # erhu: one bow sliding down through the steps
        t = t_of(.8)
        idx = np.minimum((t / step).astype(int), 4)
        f = np.array(fs)[idx]
        f = np.convolve(f, np.ones(600) / 600, mode='same')  # glide between the steps
        f[:300], f[-300:] = fs[0], fs[-1]
        ph = 2 * np.pi * np.cumsum(f) / SR
        saw = sum(np.sin(j * ph) / j for j in range(1, 14))
        y = synth.band(saw, 700, 1300) * 1.5 + synth.band(saw, 2000, 3000) * .6 + synth.lowpass(saw, 600) * .4
        return place(.85, [(0, y * np.minimum(t / .04, 1) * np.clip((.8 - t) / .2, 0, 1))])
    if k == 3:  # dizi falling
        return place(.85, [(i * step, dizi(f, .14 if i < 4 else .35)) for i, f in enumerate(fs)])
    if k == 4:  # guzheng with the erhu sliding under it
        z = [(i * step, zheng(f, .9 if i == 4 else .5) * .8) for i, f in enumerate(fs)]
        return place(.9, z + [(.4, erhu(fs[-1], .45, fs[2]) * .5)])
    if k == 5:  # bronze bells, low and dark
        return place(1.0, [(i * step, bianzhong(f / 2, .7) * (1 - .12 * i)) for i, f in enumerate(fs)])
    # a guzheng rake down the whole scale, landing on the low A
    rake = ['A5', 'G5', 'E5', 'D5', 'C5', 'A4', 'G4', 'E4', 'D4', 'C4']
    p = [(i * .03, zheng(note(n), .4) * .6) for i, n in enumerate(rake)]
    return place(.95, p + [(.33, zheng(note('A3'), .6))])


def victory_chinese(k):
    """The victory (G4 C5 E5 up to a held G5 or C6, about 1.6 s) on Chinese instruments."""
    G4, E5, C6 = note('G4'), note('E5'), note('C6')
    if k == 1:  # suona over the small gong and drums, as the opera's general enters
        p = [(0, small_gong(.9)), (0, taiko(80, .5, .14) * .9), (.3, taiko(80, .4, .12) * .7)]
        p += [(.1, suona(G4, .12)), (.24, suona(C5, .12)), (.38, suona(E5, .12)), (.52, suona(G5, .9, .03))]
        p += [(.52, small_gong(.9) * .7), (.52, taiko(70, .6, .18))]
        return place(1.7, p)
    if k == 2:  # bronze bells climbing to the top C, the gong under the last
        fs = [G4, C5, E5, G5]
        p = [(i * .13, bianzhong(f, 1.0) * .8) for i, f in enumerate(fs)] + [(.52, bianzhong(C6, 1.4)), (.52, small_gong(.8) * .5)]
        return place(2.0, p)
    if k == 3:  # a guzheng rake up the scale, the dizi holding G over it
        rake = ['C4', 'D4', 'E4', 'G4', 'A4', 'C5', 'D5', 'E5', 'G5']
        p = [(i * .035, zheng(note(n), .6) * .6) for i, n in enumerate(rake)]
        p += [(.32, dizi(G5, 1.0)), (.32, small_gong(.8) * .5), (.32, zheng(C5 / 2, 1.2) * .7)]
        return place(1.5, p)
    # drums rolling up to a crash: suona and erhu together on the top C
    p = [(i * .07, taiko(90 + 6 * i, .25, .1) * (.5 + .07 * i)) for i in range(7)]
    p += [(.49, taiko(70, .7, .2)), (.49, small_gong(1.0)), (.49, suona(C6 / 2, 1.0, .03) * .8), (.49, erhu(C5, 1.0, G4) * .6)]
    return place(1.7, p)


def main():
    out = Path(sys.argv[1])
    out.mkdir(parents=True, exist_ok=True)
    which = sys.argv[2] if len(sys.argv) > 2 else 'brass'
    count = {'brass': 10, 'chinese': 6, 'levelup': 4, 'fall': 6, 'victory': 4}[which]
    for k in range(1, count + 1):
        synth.rng = np.random.default_rng(zlib.crc32(f'turn-{which}-{k}'.encode()))
        x = {'brass': draft, 'chinese': chinese, 'levelup': levelup_dizi, 'fall': fall_chinese, 'victory': victory_chinese}[which](k)
        x = fade(loudness(finish(np.asarray(x, float), 5000 if which != 'brass' else 4000, .18), .13))
        wavfile.write(out / f'{which}-{k}.wav', SR, (x * 32767).astype(np.int16))
    print(out)


if __name__ == '__main__':
    main()
