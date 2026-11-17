package gui

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/autoplay"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestAutoplayRecording(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	var trace bytes.Buffer
	p := &autoplay.RulePlayer{D: d, G: autoplay.DefaultGenome()}
	_, err = autoplay.Run(d, p, autoplay.Options{Seed: 1, Difficulty: "hard", Stages: 1, Trace: &trace})
	require.NoError(t, err)
	rec, err := readAutoplayRecording(bytes.NewReader(trace.Bytes()), "B01")
	require.NoError(t, err)
	t.Run("commands reproduce the recorded state and events", func(t *testing.T) {
		e, err := core.Restore(d, rec.snapshot)
		require.NoError(t, err)
		_, err = e.Apply(core.Command{Kind: "start"})
		require.NoError(t, err)
		for _, step := range rec.steps {
			_, err = applyAutoplayRecord(e, step)
			require.NoError(t, err)
		}
		assert.Equal(t, "victory", e.Observe().Result)
	})
	t.Run("later stages do not replace the selected attempt", func(t *testing.T) {
		var multiple bytes.Buffer
		multiple.Write(trace.Bytes())
		enc := json.NewEncoder(&multiple)
		require.NoError(t, enc.Encode(autoplayRecord{Stage: "B01", Try: 2, Snapshot: &rec.snapshot}))
		for _, step := range rec.steps {
			require.NoError(t, enc.Encode(step))
		}
		require.NoError(t, enc.Encode(autoplayRecord{Stage: "B02", Try: 1, Snapshot: &rec.snapshot}))
		multiple.WriteString("unfinished later stage")
		got, err := readAutoplayRecording(&multiple, "B01")
		require.NoError(t, err)
		assert.Equal(t, 2, got.try)
		assert.Len(t, got.steps, len(rec.steps))
	})
	t.Run("missing or incomplete battles are rejected", func(t *testing.T) {
		_, err := readAutoplayRecording(bytes.NewReader(trace.Bytes()), "B20")
		require.ErrorContains(t, err, "no complete attempt")
		var incomplete bytes.Buffer
		enc := json.NewEncoder(&incomplete)
		require.NoError(t, enc.Encode(autoplayRecord{Stage: "B01", Try: 1, Snapshot: &rec.snapshot}))
		for _, step := range rec.steps[:len(rec.steps)-1] {
			require.NoError(t, enc.Encode(step))
		}
		_, err = readAutoplayRecording(&incomplete, "B01")
		require.ErrorContains(t, err, "no complete attempt")
	})
	t.Run("changed results are rejected", func(t *testing.T) {
		e, err := core.Restore(d, rec.snapshot)
		require.NoError(t, err)
		_, err = e.Apply(core.Command{Kind: "start"})
		require.NoError(t, err)
		step := rec.steps[0]
		step.Round++
		_, err = applyAutoplayRecord(e, step)
		require.ErrorContains(t, err, "trace differs")
	})
}
