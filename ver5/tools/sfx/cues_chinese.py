"""The smaller cues on the same Chinese instruments as the turn, the levelup, the retreat and the victory
(turn_drafts.py): the enemy's turn, a duel's opening gong, the defeat, an officer in danger, the sortie's drums,
a save, the victory window's merit cards, the scene emotes, things won, a bonus goal met, the shop's buying and
selling, one of ours retreating, an enemy officer felled, reinforcements of either side and a stratagem that
misses. Writes OUT_DIR/<cue>.wav.

    python3 tools/sfx/cues_chinese.py OUT_DIR"""
import sys
import zlib
from pathlib import Path

import numpy as np
from scipy.io import wavfile

sys.path.insert(0, str(Path(__file__).parent))
import synth  # noqa: E402
from synth import SR, fade, finish, loudness, note, place, taiko  # noqa: E402
from turn_drafts import bianzhong, dizi, erhu, small_gong, suona, t_of, zheng  # noqa: E402


def big_gong(f0=150, d=2.4):
    """The big gong (大锣, 징): a low bloom of crowded partials whose pitch sags after the strike."""
    t = t_of(d)
    sag = 1 - .06 * (1 - np.exp(-t / .3))
    x = 0
    for r, a, dec in zip([1, 1.48, 2.03, 2.71, 3.33, 4.17, 5.06], [1, .7, .55, .4, .3, .2, .12],
                         [2.0, 1.4, 1.0, .7, .5, .35, .25]):
        ph = 2 * np.pi * np.cumsum(f0 * r * sag) / SR
        x = x + a * np.sin(ph + synth.rng.uniform(0, 6.28)) * np.exp(-t / (dec * d / 2.4))
    swell = np.minimum(t / .03, 1) * (.6 + .4 * np.minimum(t / .25, 1))
    hit = synth.band(synth.rng.standard_normal(len(t)), 300, 3000) * np.exp(-t / .03) * .5
    return x * swell + hit


def bangzi(f0=1300, d=.12):
    """The opera's wooden clappers (梆子): a hard, hollow knock."""
    return synth.modal([f0, f0 * 2.3, f0 * 3.9], [1, .5, .25], [.035, .02, .012], d) * np.minimum(t_of(d) / .0005, 1)


def turn_enemy():
    """The enemy's turn: a low drum and the suona falling A to E, against the player's dizi rising C to G."""
    return place(.9, [(0, taiko(70, .5, .16)), (.02, suona(note('A4'), .14)), (.17, suona(note('E4'), .5, .02))])


def duel_start():
    """A duel's opening: the big gong, a drum under it."""
    return place(2.4, [(0, big_gong()), (0, taiko(60, .6, .2) * .8)])


def defeat():
    """The defeat: low bells falling E C A, the erhu sighing down under the last."""
    fs = [note('E4'), note('C4'), note('A3')]
    p = [(i * .32, bianzhong(f, 1.3) * (1 - .15 * i)) for i, f in enumerate(fs)]
    return place(2.2, p + [(.64, erhu(note('A3'), .9, note('C4')) * .45)])


def danger():
    """An officer in danger: the clappers knocking twice, fast, as the opera quickens."""
    return place(.35, [(0, bangzi()), (.11, bangzi(1400))])


def sortie():
    """The sortie: war drums 둥 둥 둥-, the small gong on the last."""
    return place(1.2, [(0, taiko(80, .4, .14) * .8), (.22, taiko(80, .4, .14) * .85), (.44, taiko(70, .7, .22)),
                       (.44, small_gong(.7) * .55)])


def save():
    """A save: one high bell, soft."""
    return place(1.0, [(0, bianzhong(note('G5'), .9) * .7)])


def merit(f0):
    """A merit card landing: a bell, the gold the highest."""
    return place(1.0, [(0, bianzhong(f0, .9)), (0, zheng(f0 / 2, .6) * .4)])


def merit_row():
    """The other officers coming in as lines: a guzheng pluck."""
    return place(.6, [(0, zheng(note('C5'), .5) * .8)])


def emote_surprise():
    """"!": a clapper knock and a guzheng string snapped up."""
    return place(.5, [(0, bangzi(1500)), (.02, zheng(note('E6'), .4, 2) * .5)])


def emote_question():
    """"?": a guzheng string pressed up a minor third, as a voice rises at a question."""
    return place(.6, [(0, zheng(note('A5'), .55, 3) * .7)])


def emote_silence():
    """"…": a low string plucked softly and let sag."""
    return place(.7, [(0, zheng(note('A4'), .6, -1) * .45)])


