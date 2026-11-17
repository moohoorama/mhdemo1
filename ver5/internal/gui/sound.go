package gui

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"srpg/internal/content"
	"srpg/internal/duel"
	"srpg/internal/duelview"
	"srpg/internal/storage"
)

const (
	sampleRate   = 44100
	sfxVolume    = .6
	musicVolume  = .35
	musicFadeOut = .8 // seconds for the old theme to die away before the next starts
	musicFadeIn  = 1.5
	// a looping theme dies away over its last seconds and comes back in, so its end does not cut off into its start
	musicLoopFadeOut = 6.0
	musicLoopFadeIn  = 1.0
	sfxGapTicks  = 3 // the same cue struck again within this many ticks is not doubled
)

// soundBank plays the sound effects (assets/sounds, a cue per file name, name-N files its variants) and one
// music theme at a time (assets/music/themes), cross-faded when the screen asks for another. A Game has
// one only when EnableSound was called: tests, captures and the CLI stay silent.
type soundBank struct {
	ctx      *audio.Context
	music    string // assets/music/themes
	cues     map[string][][]byte
	lastTick map[string]int
	lastPick map[string]int     // the variant each cue played last, not picked twice running
	peak     map[string]float64 // seconds from a cue's start to its loudest moment, the mean of its variants
	theme    string             // the theme playing or fading out
	player   *audio.Player
	file     *os.File
	vol      float64
	length   float64 // seconds of the theme playing
	lastPos  float64 // where in the loop the theme was last update
	looped   bool    // the theme has come round to its start at least once
}

var cueVariant = regexp.MustCompile(`-\d+$`)

// EnableSound loads the sound effects and starts the music; the game is silent until it is called.
func (g *Game) EnableSound() error {
	dir := filepath.Join(g.root, "assets")
	entries, err := os.ReadDir(filepath.Join(dir, "sounds"))
	if err != nil {
		return err
	}
	s := &soundBank{ctx: audio.NewContext(sampleRate), music: filepath.Join(dir, "music", "themes"), cues: map[string][][]byte{}, lastTick: map[string]int{}, lastPick: map[string]int{}, peak: map[string]float64{}}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".ogg")
		if !ok {
			continue
		}
		f, err := os.Open(filepath.Join(dir, "sounds", e.Name()))
		if err != nil {
			return err
		}
		pcm, err := decodeAll(vorbis.DecodeWithSampleRate(sampleRate, f))
		f.Close()
		if err != nil {
			return fmt.Errorf("sound %s: %w", e.Name(), err)
		}
		cue := cueVariant.ReplaceAllString(name, "")
		s.cues[cue] = append(s.cues[cue], pcm)
	}
	for cue, variants := range s.cues {
		sum := 0.0
		for _, pcm := range variants {
			sum += loudestAt(pcm)
		}
		s.peak[cue] = sum / float64(len(variants))
	}
	g.sound = s
	return nil
}

// loudestAt is the middle of the loudest 20 ms of 16-bit stereo PCM, in seconds.
func loudestAt(pcm []byte) float64 {
	const win = sampleRate / 50
	frames := len(pcm) / 4
	best, at := -1.0, 0
	for i := 0; i+win <= frames; i += win / 2 {
		e := 0.0
		for j := i; j < i+win; j++ {
			l := float64(int16(uint16(pcm[4*j]) | uint16(pcm[4*j+1])<<8))
			r := float64(int16(uint16(pcm[4*j+2]) | uint16(pcm[4*j+3])<<8))
			e += l*l + r*r
		}
		if e > best {
			best, at = e, i+win/2
		}
	}
	return float64(at) / sampleRate
}

// sfxLead is how long a cue takes to reach its loudest moment: started that much early, it strikes on time.
// It is 0 without sound, so silent captures keep their cues at the moment they belong to.
func (g *Game) sfxLead(cue string) float64 {
	if g.sound == nil {
		return 0
	}
	return g.sound.peak[cue]
}

