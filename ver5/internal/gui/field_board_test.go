package gui

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every scene map drawn as a picture has its diorama edge baked from its present contents
// (python3 tools/scene_maps/diorama.py <ID>; go run ./cmd/_walkmap -fmt <ID> runs it).
func TestBoardsBaked(t *testing.T) {
	gfx := "../../assets/graphics"
	raw, err := os.ReadFile(filepath.Join(gfx, "index.json"))
	require.NoError(t, err)
	var index struct{ Maps []string }
	require.NoError(t, json.Unmarshal(raw, &index))
	for _, path := range index.Maps {
		id := strings.TrimSuffix(filepath.Base(path), ".json")
		if !strings.HasPrefix(id, "S_") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(gfx, path))
		require.NoError(t, err)
		var head struct {
			Format int
			Native bool
			Visual []int
		}
		require.NoError(t, json.Unmarshal(data, &head))
		if head.Format != 2 && (!head.Native || len(head.Visual) == 0) {
			continue // an old map drawn at twice the tile size: no board
		}
		sum := sha1.Sum(data)
		board, err := os.ReadFile(filepath.Join(gfx, "maps", "board", id+".json"))
		if !assert.NoError(t, err, "%s has no board: python3 tools/scene_maps/diorama.py %s", id, id) {
			continue
		}
		var b struct{ Source string }
		require.NoError(t, json.Unmarshal(board, &b))
		assert.Equal(t, hex.EncodeToString(sum[:]), b.Source, "%s changed since its board was baked: python3 tools/scene_maps/diorama.py %s", id, id)
	}
}

// A format 1 scene map leaves out what would stand across its board's front edges: S_zhuo's walls stop at the edge.
func TestTrimToBoard(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	art := d.g.assets.Maps["S_zhuo"]
	require.NotNil(t, art.Board)
	u1 := float64(art.Visual[0] + art.Visual[2])
	walls := 0
	for _, p := range art.Props {
		if p.Name == "swall_u" {
			walls++
			assert.LessOrEqual(t, 2*p.U+1.5, u1, "swall_u at %g", p.U)
		}
	}
	assert.Positive(t, walls)
	assert.Len(t, art.Board.Cuts, 2, "the u wall's and the v wall's cut")
}
