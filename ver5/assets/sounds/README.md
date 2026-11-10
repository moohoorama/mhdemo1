# Sounds

Sound effects the GUI plays (`internal/gui/sound.go`). A file is a cue by its name; `name-N.ogg` files are variants of
the cue `name`, one picked at random each time. A duel plays swing, swing-crit, hit, crit and block at its exchanges and duel-fall when it is won. A skill's cue is its `Sound` in skills.json, else its element (fire,
water, earth) or kind (heal, buff, status → debuff); a physical skill has none of its own (its swing and blows). All files are 44.1 kHz stereo Vorbis.

## Recordings from Pixabay

Pixabay Content License (pixabay.com/service/license-summary): free to use and modify, no attribution needed; not to
be sold or distributed on their own as unchanged files, only inside the game. Fitted with `tools/sfx/fit.py` (silence
and noise floor cut, 30 Hz highpass, loudest 20 ms matched to the level) unless the row says otherwise.

| cue | downloaded as | level |
|---|---|---|
| swing | konpeito_sound-knife_swish01-261127.mp3 | volume ×0.77 (by hand) |
| swing-crit | daviddumaisaudio-sword-slash-and-swing-185432.mp3 | by hand: 60 ms cut, 30 Hz highpass, ×0.54, faded from 1.2 s |
| hit | hiited.mp3 | .17 |
| block-1..3 | guard1.mp3, guard2.mp3, guard3.mp3 | .15 |
| shoot | arrow.mp3 | .14 |
| fire | fire.mp3 | .14 |
| fire-big | fire.mp3, three layers 0.12 s apart | .17 |
| earth | stone.mp3 | .14 |
| boulder | huge_stone.mp3 | .17 |
| water | water.mp3 | .15 |
| heal | heal.mp3, cut at 3.5 s | .11 |
| buff | buff.mp3 | .12 |
| debuff | buff.mp3 played backwards | .12 |
| scheme | dark.mp3 | .12 |
| crit | heavy_blade.mp3 | .20 |
| coins | coin.mp3 | .10 |
| goals | accept.mp3 | .11 |
| arrival | short_impact.mp3 sped up ×8.8 to 0.3 s | .16 |
| step-1..7 | freesound_community-running-in-grass-6237.mp3, its seven steps cut 0.22 s each by `tools/sfx/steps.py` (one per cell walked; a variant never twice running) | .05 |
| hoof-1..6 | universfield-horse-walking-123782.mp3: ten single hoof beats cut 0.1 s each (at 2.12 2.34 2.60 3.33 3.84 4.10 4.35 5.12 5.37 5.62 s), paired into six 따각s by `steps.py … --pairs 6` (one per cell a rider walks) | .05 |
| ok | ok.mp3 (an order given: a battle command's cell, target or 대기, or an explore destination — a person, exit or sign) | .064 |
| close | cancel.mp3 (a window or modal closing), sped up ×1.35 to 0.2 s | .056 |
| ui-denied | denied.mp3 (a greyed-out menu row clicked, a warning toast), sped up ×1.55 to 0.2 s | .07 |
| duel-fall | fall.mp3 (the loser of a duel sinking) | .15 |

ok, close and ui-denied (2026-11-09) are cut tighter than `fit.py` cuts (from the onset to 30 dB under the peak) and the
longer two resampled faster to about 0.2 s, which raises their pitch. A plain click (ui-select) or Esc (ui-back) sounds
only when what it did made no sound of its own.

Every recording above (all but the synthesised cues below) was then brought toward the music's tone with
`tools/sfx/match.py` (2026-11-09): the mean 1/3-octave spectrum of sipmyeon-maebok, gosan-yusu and maehwa-samnong, each
band pulled 60% of the way, at most 9 dB, loudest 20 ms kept. Fit a new recording first, then match it.

## Synthesised

With `tools/sfx/synth.py` (2026-11-08). From the round fitted to the music: ui-select A, ui-back A. From the
first round: intro (an officer's first appearance) is swing B, fall (a troop retreating) is draft 5 of
`tools/sfx/turn_drafts.py fall`: bronze bells (편종) falling E C A E A an octave low, cut to 0.8 s. turn is draft 2 of `tools/sfx/turn_drafts.py chinese`: the dizi playing C then G; levelup (and promote, a copy of
it) is draft 2 of `turn_drafts.py levelup`: the dizi playing C C C, a flicked A, then G. victory (the won battle's result window opening) is draft 4 of `turn_drafts.py victory`: drums
rolling up to a crash, the suona and the erhu on C. `tools/sfx/cues_chinese.py` makes, on the same instruments,
turn-enemy (the enemy's turn), duel-start (a duel's big gong), defeat (the lost battle's window), danger (one of ours
down to a quarter of its troops), sortie (a battle begun), save, merit-gold/-silver/-bronze/-row (the victory
window's podium cards and lines), emote-surprise/-question/-silence (a scene's !, ? and …), acquire (the window
of things won or officers joined), subgoal (a bonus goal met on the field), shop-buy and shop-sell, ally-fall (an
officer of ours retreating), leader-fall (an enemy officer or commander beaten), reinforce-enemy and reinforce-ally
(an army arriving on the field) and spell-miss (a stratagem that misses).

Music is `assets/music/themes/*.mp3`; which theme plays where is `assets/music/music.json` (by screen, and by story or
walk map), and a scene's own `Music` (assets/scenes) wins over its map's. gosan-yusu (고산유수), maehwa-samnong (매화삼롱),
sipmyeon-maebok (십면매복) and yangchun-baekseol (양춘백설) come from the user's musics.zip.

To hear a capture with its sound: run `cmd/_uidemo-vm` with `SRPG_SFX_TRACE=trace.txt` and `-mp4`, then
`python3 tools/sfx/dub.py trace.txt capture.mp4 out.mp4`.