// sfxAt plays a cue so that its loudest moment falls `at` seconds into the replay from now (or at once when
// it needs longer than that to get there).
func (p *Player) sfxAt(cue string, at float64) {
	if cue == "" {
		return
	}
	if d := at - p.g.sfxLead(cue); d > 0 {
		p.sounds = append(p.sounds, delayedSfx{d, cue})
		return
	}
	p.g.sfx(cue)
}

// sfxFrom starts a cue `at` seconds into the replay from now.
func (p *Player) sfxFrom(cue string, at float64) {
	if cue == "" {
		return
	}
	if at > 0 {
		p.sounds = append(p.sounds, delayedSfx{at, cue})
		return
	}
	p.g.sfx(cue)
}

type delayedSfx struct {
	left float64
	cue  string
}

// updateSounds plays the delayed cues that are due.
func (p *Player) updateSounds(dt float64) {
	kept := p.sounds[:0]
	for _, d := range p.sounds {
		if d.left -= dt; d.left <= 0 {
			p.g.sfx(d.cue)
		} else {
			kept = append(kept, d)
		}
	}
	p.sounds = kept
}

func decodeAll(r io.Reader, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// sfxTrace, when SRPG_SFX_TRACE names a file, gets a line "tick cue" (or "tick cue volume" when it plays quieter) for every cue and "tick music theme" for
// every change of theme, played or not, so a silent capture can be given its sound afterwards (tools/sfx/dub.py).
var sfxTrace = func() *os.File {
	if p := os.Getenv("SRPG_SFX_TRACE"); p != "" {
		if f, err := os.Create(p); err == nil {
			return f
		}
	}
	return nil
}()

// sfx plays a cue, unless effects are off or the same cue just played.
func (g *Game) sfx(cue string) { g.sfxVol(cue, 1) }

// sfxAtCell plays a cue made by someone standing at cell (u, v) of the field, quieter the farther it is from the
// middle of the screen.
func (g *Game) sfxAtCell(cue string, u, v float64) { g.sfxVolAtCell(cue, u, v, 1) }

// sfxVolAtCell is sfxAtCell at a share of the volume.
func (g *Game) sfxVolAtCell(cue string, u, v, vol float64) {
	if g.field == nil {
		g.sfxVol(cue, vol)
		return
	}
	x, y := g.field.Center(u, v)
	g.sfxVol(cue, vol*distanceGain((x-g.camX)*g.zoom/(float64(worldW)/2), (y-g.camY)*g.zoom/(float64(worldH)/2)))
}

// distanceGain is the volume of a sound at (dx, dy) from the middle of the screen, in halves of its width and
// height: full near the middle, a fifth at the screen's edge, silent well off it.
func distanceGain(dx, dy float64) float64 {
	const near, edge, far, atEdge = .35, 1.0, 1.6, .2
	switch d := math.Hypot(dx, dy); {
	case d <= near:
		return 1
	case d <= edge:
		return 1 - (1-atEdge)*(d-near)/(edge-near)
	case d < far:
		return atEdge * (far - d) / (far - edge)
	}
	return 0
}

// sfxVol plays a cue at a share of the effects' volume.
func (g *Game) sfxVol(cue string, vol float64) {
	if cue == "" || vol <= 0 {
		return
	}
	g.sfxCalls++
	if sfxTrace != nil && vol < 1 {
		fmt.Fprintf(sfxTrace, "%d %s %.2f\n", g.ticks, cue, vol)
	} else if sfxTrace != nil {
		fmt.Fprintf(sfxTrace, "%d %s\n", g.ticks, cue)
	}
	s := g.sound
	if s == nil || !sfxOn(g.S.Settings) {
		return
	}
	variants := s.cues[cue]
	if len(variants) == 0 {
		return
	}
	if t, ok := s.lastTick[cue]; ok && g.ticks-t < sfxGapTicks {
		return
	}
	s.lastTick[cue] = g.ticks
	i := rand.IntN(len(variants))
	if last, ok := s.lastPick[cue]; ok && len(variants) > 1 && i == last {
		i = (i + 1 + rand.IntN(len(variants)-1)) % len(variants)
	}
	s.lastPick[cue] = i
	p := s.ctx.NewPlayerFromBytes(variants[i])
	p.SetVolume(sfxVolume * vol)
	p.Play()
}

// sfxUnless plays a cue unless another was asked for since sfxCalls was `before`: a click's own sound gives way to
// the sound of what the click did (an order given, a window closed).
func (g *Game) sfxUnless(before int, cue string) {
	if g.sfxCalls == before {
		g.sfx(cue)
	}
}

func sfxOn(s storage.Settings) bool   { return s.Sound == "" || s.Sound == "sfx" }
func musicOn(s storage.Settings) bool { return s.Sound == "" || s.Sound == "music" }

// updateMusic fades toward the theme the screen asks for.
func (g *Game) updateMusic(dt float64) {
	if sfxTrace != nil {
		if theme := g.musicFor(); theme != g.tracedTheme {
			g.tracedTheme = theme
			fmt.Fprintf(sfxTrace, "%d music %s\n", g.ticks, theme)
		}
	}
	s := g.sound
	if s == nil {
		return
	}
	want := ""
	if musicOn(g.S.Settings) {
		want = g.musicFor()
	}
	if s.theme != want {
		s.vol -= dt / musicFadeOut
		if s.vol > 0 && s.player != nil {
			s.player.SetVolume(s.vol * musicVolume)
			return
		}
		s.stop()
		s.start(want)
	} else if s.vol < 1 {
		s.vol = min(1, s.vol+dt/musicFadeIn)
	}
	if s.player != nil {
		s.player.SetVolume(s.vol * s.loopEnvelope() * musicVolume)
	}
}

// loopEnvelope is the volume share the theme plays at where it is in its loop: down to nothing over its last
// musicLoopFadeOut seconds, back up over the first musicLoopFadeIn of the next time round.
func (s *soundBank) loopEnvelope() float64 {
	if s.length < 3*musicLoopFadeOut {
		return 1
	}
	pos := math.Mod(s.player.Position().Seconds(), s.length)
	s.looped = s.looped || pos < s.lastPos
	s.lastPos = pos
	k := min(1, (s.length-pos)/musicLoopFadeOut)
	if s.looped && pos < musicLoopFadeIn {
		k = min(k, pos/musicLoopFadeIn)
	}
	return max(0, k)
}

func (s *soundBank) stop() {
	if s.player != nil {
		s.player.PauseAndStopReading()
		s.file.Close()
	}
	s.player, s.file, s.theme, s.vol, s.length, s.lastPos, s.looped = nil, nil, "", 0, 0, 0, false
}

// start plays a theme from its beginning, looping; a theme that cannot be read stays silent.
func (s *soundBank) start(theme string) {
	s.theme = theme
	if theme == "" {
		return
	}
	f, err := os.Open(filepath.Join(s.music, theme+".ogg"))
	if err != nil {
		return
	}
	stream, err := vorbis.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		f.Close()
		return
	}
	p, err := s.ctx.NewPlayer(audio.NewInfiniteLoop(stream, stream.Length()))
	if err != nil {
		f.Close()
		return
	}
	s.player, s.file = p, f
	s.length = float64(stream.Length()) / (4 * sampleRate)
	p.SetVolume(0)
	p.Play()
}

