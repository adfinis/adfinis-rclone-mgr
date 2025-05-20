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

	defer os.Chmod(noPermDir, 0755) // nolint:errcheck

	err = moveFile(src, dest)
	assert.Error(t, err)
}

func TestShouldTriggerError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "copy error with insufficientParentPermissions",
			message: "ERROR : test: Failed to copy: googleapi: Error 403: Insufficient permissions for the specified parent., insufficientParentPermissions",
			want:    false,
		},
		{
			name:    "vfs cache upload error with insufficientParentPermissions",
			message: "ERROR : test: vfs cache: failed to upload try #3, will retry in 40s: vfs cache: failed to transfer file from cache to remote: googleapi: Error 403: Insufficient permissions for the specified parent., insufficientParentPermissions",
			want:    true,
		},
		{
			name:    "make directory error with insufficientParentPermissions",
			message: "ERROR : IO error: failed to make directory: googleapi: Error 403: Insufficient permissions for the specified parent., insufficientParentPermissions",
			want:    false,
		},
		{
			name:    "mkdir failed to create directory with insufficientParentPermissions",
			message: "ERROR : /: Dir.Mkdir failed to create directory: failed to make directory: googleapi: Error 403: Insufficient permissions for the specified parent., insufficientParentPermissions",
			want:    false,
		},
		{
			name:    "vfs cache upload error without insufficientParentPermissions",
			message: "ERROR : test: vfs cache: failed to upload try #3, will retry in 40s: vfs cache: failed to transfer file from cache to remote: some other error",
			want:    false,
		},
		{
			name:    "random error",
			message: "ERROR : something else",
			want:    false,
		},
	}

	for _, tt := range tests {
		entry := LogEntry{Message: tt.message}
		got := shouldTriggerError(entry)
		assert.Equalf(t, tt.want, got, "%s: shouldTriggerError() = %v, want %v", tt.name, got, tt.want)
	}
}
