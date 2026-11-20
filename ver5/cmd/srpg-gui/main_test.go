package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourcePath(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	exeDir := filepath.Join(root, "app", "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(work, 0755))
	require.NoError(t, os.MkdirAll(exeDir, 0755))
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(work))
	t.Cleanup(func() { assert.NoError(t, os.Chdir(old)) })
	path := filepath.Join("assets", "data")
	exe := filepath.Join(exeDir, "srpg.exe")
	installed := filepath.Join(exeDir, path)
	bundle := filepath.Join(root, "app", "Contents", "Resources", path)

	t.Run("missing", func(t *testing.T) { assert.Equal(t, path, resourcePath(path, exe)) })
	t.Run("mac bundle", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(bundle, 0755))
		assert.Equal(t, bundle, resourcePath(path, exe))
	})
	t.Run("beside executable", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(installed, 0755))
		assert.Equal(t, installed, resourcePath(path, exe))
	})
	t.Run("current directory takes precedence", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(path, 0755))
		assert.Equal(t, path, resourcePath(path, exe))
	})
	t.Run("missing absolute path stays explicit", func(t *testing.T) {
		missing := filepath.Join(root, "missing")
		assert.Equal(t, missing, resourcePath(missing, exe))
	})
}