// musicTable is assets/music/music.json: the theme for each kind of screen, and for the story and walk maps that
// have their own (a town, a government office). A scene's own Music wins over its map's.
type musicTable struct {
	Title, World, Story, Explore, Sortie, Battle, Duel, Victory, Defeat string
	Maps                                                          map[string]string // scene and walk maps' own themes
	Stages                                                        map[string]string // battles' own themes, by stage ID
}

func (g *Game) music() *musicTable {
	if g.musicTab == nil {
		g.musicTab = &musicTable{}
		if b, err := os.ReadFile(filepath.Join(g.root, "assets", "music", "music.json")); err == nil {
			_ = json.Unmarshal(b, g.musicTab)
		}
	}
	return g.musicTab
}

// musicFor is the theme of what is on screen.
func (g *Game) musicFor() string {
	m := g.music()
	switch {
	case g.gallery != nil || g.editor != nil || g.tour != nil:
		return ""
	case g.world != nil:
		return m.World
	case g.S.Screen == "title" || g.S.Engine == nil:
		return m.Title
	case g.player.Duel != nil && m.Duel != "":
		// the duel on screen, even when the engine has already moved on to the victory it won
		return m.Duel
	}
	byMap := func(name, fallback string) string {
		if t := m.Maps[name]; t != "" {
			return t
		}
		return fallback
	}
	o := g.S.View()
	switch o.Phase {
	case "scenario":
		switch o.Dialogue.Kind {
		case "battle":
			return m.Sortie
		case "explore":
			return byMap(g.S.Engine.ExploreFile().Areas[o.Spot.Area].Map, m.Explore)
		}
		if def, _ := g.scenes.For(o.Node); def != nil {
			if def.Music != "" {
				return def.Music
			}
			return byMap(def.Map, m.Story)
		}
		return m.Story
	case "deployment":
		return m.Sortie
	case "battle":
		if o.Map != nil && m.Stages[o.Map.ID] != "" {
			return m.Stages[o.Map.ID]
		}
		return m.Battle
	case "result":
		if o.Result == "victory" {
			return m.Victory
		}
		return m.Defeat
	}
	return m.Story
}

