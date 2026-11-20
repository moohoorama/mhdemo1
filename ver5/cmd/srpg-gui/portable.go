//go:build portable

package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func portableResources(content, font *string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	root, err := unpackPortable(exe, filepath.Join(cache, "srpg-portable"))
	if err != nil {
		return err
	}
	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["content"] {
		*content = filepath.Join(root, "assets", "data")
	}
	if !set["font"] {
		*font = filepath.Join(root, "assets", "fonts", "NotoSansKR.ttf")
	}
	return nil
}

func unpackPortable(exe, cache string) (string, error) {
	f, err := os.Open(exe)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	id := hex.EncodeToString(h.Sum(nil))
	dest := filepath.Join(cache, id)
	if _, err = os.Stat(filepath.Join(dest, ".complete")); err == nil {
		return dest, nil
	}
	z, err := zip.NewReader(f, info.Size())
	if err != nil {
		return "", fmt.Errorf("open bundled assets: %w", err)
	}
	if err = os.MkdirAll(cache, 0700); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(cache, ".extract-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	for _, entry := range z.File {
		name := filepath.FromSlash(entry.Name)
		if !filepath.IsLocal(name) || !strings.HasPrefix(entry.Name, "assets/") || !entry.Mode().IsRegular() {
			return "", fmt.Errorf("invalid bundled asset %q", entry.Name)
		}
		target := filepath.Join(tmp, name)
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		if err = extractPortable(entry, target); err != nil {
			return "", err
		}
	}
	if err = os.WriteFile(filepath.Join(tmp, ".complete"), nil, 0600); err != nil {
		return "", err
	}
	if err = os.Rename(tmp, dest); err != nil {
		if _, check := os.Stat(filepath.Join(dest, ".complete")); check != nil {
			return "", err
		}
	}
	return dest, nil
}

func extractPortable(entry *zip.File, target string) error {
	in, err := entry.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
