package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/adrg/xdg"
)

// dont ask...
func sanitizeDriveName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	name = strings.ReplaceAll(name, "'", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "%", "_")
	return name
}

func ensureFolderExists(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func unitNameToDriveName(name string) string {
	name = strings.Split(name, "@")[1]
	name = strings.Split(name, ".service")[0]
	return name
}

func driveNameToUnitName(name string) string {
	return fmt.Sprintf("rclone@%s.service", name)
}

func getGooglePath() string {
	return path.Join(xdg.Home, "google")
}

func getDriveDataPath(name string) string {
	return path.Join(getGooglePath(), name)
}

func getDriveCachePath(name string) string {
	return path.Join(xdg.CacheHome, "google", name)
}

func fileNameToPath(driveName, fileName string) string {
	return path.Join(getDriveDataPath(driveName), fileName)
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		log.Println(err)
		return false
	}
	return info.IsDir()
}
