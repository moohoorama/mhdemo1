package content

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEngineCookbook loads every json block of the 22-27 battle cookbook (plan/_engine_22_27.md) as a stage of the
// real bundle: each block's fields replace those of a bare copy of B21, and officers the chapters have yet to add
// stand in as copies of an existing one.
func TestEngineCookbook(t *testing.T) {
	doc, err := os.ReadFile("../../plan/_engine_22_27.md")
	if os.IsNotExist(err) {
		t.Skip("no cookbook")
	}
	require.NoError(t, err)
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	base := cloneMap(d.Stages[d.StageIndex("B21")])
	base.Areas, base.Events, base.Defeat, base.Sub, base.Duels, base.Party, base.Groups, base.Recommend, base.Reserve = nil, nil, nil, nil, nil, nil, nil, nil, nil
	base.Required = []string{"liubei"}
	base.Victory = []Goal{{If: []string{"alldefeated:enemy"}, Text: "적 전멸"}}
	base.Enemies = []Spawn{{Character: "troop_footman", X: 19, Y: 31, Policy: "hold"}}
	baseJSON, err := json.Marshal(base)
	require.NoError(t, err)
	named := regexp.MustCompile(`"(?:Character|Actor|Unit|Ally|Enemy)":\s*"([a-z_]+)"`)
	blocks := regexp.MustCompile("(?s)```json\n(.*?)```").FindAllStringSubmatch(string(doc), -1)
	require.NotEmpty(t, blocks)
	for i, b := range blocks {
		t.Run(fmt.Sprint("block ", i+1), func(t *testing.T) {
			var fields, frag map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(baseJSON, &fields))
			require.NoError(t, json.Unmarshal([]byte(b[1]), &frag), b[1])
			for k, v := range frag {
				fields[k] = v
			}
			merged, err := json.Marshal(fields)
			require.NoError(t, err)
			var st Stage
			require.NoError(t, strictDecode(merged, &st))
			st.ID = fmt.Sprint("COOK", i+1)
			c := cloneMap(*d)
			c.Explores = d.Explores
			for _, m := range named.FindAllStringSubmatch(b[1], -1) {
				if _, ok := c.Characters[m[1]]; !ok {
					c.Characters[m[1]] = c.Characters["chensun"]
				}
			}
			c.Stages = append(c.Stages, st)
			require.NoError(t, c.Validate(), b[1])
		})
	}
}
