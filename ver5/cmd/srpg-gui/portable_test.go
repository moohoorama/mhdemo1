//go:build portable

package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"srpg/internal/content"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnpackPortable(t *testing.T) {
	makeArchive := func(t *testing.T, name string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "game.exe")
		f, err := os.Create(path)
		require.NoError(t, err)
		_, err = f.WriteString("executable prefix")
		require.NoError(t, err)
		z := zip.NewWriter(f)
		w, err := z.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte("data"))
		require.NoError(t, err)
		require.NoError(t, z.Close())
		require.NoError(t, f.Close())
		return path
	}
	t.Run("extract and reuse", func(t *testing.T) {
		exe, cache := makeArchive(t, "assets/data/test.json"), t.TempDir()
		root, err := unpackPortable(exe, cache)
		require.NoError(t, err)
		b, err := os.ReadFile(filepath.Join(root, "assets/data/test.json"))
		require.NoError(t, err)
		assert.Equal(t, "data", string(b))
		again, err := unpackPortable(exe, cache)
		require.NoError(t, err)
		assert.Equal(t, root, again)
	})
	t.Run("reject traversal", func(t *testing.T) {
		_, err := unpackPortable(makeArchive(t, "assets/../../escape"), t.TempDir())
		assert.Error(t, err)
	})
}

func TestPortableBuiltBundle(t *testing.T) {
	exe := os.Getenv("SRPG_PORTABLE_EXE")
	if exe == "" {
		t.Skip("no built portable executable")
	}
	root, err := unpackPortable(exe, t.TempDir())
	require.NoError(t, err)
	_, err = content.Load(filepath.Join(root, "assets", "data"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "assets", "fonts", "NotoSansKR.ttf"))
	require.NoError(t, err)
}