// skillCue is the sound of a skill landing: its own, else by its element or kind; none for a physical one.
func (g *Game) skillCue(skill string) string {
	s := g.S.Data.Skills[skill]
	switch {
	case s.Sound != "":
		return s.Sound
	case s.Element == "fire" || s.Element == "water" || s.Element == "earth":
		return s.Element
	case s.Kind == "heal":
		return "heal"
	case s.Kind == "buff":
		return "buff"
	case s.Kind == "status":
		return "debuff"
	}
	return "" // a physical skill is heard by its swing and its blows
}

// duelSounds plays the swings and blows of the duel's exchanges whose moments its clock passed since from.
func (g *Game) duelSounds(v *duelview.View, from float64) {
	for _, cue := range duelCues(v.Story, from, v.Time) {
		g.sfx(cue)
	}
}

// duelCues are the sounds of the exchanges between story times from (exclusive) and to: the swing as the blow is
// raised, then the hit, the critical hit or the block as it lands (.56 into the exchange, where duel.apply puts
// the contact); the loser's fall as the duel is won.
func duelCues(story duel.Story, from, to float64) []string {
	var cues []string
	for i, b := range story.Beats {
		start := story.Start(i)
		at := func(t float64, cue string) {
			if t += start; t > from && t <= to {
				cues = append(cues, cue)
			}
		}
		switch strings.TrimPrefix(b.Kind, "rush-") {
		case "hit", "backhit":
			at(.2, "swing")
			at(.56, "hit")
		case "critical":
			at(.2, "swing-crit")
			at(.56, "crit")
		case "block", "yielding-block":
			at(.2, "swing")
			at(.56, "block")
		case "critical-block":
			at(.2, "swing-crit")
			at(.56, "block")
		case "dodge":
			at(.2, "swing")
		case "critical-dodge":
			at(.2, "swing-crit")
		case "crossing-charge":
			at(.3, "swing")
			at(.6, "block")
		case "slash-through":
			at(.1, "swing-crit")
			at(.5, "crit")
		case "shoot": // loosed at .35 s, strikes hard at .7 s (duel/timeline.go)
			at(.35, "shoot")
			at(.7, "crit")
		case "win":
			at(.1, "duel-fall") // the loser sinks, greying
		}
	}
	return cues
}

// riderArt reports whether a sprite sheet shows someone on horseback: a mounted class's sheet, an officer's sheet
// for a mounted class, or one named for riding.
func (g *Game) riderArt(id string) bool {
	if g.S == nil || g.S.Data == nil {
		return false
	}
	if strings.Contains(id, "mounted") || strings.HasPrefix(id, "horse_") {
		return true
	}
	c, ok := g.artClass(id)
	return ok && slices.Contains(c.Tags, content.TagMounted)
}

