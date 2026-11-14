package gui

import "srpg/internal/content"

func init() { demos["offstage"] = demoOffstage }

// demoOffstage opens a stage (-stage B18) as demoOpening does and, at frame 90, shows a battlefield line said by
// someone not on the field (an Offstage line: the messenger's box at the top of the screen) and an ordinary one.
func demoOffstage(d *Demo) error {
	if err := demoOpening(d); err != nil {
		return err
	}
	opening := d.step
	d.step = func(f int) {
		opening(f)
		if f == 90 {
			d.g.showBattleSpeech([]content.Speech{{Actor: "messenger", Offstage: true, Text: "주공! 하비가 조조군에게 에워싸였습니다! 관 장군께서 성안에 갇히셨습니다!"}})
		}
		if f == 200 {
			d.g.showBattleSpeech([]content.Speech{{Actor: "liubei", Text: "운장이……!"}})
		}
	}
	return nil
}
