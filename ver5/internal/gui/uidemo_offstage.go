package gui

import (
	"strings"

	"srpg/internal/content"
)

func init() { demos["offstage"] = demoOffstage }

// demoOffstage opens a stage (-stage B18) as demoOpening does and, at frame 90, shows a battlefield line said by
// someone not on the field (an Offstage line: the story's dialogue box with the messenger's name) and an ordinary one,
// spoken by the two characters -node names as "offstage,onstage" (default: the first of the stage's speeches with an
// Offstage line, then the lord).
func demoOffstage(d *Demo) error {
	if err := demoOpening(d); err != nil {
		return err
	}
	far, near := d.offstageSpeakers()
	opening := d.step
	d.step = func(f int) {
		opening(f)
		if f == 90 {
			d.g.showBattleSpeech([]content.Speech{{Actor: far, Offstage: true, Text: "주공! 하비가 조조군에게 에워싸였습니다! 관 장군께서 성안에 갇히셨습니다!"}})
		}
		if f == 200 {
			d.g.showBattleSpeech([]content.Speech{{Actor: near, Text: "운장이……!"}})
		}
	}
	return nil
}

func (d *Demo) offstageSpeakers() (far, near string) {
	if a, b, ok := strings.Cut(d.node, ","); ok {
		return a, b
	}
	data := d.g.S.Data
	for _, id := range data.Campaign.Roster {
		if data.Characters[id].Lord {
			near = id
		}
	}
	for _, st := range data.Stages {
		if st.ID != d.stage {
			continue
		}
		for _, ev := range st.Events {
			for _, a := range ev.Do {
				for _, l := range a.Lines {
					if l.Offstage && far == "" {
						far = l.Actor
					}
				}
			}
		}
	}
	return far, near
}
