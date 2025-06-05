package main

import (
	"os"
	"path"
	"testing"

	"github.com/adrg/xdg"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeDriveName(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"My Drive", "My_Drive"},
		{"My/Drive", "My_Drive"},
		{"My\\Drive", "My_Drive"},
		{"My:Drive", "My_Drive"},
		{"My?Drive", "My_Drive"},
		{"My*Drive", "My_Drive"},
		{"My\"Drive", "My_Drive"},
		{"My<Drive", "My_Drive"},
		{"My>Drive", "My_Drive"},
		{"My|Drive", "My_Drive"},
		{"My&Drive", "My_Drive"},
	} {
		result := sanitizeDriveName(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestUnitNameToDriveName(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"rclone@mydrive.service", "mydrive"},
		{"rclone@my-drive-1.service", "my-drive-1"},
		{"rclone@my_drive_xyz.service", "my_drive_xyz"},
	} {
		result := unitNameToDriveName(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestDriveNameToUnitName(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		{"mydrive", "rclone@mydrive.service"},
		{"my-drive-1", "rclone@my-drive-1.service"},
		{"my_drive_xyz", "rclone@my_drive_xyz.service"},
	} {
		result := driveNameToUnitName(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestStatusesToServiceStatuses(t *testing.T) {
	for _, test := range []struct {
		input    []dbus.UnitStatus
		expected []serviceStatus
	}{
		{
			input: []dbus.UnitStatus{
				{
					Name:        "rclone@test.service",
					Description: "Test Service",
					ActiveState: "active",
				},
				{
					Name:        "rclone@my_drive.service",
					Description: "My Drive Service",
					ActiveState: "inactive",
				},
			},
			expected: []serviceStatus{
				{
					Name:      "test",
					Status:    "active",
					MountPath: path.Join(xdg.Home, "google", "test"),
				},
				{
					Name:      "my_drive",
					Status:    "inactive",
					MountPath: path.Join(xdg.Home, "google", "my_drive"),
				},
			},
		},
	} {
		result := statusesToServiceStatuses(test.input)
		assert.Equal(t, test.expected, result)
	}

}

func TestEnsureFolderExists(t *testing.T) {
	dir := t.TempDir()
	err := os.Mkdir(path.Join(dir, "test"), 0755)
	assert.NoError(t, err)
	err = ensureFolderExists(path.Join(dir, "test"))
	assert.NoError(t, err)
}

func TestIsDir(t *testing.T) {
	dir := t.TempDir()
	err := os.Mkdir(path.Join(dir, "test"), 0755)
	assert.NoError(t, err)

	isd := isDir(path.Join(dir, "test"))
	assert.True(t, isd)

	isd = isDir(path.Join(dir, "nonexistent"))
	assert.False(t, isd)
}
