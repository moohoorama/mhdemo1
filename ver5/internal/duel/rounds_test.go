package duel

import (
	"srpg/internal/content"
	"testing"
)

// earlierRounds are the duels written before the rule "합 = max(5, 30 − 무력 차)" (or bent on purpose: 1장 tutorial,
// 목순·방열, a worn-out 장료). Each keeps the count it has; reworking one means deleting its line here.
var earlierRounds = map[string]int{
	"zhuo-zhangfei-dengmao": 12, "zhuo-guanyu-chengyuanzhi": 12, "yingchuan-liubei-bocai": 12,
	"sishui-guan-yu": 20, "hulao-brothers": 28, "hulao-fangyue": 4, "bianshui-zhangfei-lubu": 20,
	"xindu-guanyu-chunyuqiong": 12, "julu-zhangfei-yanliang": 20, "jiejiao-zhangfei-wenchou": 20,
	"beihai-zhaoyun-guanhai": 11, "taishan-guanyu-zangba": 9, "taishan-gaoshun-sunguan": 20,
	"xuyi-guanyu-jiling": 19, "xiaopei-xiahoudun-caoxing": 4, "xiapi-guanyu-zhangliao": 16,
	"runan-gongdu-xiahouyuan": 10, "bowang-zhaoyun-xiahoudun": 10,
}

func TestDuelRoundsFollowTheMightGap(t *testing.T) {
	data, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, stage := range data.Stages {
		for _, d := range stage.Duels {
			if len(d.Beats) == 0 {
				continue
			}
			seen[d.ID] = true
			rounds := 0
			for _, b := range d.Beats {
				if isStrike(b.Kind) {
					rounds++
				}
			}
			gap := data.Characters[d.Ally].Stats[1] - data.Characters[d.Enemy].Stats[1]
			if gap < 0 {
				gap = -gap
			}
			want := max(5, 30-gap)
			if exempt, ok := earlierRounds[d.ID]; ok {
				if rounds != exempt {
					t.Errorf("%s has %d 합, the exception table says %d (formula: %d)", d.ID, rounds, exempt, want)
				}
				continue
			}
			if rounds != want {
				t.Errorf("%s has %d 합, 무력 차 %d wants %d", d.ID, rounds, gap, want)
			}
		}
	}
	for id := range earlierRounds {
		if !seen[id] {
			t.Errorf("exception for unknown duel %s", id)
		}
	}
}
