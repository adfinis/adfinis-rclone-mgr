package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSubdir(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(tmp+"/root/subdir", 0o755) // nolint:errcheck
	os.MkdirAll(tmp+"/other", 0o755)       // nolint:errcheck

	tests := []struct {
		name string
		root string
		sub  string
		want bool
	}{
		{"self", tmp + "/root", tmp + "/root", true},
		{"direct subdir", tmp + "/root", tmp + "/root/subdir", true},
		{"not subdir", tmp + "/root", tmp + "/other", false},
		{"parent", tmp + "/root/subdir", tmp + "/root", false},
		{"parent of root", tmp + "/root", tmp, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isSubdir(tt.root, tt.sub)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got, "isSubdir(%q, %q)", tt.root, tt.sub)
		})
	}
}

func TestToRclonePath(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(tmp+"/google/drive1/folder", 0o755) // nolint:errcheck
	f, _ := os.Create(tmp + "/google/drive1/folder/file.txt")
	f.Close() // nolint:errcheck
	f2, _ := os.Create(tmp + "/google/drive1/file.txt")
	f2.Close() // nolint:errcheck

	tests := []struct {
		name     string
		root     string
		abs      string
		wantRem  string
		wantPath string
		wantErr  bool
	}{
		{"file in drive root", tmp + "/google", tmp + "/google/drive1/file.txt", "drive1", "file.txt", false},
		{"file in subdir", tmp + "/google", tmp + "/google/drive1/folder/file.txt", "drive1", "folder/file.txt", false},
		{"drive root itself", tmp + "/google", tmp + "/google/drive1", "drive1", "", false},
		{"not under root", tmp + "/google", tmp, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rem, sub, err := toRclonePath(tt.root, tt.abs)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRem, rem)
				assert.Equal(t, tt.wantPath, sub)
			}
		})
	}
}

func TestPatchDestPath(t *testing.T) {
	tests := []struct {
		name string
		src  string
		dest string
		want string
	}{
		{"dest with trailing slash", "/foo/bar.txt", "/dest/", "/dest/bar.txt"},
		{"dest without trailing slash", "/foo/bar.txt", "/dest", "/dest/bar.txt"},
		{"src is dir", "/foo/dir", "/dest", "/dest/dir"},
		{"src is file, dest is root", "/foo/file.txt", "/", "/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := patchDestPath(tt.src, tt.dest)
			assert.Equal(t, tt.want, got, "patchDestPath(%q, %q)", tt.src, tt.dest)
		})
	}
}
