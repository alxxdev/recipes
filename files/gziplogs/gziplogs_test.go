package main

import (
	"io/fs"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func touch(t *testing.T, path string, mtime time.Time) {
	file, err := os.Create(path)
	require.NoError(t, err)
	file.Close()

	err = os.Chtimes(path, mtime, mtime)
	require.NoError(t, err)
}

func TestCompressFiles(t *testing.T) {
	dir := t.TempDir()
	touch(t, path.Join(dir, "a.log"), time.Now())
	touch(t, path.Join(dir, "b.log"), time.Now().Add(-2*time.Hour))
	touch(t, path.Join(dir, "c.txt"), time.Now().Add(-2*time.Hour))
	time.Sleep(time.Second) // sync files

	err := compressFiles(dir, time.Hour)
	require.NoError(t, err)

	files, err := fs.Glob(os.DirFS(dir), "*")
	require.NoError(t, err)
	expected := []string{"a.log", "b.log.gz", "c.txt"}
	require.Equal(t, expected, files)
}
