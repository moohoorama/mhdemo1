package gui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUIContentFits(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "dialogue"})
	require.NoError(t, err)
	require.NoError(t, d.g.validateUIContent())
	for i := range d.g.S.Data.Nodes {
		node := &d.g.S.Data.Nodes[i]
		if node.Kind != "say" {
			continue
		}
		original := node.Text
		node.Text = strings.Repeat("아주 긴 대사 ", 80)
		err = d.g.validateUIContent()
		node.Text = original
		require.ErrorContains(t, err, node.ID)
		break
	}

	_, _, ok := d.g.dialogueFit(strings.Repeat("아주 긴 대사 ", 80), 800, 531)
	require.False(t, ok)
	_, _, ok = d.g.confirmLayout(strings.Repeat("출진하시겠습니까? ", 100), 800, 531)
	require.False(t, ok)
	_, ok = d.g.objectiveLayout([]objectiveRow{{name: "승리 조건", text: strings.Repeat("적군을 물리치고 아군을 지킨다. ", 100)}}, 800, 531)
	require.False(t, ok)
}
