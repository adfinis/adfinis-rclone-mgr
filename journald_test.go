package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMoveFile_Success(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dest := filepath.Join(dir, "dest.txt")
	content := []byte("hello world")

	err := os.WriteFile(src, content, 0644)
	assert.NoError(t, err)

	err = moveFile(src, dest)
	assert.NoError(t, err)

	// Source should not exist
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))

	// Dest should exist and have correct content
	got, err := os.ReadFile(dest)
	assert.NoError(t, err)
	assert.Equal(t, content, got)

	// Permissions should be preserved
	info, err := os.Stat(dest)
	assert.NoError(t, err)
	assert.Equal(t, fs.FileMode(0644), info.Mode().Perm())
}

func TestMoveFile_OverwriteExistingFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dest := filepath.Join(dir, "dest.txt")

	err := os.WriteFile(src, []byte("src data"), 0600)
	assert.NoError(t, err)
	err = os.WriteFile(dest, []byte("old dest data"), 0644)
	assert.NoError(t, err)

	err = moveFile(src, dest)
	assert.NoError(t, err)

	// Source should not exist
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))

	// Dest should have new content
	got, err := os.ReadFile(dest)
	assert.NoError(t, err)
	assert.Equal(t, []byte("src data"), got)
}

func TestMoveFile_DestIsDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	destDir := filepath.Join(dir, "destdir")

	err := os.WriteFile(src, []byte("data"), 0644)
	assert.NoError(t, err)
	err = os.Mkdir(destDir, 0755)
	assert.NoError(t, err)

	err = moveFile(src, destDir)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "is a directory"))
}

func TestMoveFile_SrcDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "no_such_file.txt")
	dest := filepath.Join(dir, "dest.txt")

	err := moveFile(src, dest)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestMoveFile_DestNoWritePermission(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	noPermDir := filepath.Join(dir, "no_perm")
	dest := filepath.Join(noPermDir, "dest.txt")

	err := os.WriteFile(src, []byte("data"), 0644)
	assert.NoError(t, err)
	err = os.Mkdir(noPermDir, 0500)
	assert.NoError(t, err)

	// Remove write permission from directory
	defer os.Chmod(noPermDir, 0755) // restore for cleanup

	err = moveFile(src, dest)
	assert.Error(t, err)
}
