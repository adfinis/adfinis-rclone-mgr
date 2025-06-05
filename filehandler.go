package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func copy(cmd *cobra.Command, args []string) {
	// last arg is the destination directory
	dest := args[len(args)-1]
	copyFile(args[:len(args)-1], dest)
}

// isSubdir checks if sub is a subdirectory (or the same) as root.
func isSubdir(root, sub string) (bool, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	absSub, err := filepath.Abs(sub)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(absRoot, absSub)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false, nil
	}
	return true, nil
}

// toRclonePath converts an absolute path under the Google root to rclone remote:path format.
func toRclonePath(root, abs string) (remote, subpath string, err error) {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", "", err
	}
	split := strings.Split(rel, string(os.PathSeparator))
	if len(split) < 1 {
		return "", "", fmt.Errorf("invalid path: %q", abs)
	}
	remote = split[0]
	subpath = path.Join(strings.Join(split[1:], string(os.PathSeparator)))
	// assume something went horribly wrong if any of those start with ".."
	if strings.HasPrefix(subpath, "..") || strings.HasPrefix(remote, "..") {
		return "", "", fmt.Errorf("invalid path: %q", abs)
	}
	return remote, subpath, nil
}

// patchDestPath modifies the destination path to ensure the name of the source folder is added to the destination path.
// otherwise rclone would copy the files into the destination directory without creating a subdirectory.
func patchDestPath(src, dest string) string {
	if !strings.HasSuffix(dest, string(os.PathSeparator)) {
		dest += string(os.PathSeparator)
	}
	srcBase := filepath.Base(src)
	return filepath.Join(dest, srcBase)
}

func showZenityError(msg string) {
	if err := exec.Command("zenity", "--error", "--text", msg).Run(); err != nil {
		log.Println("Failed to show error dialog:", err)
	}
}

func selectDestination() (string, error) {
	zenityCmd := exec.Command("zenity", "--file-selection", "--directory", "--title=Select destination directory for copy")
	zenityOut, err := zenityCmd.Output()
	if err != nil {
		showZenityError("Copy cancelled or failed to select destination")
		return "", err
	}
	destDir := strings.TrimSpace(string(zenityOut))
	log.Println("Selected destination directory:", destDir)
	return destDir, nil
}

func selectDestAndCopy(srcPaths []string) {
	destDir, err := selectDestination()
	if err != nil {
		log.Println("Failed to select destination:", err)
		return
	}

	if destDir == "" {
		showZenityError("No destination directory selected")
		return
	}

	copyFile(srcPaths, destDir)
}

func copyFile(srcPaths []string, destDir string) {
	absGoogleRoot, err := filepath.Abs(getGooglePath())
	if err != nil {
		showZenityError("Failed to resolve Google Drive root")
		return
	}
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		showZenityError("Failed to resolve destination directory")
		return
	}
	// Check if absDestDir is under absGoogleRoot
	isSub, err := isSubdir(absGoogleRoot, absDestDir)
	if err != nil || !isSub {
		showZenityError("Destination must be inside your Google Drive mount")
		return
	}

	for _, src := range srcPaths {
		isSub, err := isSubdir(absGoogleRoot, src)
		if err != nil || !isSub {
			showZenityError("Source must be inside your Google Drive mount")
			return
		}
		srcDriveName, srcPath, err := toRclonePath(absGoogleRoot, src)
		if err != nil {
			showZenityError("Failed to parse source path")
			return
		}
		srcRclone := fmt.Sprintf("%s:%s", srcDriveName, srcPath)
		destDriveName, destPath, err := toRclonePath(absGoogleRoot, absDestDir)
		if err != nil {
			showZenityError("Failed to parse destination path")
			return
		}
		destRclone := fmt.Sprintf("%s:%s", destDriveName, destPath)

		if isDir(src) {
			destRclone = patchDestPath(srcPath, destRclone)
		}
		log.Printf("Copying from %s to %s", srcRclone, destRclone)

		cmd := exec.Command("rclone", "copy", "--drive-server-side-across-configs", srcRclone, destRclone, "-v")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			showZenityError("rclone copy failed: " + err.Error())
			return
		}
	}
	if err := exec.Command("zenity", "--info", "--text", "File(s) copied successfully").Run(); err != nil {
		log.Println("Failed to show success dialog:", err)
	}
}