def acquire():
    """Something won: the guzheng swept up the pentatonic, a high bell on top."""
    run = ['C5', 'D5', 'E5', 'G5', 'A5', 'C6']
    p = [(i * .045, zheng(note(n), .5) * .55) for i, n in enumerate(run)]
    return place(1.4, p + [(len(run) * .045, bianzhong(note('C6'), 1.0))])


def subgoal():
    """A bonus goal met: two bells G then C, the small gong under the second."""
    return place(1.3, [(0, bianzhong(note('G5'), .8) * .8), (.16, bianzhong(note('C6'), 1.1)), (.16, small_gong(.6) * .3)])


def coin(f0):
    """A copper cash coin struck: a short, bright, slightly clattering ring."""
    return synth.modal([f0, f0 * 1.52, f0 * 2.27, f0 * 3.1], [1, .6, .4, .2], [.09, .06, .04, .03], .25)


def abacus():
    """An abacus bead flicked home: a small, hard wooden click."""
    return bangzi(2600, .05) * .7


def shop_buy():
    """Buying: the abacus clicked three times, then the coins counted out onto the counter."""
    p = [(i * .07, abacus()) for i in range(3)]
    p += [(.26 + i * .06, coin(f) * .5) for i, f in enumerate([3400, 3000, 2700])]
    return place(.8, p)


def shop_sell():
    """Selling: coins poured into the purse, a little bell as they settle."""
    p = [(i * .045 + synth.rng.uniform(0, .015), coin(synth.rng.uniform(2800, 4200)) * .45) for i in range(6)]
    return place(1.0, p + [(.3, bianzhong(note('E6'), .6) * .5)])


def ally_fall():
    """One of ours retreating: the erhu sliding down a fourth, a low bell under it."""
    return place(1.4, [(0, erhu(note('E4'), .9, note('A4')) * .6), (.05, bianzhong(note('A3'), 1.2) * .7)])


def leader_fall():
    """An enemy officer felled: the drum rolled three times into the big gong."""
    p = [(i * .09, taiko(90, .3, .1) * (.6 + .1 * i)) for i in range(3)]
    return place(2.2, p + [(.27, big_gong(170, 1.9)), (.27, taiko(60, .6, .2) * .8)])


def reinforce_enemy():
    """The enemy's reinforcements: the suona's alarm, two short blasts and a long one, over the drum."""
    return place(1.1, [(0, taiko(70, .5, .16)), (0, suona(note('E5'), .12)), (.15, suona(note('E5'), .12)),
                       (.3, suona(note('G5'), .5, .02))])


def reinforce_ally():
    """Our reinforcements: the drum and the dizi leaping up G to D."""
    return place(1.1, [(0, taiko(80, .45, .15) * .8), (.02, dizi(note('G5'), .16)), (.18, dizi(note('D6'), .55))])


def spell_miss():
    """A stratagem that misses: a guzheng string plucked and damped at once, 탁, a knock under it."""
    t = t_of(.25)
    return place(.4, [(0, zheng(note('D4'), .25) * np.exp(-t / .03)), (0, bangzi(700, .1) * .6)])


CUES = {
    'turn-enemy': turn_enemy, 'duel-start': duel_start, 'defeat': defeat, 'danger': danger, 'sortie': sortie,
    'save': save, 'merit-gold': lambda: merit(note('C6')), 'merit-silver': lambda: merit(note('G5')),
    'merit-bronze': lambda: merit(note('E5')), 'merit-row': merit_row, 'emote-surprise': emote_surprise,
    'emote-question': emote_question, 'emote-silence': emote_silence, 'acquire': acquire, 'subgoal': subgoal,
    'shop-buy': shop_buy, 'shop-sell': shop_sell, 'ally-fall': ally_fall, 'leader-fall': leader_fall,
    'reinforce-enemy': reinforce_enemy, 'reinforce-ally': reinforce_ally, 'spell-miss': spell_miss,
}
LEVEL = {'danger': .11, 'save': .09, 'merit-row': .09, 'emote-surprise': .1, 'emote-question': .09,
         'emote-silence': .07, 'shop-buy': .1, 'shop-sell': .1, 'spell-miss': .1}


def main():
    out = Path(sys.argv[1])
    out.mkdir(parents=True, exist_ok=True)
    for name, make in CUES.items():
        synth.rng = np.random.default_rng(zlib.crc32(name.encode()))
        x = fade(loudness(finish(np.asarray(make(), float), 5000, .18), LEVEL.get(name, .13)))
        wavfile.write(out / f'{name}.wav', SR, (x * 32767).astype(np.int16))
        print(name, f'{len(x) / SR:.2f}s')


if __name__ == '__main__':
    main()