// artClass is the class a sprite sheet draws: a class's own sheet, an officer's sheet for a class, or a variant of a
// class's sheet named after it (light_cavalry_white).
func (g *Game) artClass(id string) (content.Class, bool) {
	if g.S == nil || g.S.Data == nil {
		return content.Class{}, false
	}
	d := g.S.Data
	for _, c := range d.Classes {
		if c.Sprite == id {
			return c, true
		}
	}
	for _, ch := range d.Characters {
		for class, sheet := range ch.Look.Sprites {
			if sheet == id {
				c, ok := d.Classes[strings.SplitN(class, ":", 2)[0]]
				return c, ok
			}
		}
	}
	best := content.Class{}
	for _, c := range d.Classes {
		if c.Sprite != "" && strings.HasPrefix(id, c.Sprite+"_") && len(c.Sprite) > len(best.Sprite) {
			best = c
		}
	}
	return best, best.Sprite != ""
}

// sceneFootfall sounds a scene walker's steps, about one per cell, hoof beats for a rider; the people who wander
// or patrol of their own accord walk quietly.
func (g *Game) sceneFootfall(key string, v *unitVis, to [2]float64) {
	s := g.scene
	if s.patrol[key] != nil || g.explore != nil && g.explore.people[key] != "" {
		return
	}
	stride, ok := s.stride[key]
	if !ok {
		stride = 1
	}
	if stride += math.Hypot(to[0]-v.u, to[1]-v.v); stride < .9 {
		s.stride[key] = stride
		return
	}
	s.stride[key] = stride - 1
	cue := "step"
	if v.mounted {
		cue = "hoof"
	}
	vol := 1.0
	if s.def.Steps > 0 {
		vol = s.def.Steps
	}
	g.sfxVolAtCell(cue, v.u, v.v, vol)
}

// fallCue is the sound of a unit beaten off the field: an officer of ours with the erhu's sigh, an enemy officer
// or commander with the drum and the big gong, a troop with the falling bells.
func (g *Game) fallCue(id string) string {
	u := g.unitView(g.player.final, id)
	if u == nil {
		return "fall"
	}
	c, named := g.S.Data.Characters[u.Character]
	named = named && !c.Template
	switch {
	case u.Faction == "ally" && named:
		return "ally-fall"
	case u.Faction == "enemy" && (named || u.Rank == "commander" || u.Rank == "boss"):
		return "leader-fall"
	}
	return "fall"
}

// sceneMotionCue is the sound of a motion a scene plays: a swing, or a shot from a bow; a blow taken or blocked.
func (g *Game) sceneMotionCue(v *unitVis, anim string) string {
	switch anim {
	case "attack":
		if c, ok := g.artClass(v.art.ID); ok && c.Attack == content.AttackRanged {
			return "shoot"
		}
		return "swing"
	case "hit", "block":
		return anim
	case "cleave", "slash_l", "slash_r": // 관우's scene-only overhead cleave and flat sweeps
		return "swing"
	case "brace": // a block held overhead against it
		return "block"
	}
	return ""
}

// emoteCues are the sounds of a scene's emotes; one not listed is silent.
var emoteCues = map[string]string{"!": "emote-surprise", "?": "emote-question", "…": "emote-silence"}

// meritSounds sounds what the victory window's show has brought in since the last frame, t seconds into it: each
// podium card as it lands (a higher bell for a higher place), the other officers' lines, the spoils. A click that
// skips the show skips its sounds.
func (g *Game) meritSounds(officers int, t float64) {
	if g.resultSkip {
		g.meritHeard = t
		return
	}
	places, restAt, spoilsAt := meritTimes()
	type cue struct {
		at   float64
		name string
	}
	var cues []cue
	for i, name := range []string{"merit-gold", "merit-silver", "merit-bronze"} {
		if i < officers {
			cues = append(cues, cue{places[i] + meritDrop*.7, name})
		}
	}
	if officers > 3 {
		cues = append(cues, cue{restAt, "merit-row"})
	}
	cues = append(cues, cue{spoilsAt, "coins"})
	for _, c := range cues {
		if c.at > g.meritHeard && c.at <= t {
			g.sfx(c.name)
		}
	}
	g.meritHeard = t
}
