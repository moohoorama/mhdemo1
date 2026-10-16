package gui

import (
	"image/color"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Battle log lines are coloured by meaning (docs/ui-mockup): damage and defeats in
// cinnabar, healing and gains in green, the officers who act in pale gold, the rest white.
var (
	logPlain  = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	logHurt   = color.NRGBA{R: 236, G: 112, B: 90, A: 255}
	logGain   = color.NRGBA{R: 143, G: 217, B: 156, A: 255}
	logPerson = color.NRGBA{R: 240, G: 217, B: 160, A: 255}

	logHurtRe = regexp.MustCompile(`(지속 )?피해 \d+|격파|퇴각|사망`)
	logGainRe = regexp.MustCompile(`(병력|병법치|HP|MP) \+\d+|레벨 \d+|레벨 업`)
)

type logSpan struct {
	text string
	c    color.NRGBA
}

// logSpans splits a log line into coloured spans: the names before the colon, then the
// hurt and gain phrases after it.
func (g *Game) logSpans(line string) []logSpan {
	out := []logSpan{}
	rest := line
	if i := strings.Index(line, ": "); i > 0 {
		head := line[:i]
		for j, name := range strings.Split(head, " → ") {
			if j > 0 {
				out = append(out, logSpan{" → ", logPlain})
			}
			c := logPlain
			if _, ok := g.names[name]; ok {
				c = logPerson
			}
			out = append(out, logSpan{name, c})
		}
		out = append(out, logSpan{": ", logPlain})
		rest = line[i+2:]
	}
	type mark struct {
		a, b int
		c    color.NRGBA
	}
	marks := []mark{}
	for _, m := range logHurtRe.FindAllStringIndex(rest, -1) {
		marks = append(marks, mark{m[0], m[1], logHurt})
	}
	for _, m := range logGainRe.FindAllStringIndex(rest, -1) {
		marks = append(marks, mark{m[0], m[1], logGain})
	}
	at := 0
	for len(marks) > 0 {
		first := 0
		for i, m := range marks {
			if m.a < marks[first].a {
				first = i
			}
		}
		m := marks[first]
		marks = append(marks[:first], marks[first+1:]...)
		if m.a < at {
			continue
		}
		if m.a > at {
			out = append(out, logSpan{rest[at:m.a], logPlain})
		}
		out = append(out, logSpan{rest[m.a:m.b], m.c})
		at = m.b
	}
	if at < len(rest) {
		out = append(out, logSpan{rest[at:], logPlain})
	}
	return out
}

func (g *Game) logLine(dst *ebiten.Image, line string, x, y float64) {
	for _, s := range g.logSpans(line) {
		g.label(dst, s.text, x, y, 14, s.c)
		x += g.width(s.text, 14)
	}
}
