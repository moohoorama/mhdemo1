package gui

func init() { demos["storyconfirm"] = demoStoryConfirm }

func demoStoryConfirm(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	d.g.askWith("장세평과 소쌍을 구하러 지금 출진하시겠습니까?", "예", "아니오", func() {})
	d.g.confirmTitle = "출진 확인"
	return nil
}
